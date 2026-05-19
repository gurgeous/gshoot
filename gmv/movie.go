package gmv

// Movie loads the embedded GMV sprite sheet.
// A GMV is a paletted PNG plus tiny tEXt metadata; decoded movies keep palette indexes for cheap terminal sampling.

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
)

// builtinGMV is the bundled first-run animation.
//
//go:embed assets/bluelight.gmv
var builtinGMV []byte

// movie is a decoded paletted GMV sprite sheet.
type movie struct {
	Size       size            // pixel size of one frame
	Frames     int             // number of frames
	pix        []uint8         // raw paletted sprite-sheet pixels
	stride     int             // image row stride
	bounds     image.Rectangle // decoded sprite-sheet bounds
	palette    []color.RGBA    // decoded source colors
	dimPalette []color.RGBA    // source colors dimmed for card backgrounds
}

// loadMovie decodes and validates the embedded animation.
func loadMovie() (*movie, error) {
	size, frames, err := readMetadata(builtinGMV)
	if err != nil {
		return nil, err
	}

	img, err := png.Decode(bytes.NewReader(builtinGMV))
	if err != nil {
		return nil, fmt.Errorf("decode gmv png: %w", err)
	}
	bounds := img.Bounds()

	paletted, ok := img.(*image.Paletted)
	if !ok {
		return nil, fmt.Errorf("gmv png must be paletted, got %T", img)
	}
	if err := gmvSanity(size, frames, bounds); err != nil {
		return nil, err
	}

	// parse palette
	palette := make([]color.RGBA, len(paletted.Palette))
	dimPalette := make([]color.RGBA, len(paletted.Palette))
	for i, c := range paletted.Palette {
		rgba := rgba(c)
		palette[i] = rgba
		dimPalette[i] = dim(rgba, 0.35)
	}

	return &movie{
		Size:       size,
		Frames:     frames,
		pix:        paletted.Pix,
		stride:     paletted.Stride,
		bounds:     bounds,
		palette:    palette,
		dimPalette: dimPalette,
	}, nil
}

// frameOrigin returns the sprite-sheet origin for a frame index.
func (m *movie) frameOrigin(fr int) point {
	columns := m.bounds.Dx() / m.Size.X
	return pt((fr%columns)*m.Size.X, (fr/columns)*m.Size.Y)
}

// sample returns the palette index for a drawn pixel in one frame.
func (m *movie) sample(frameOrigin point, draw rect, p point) uint8 {
	srcX := frameOrigin.X + p.X*m.Size.X/draw.Dx()
	srcY := frameOrigin.Y + p.Y*m.Size.Y/draw.Dy()
	rowOffset := (srcY-m.bounds.Min.Y)*m.stride - m.bounds.Min.X
	return m.pix[rowOffset+srcX]
}

// readMetadata extracts the GMV tEXt chunk from a PNG.
func readMetadata(data []byte) (size, int, error) {
	const signature = "\x89PNG\r\n\x1a\n"
	if !bytes.HasPrefix(data, []byte(signature)) {
		return size{}, 0, fmt.Errorf("gmv asset is not a png")
	}

	pos := len(signature)
	for pos+8 <= len(data) {
		length := int(binary.BigEndian.Uint32(data[pos : pos+4]))
		chunkType := string(data[pos+4 : pos+8])
		start := pos + 8
		end := start + length
		if end+4 > len(data) {
			return size{}, 0, fmt.Errorf("truncated gmv png chunk")
		}

		chunk := data[start:end]
		if chunkType == "tEXt" && bytes.HasPrefix(chunk, []byte("gmv\x00")) {
			var meta struct {
				Width  int `json:"w"`
				Height int `json:"h"`
				Frames int `json:"n"`
			}
			if err := json.Unmarshal(chunk[4:], &meta); err != nil {
				return size{}, 0, fmt.Errorf("decode gmv metadata: %w", err)
			}
			return sz(meta.Width, meta.Height), meta.Frames, nil
		}

		pos = end + 4
	}

	return size{}, 0, fmt.Errorf("missing gmv metadata")
}

// gmvSanity checks sprite-sheet dimensions against frame geometry.
func gmvSanity(frameSize size, frames int, bounds image.Rectangle) error {
	switch {
	case frameSize.X <= 0:
		return fmt.Errorf("invalid gmv frame width %d", frameSize.X)
	case frameSize.Y <= 0:
		return fmt.Errorf("invalid gmv frame height %d", frameSize.Y)
	case frameSize.Y%2 != 0:
		return fmt.Errorf("gmv frame height must be even")
	case frames <= 0:
		return fmt.Errorf("invalid gmv frame count %d", frames)
	case bounds.Dx()%frameSize.X != 0:
		return fmt.Errorf("gmv sheet width %d is not divisible by frame width %d", bounds.Dx(), frameSize.X)
	case bounds.Dy()%frameSize.Y != 0:
		return fmt.Errorf("gmv sheet height %d is not divisible by frame height %d", bounds.Dy(), frameSize.Y)
	}

	cols := bounds.Dx() / frameSize.X
	rows := bounds.Dy() / frameSize.Y
	if cols*rows < frames {
		return fmt.Errorf("gmv sheet has %d slots for %d frames", cols*rows, frames)
	}
	return nil
}
