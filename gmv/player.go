package gmv

// Player is the outer GMV playback loop.
// It owns raw mode, key exit, resize handling, pingpong timing, and stdout writes.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"time"

	lipgloss "charm.land/lipgloss/v2"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
)

//
// entrypoints
//

// Play shows the built-in GMV behind a centered ANSI card until any key is pressed.
func Play(ctx context.Context, card string, showStats bool) error {
	p, err := NewPlayer(card, showStats)
	if err != nil {
		return err
	}
	return p.Play(ctx)
}

// NewPlayer initializes the built-in GMV player.
func NewPlayer(card string, showStats bool) (*Player, error) {
	movie, err := loadMovie()
	if err != nil {
		return nil, err
	}

	cfg := newConfig()
	renderedCard := downsample(card, cfg.colorProfile())
	termSize := util.TerminalSize(movie.Size)
	return &Player{
		movie:     movie,
		cfg:       cfg,
		showStats: showStats,
		cardText:  card,
		card:      newCard(renderedCard),
		renderer:  newRenderer(movie, cfg, termSize),
		start:     time.Now(),
	}, nil
}

// Play runs the animation until context cancel or any key is pressed.
func (p *Player) Play(ctx context.Context) error {
	if !util.IsTty(os.Stdout) {
		if p.cardText != "" {
			fmt.Fprintln(os.Stdout, p.cardText)
		}
		return nil
	}

	closeRaw, err := util.EnterRawMode()
	if err != nil {
		return err
	}
	defer closeRaw()

	key, closeKey, err := keyWatchInit()
	if err != nil {
		return err
	}
	defer func() {
		p.key = nil
		_ = closeKey()
	}()

	p.key = key
	p.start = time.Now()
	p.renderer.reset()
	return p.play(ctx)
}

// Demo plays the built-in GMV with sample first-run text.
func Demo(ctx context.Context) error {
	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#60a5fa")).
		Padding(1, 4).
		Render(strings.Join([]string{
			ux.Brand.Render("welcome to gshoot"),
			"",
			ux.Info.Render("auth is not set up yet"),
			ux.Subtle.Render("press any key to continue"),
		}, "\n"))
	p, err := NewPlayer(card, true)
	if err != nil {
		return err
	}
	return p.Play(ctx)
}

//
// helpers
//

// errStopped marks a keypress-driven playback stop.
var errStopped = errors.New("gmv stopped")

// Player keeps all mutable state for the animation loop.
type Player struct {
	movie     *movie          // loaded source movie
	cfg       config          // resolved playback configuration from env
	showStats bool            // whether to show stats
	cardText  string          // plain fallback shown outside a TTY
	card      card            // parsed overlay image
	frame     bytes.Buffer    // rendered terminal byte buffer
	renderer  *renderer       // image composition and terminal diffing
	stats     statsTracker    // rolling demo playback metrics
	start     time.Time       // playback clock anchor
	key       <-chan struct{} // closed when the user presses a key
}

// play renders movie frames until context cancel or keypress.
func (p *Player) play(ctx context.Context) error {
	ticker := time.NewTicker(p.cfg.frameDelay())
	defer ticker.Stop()

	for {
		if p.stopped(ctx) {
			return nil
		}
		if err := p.renderFrame(ctx); err != nil {
			return err
		}
		p.waitForFrame(ctx, ticker.C)
	}
}

// renderFrame renders and writes one animation frame.
func (p *Player) renderFrame(ctx context.Context) error {
	p.refreshSize()

	fr := p.frameIndex(time.Now())
	var stats statsTracker
	if p.showStats {
		stats = p.stats
	}
	p.renderer.render(&p.frame, fr, p.card, stats)
	data := p.frame.Bytes()
	if err := p.writeAll(ctx, data); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, errStopped) {
			return nil
		}
		return err
	}
	if p.showStats {
		p.stats.Observe(len(data), time.Now())
	}
	return nil
}

// frameIndex returns the pingpong frame index for the playback clock.
func (p *Player) frameIndex(now time.Time) int {
	fr := int(now.Sub(p.start).Seconds() * p.cfg.fps)
	return pingpong(fr, p.movie.Frames)
}

// pingpong maps frame counts into forward-then-reverse playback.
func pingpong(fr, frames int) int {
	if frames <= 1 {
		return 0
	}

	period := frames*2 - 2
	fr %= period
	if fr < frames {
		return fr
	}
	return period - fr
}

// refreshSize rebuilds layout when the terminal size changes.
func (p *Player) refreshSize() {
	if !p.renderer.resize(util.TerminalSize(p.movie.Size)) {
		return
	}
	fmt.Fprint(os.Stdout, xansi.EraseEntireScreen)
}

// stopped reports whether playback should stop immediately.
func (p *Player) stopped(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	case <-p.key:
		return true
	default:
		return false
	}
}

// waitForFrame waits for the next tick or an early stop signal.
func (p *Player) waitForFrame(ctx context.Context, tick <-chan time.Time) {
	select {
	case <-ctx.Done():
	case <-p.key:
	case <-tick:
	}
}

// writeAll writes all bytes, retrying temporary TTY failures.
func (p *Player) writeAll(ctx context.Context, data []byte) error {
	for len(data) > 0 {
		n, err := os.Stdout.Write(data)
		if n > 0 {
			data = data[n:]
		}
		if len(data) == 0 {
			return nil
		}
		if err == nil {
			if n == 0 {
				return io.ErrShortWrite
			}
			continue
		}
		if !canRetry(err) {
			return err
		}
		if err := p.waitForWrite(ctx); err != nil {
			return err
		}
	}
	return nil
}

// canRetry reports temporary terminal write errors.
func canRetry(err error) bool {
	return errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EINTR)
}

// waitForWrite pauses briefly before retrying a write.
func (p *Player) waitForWrite(ctx context.Context) error {
	timer := time.NewTimer(time.Millisecond)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.key:
		return errStopped
	case <-timer.C:
		return nil
	}
}

//
// wait for key
//

// keyWatchInit starts a blocking keypress reader on a duplicate stdin handle.
func keyWatchInit() (<-chan struct{}, func() error, error) {
	fd, err := syscall.Dup(int(os.Stdin.Fd()))
	if err != nil {
		return nil, nil, err
	}
	in := os.NewFile(uintptr(fd), os.Stdin.Name())
	if in == nil {
		_ = syscall.Close(fd)
		return nil, nil, fmt.Errorf("dup stdin")
	}

	key := make(chan struct{})
	go keyWatch(in, key)
	return key, in.Close, nil
}

// keyWatch closes key after the first input byte.
func keyWatch(in *os.File, key chan<- struct{}) {
	var buf [1]byte
	_, _ = in.Read(buf[:])
	close(key)
}
