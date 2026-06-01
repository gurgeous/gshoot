package gmv

import (
	"fmt"
	"time"
)

//
// Stats tracks compact demo playback metrics. The renderer treats stats as a
// tiny overlay image.
//

type statsTracker struct {
	last   time.Time
	bytes  int
	frames int
	line   string
}

// Observe records one rendered frame for the stats overlay.
func (stats *statsTracker) Observe(bytes int, now time.Time) {
	if stats.last.IsZero() {
		stats.last = now
	}
	stats.bytes += bytes
	stats.frames++

	elapsed := now.Sub(stats.last)
	if elapsed < 250*time.Millisecond {
		return
	}

	fps := float64(stats.frames) / elapsed.Seconds()
	bytesPerFrame := float64(stats.bytes) / float64(stats.frames)
	stats.line = fmt.Sprintf("%.0ffps %.0fk/f", fps, bytesPerFrame/1024)
	stats.last = now
	stats.bytes = 0
	stats.frames = 0
}

// image renders stats as a terminal image.
func (stats statsTracker) image(style string, bg paletteColor) timage {
	if stats.line == "" {
		return timage{}
	}

	str := " " + stats.line + " "
	img := timage{}
	img.resize(sz(len(str), 1))
	for col := range len(str) {
		img.set(pt(col, 0), tpixel{
			Color: bg,
			Style: style,
			Ch:    str[col : col+1],
		})
	}
	return img
}
