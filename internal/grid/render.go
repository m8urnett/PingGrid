package grid

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"
	"strings"

	"github.com/m8urnett/PingGrid/internal/scanner"
)

// Default grid dimensions and palette matching .notes/grid.png.
const (
	DefaultRows        = 8
	DefaultCols        = 32
	DefaultWidth       = 295
	DefaultHeight      = 77
	DefaultBorderWidth = 1
)

var (
	// Default palette uses strictly the 9 curated colors:
	// Frame/Border: #2c2c2c, Offline: #404e41, Online: #4d86a2, Highlight: #e4f9d4
	DefaultColorOffline   = PaletteMossForest // #404e41
	DefaultColorOnline    = PaletteSlateTeal  // #4d86a2
	DefaultColorHighlight = PaletteMint       // #e4f9d4
	DefaultColorSlow      = PaletteTerracotta // #ab7550
	DefaultColorBorder    = PaletteCharcoal   // #2c2c2c
	DefaultColorFrame     = PaletteCharcoal   // #2c2c2c
)

// GridConfig holds user-configurable options for the grid image.
type GridConfig struct {
	Rows           int
	Cols           int
	Width          int
	Height         int
	BorderWidth    int
	ColorOffline   color.RGBA
	ColorOnline    color.RGBA
	ColorHighlight color.RGBA
	ColorSlow      color.RGBA
	ColorBorder    color.RGBA
	ColorFrame     color.RGBA
}

// DefaultConfig returns the standard GridConfig using the default dark scheme.
func DefaultConfig() GridConfig {
	return GridConfig{
		Rows:           DefaultRows,
		Cols:           DefaultCols,
		Width:          DefaultWidth,
		Height:         DefaultHeight,
		BorderWidth:    DefaultBorderWidth,
		ColorOffline:   DefaultColorOffline,
		ColorOnline:    DefaultColorOnline,
		ColorHighlight: DefaultColorHighlight,
		ColorSlow:      DefaultColorSlow,
		ColorBorder:    DefaultColorBorder,
		ColorFrame:     DefaultColorFrame,
	}
}

// AvailableSchemes returns the list of built-in harmonious color schemes.
func AvailableSchemes() []string {
	return []string{"dark", "light", "earth", "moss", "linen"}
}

// GetScheme returns a GridConfig configured with a built-in scheme.
func GetScheme(name string) (GridConfig, error) {
	cfg := DefaultConfig()
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "dark", "default":
		// Dark theme: Charcoal, Forest Moss, Slate Teal, Soft Mint, Terracotta
		cfg.ColorFrame = PaletteCharcoal
		cfg.ColorBorder = PaletteCharcoal
		cfg.ColorOffline = PaletteMossForest
		cfg.ColorOnline = PaletteSlateTeal
		cfg.ColorHighlight = PaletteMint
		cfg.ColorSlow = PaletteTerracotta
		return cfg, nil

	case "light", "nordic":
		// Light theme: Warm Linen, Light Gray, Off-White, Slate Teal, Terracotta
		cfg.ColorFrame = PaletteWarmLinen
		cfg.ColorBorder = PaletteLightGray
		cfg.ColorOffline = PaletteOffWhite
		cfg.ColorOnline = PaletteSlateTeal
		cfg.ColorHighlight = PaletteTerracotta
		cfg.ColorSlow = PaletteCharcoal
		return cfg, nil

	case "earth", "terracotta":
		// Earth theme: Charcoal, Forest Moss, Sage, Terracotta, Slate Teal
		cfg.ColorFrame = PaletteCharcoal
		cfg.ColorBorder = PaletteCharcoal
		cfg.ColorOffline = PaletteMossForest
		cfg.ColorOnline = PaletteSage
		cfg.ColorHighlight = PaletteTerracotta
		cfg.ColorSlow = PaletteSlateTeal
		return cfg, nil

	case "moss", "forest":
		// Moss theme: Forest Moss frame, Charcoal offline, Slate Teal online, Mint highlight
		cfg.ColorFrame = PaletteMossForest
		cfg.ColorBorder = PaletteCharcoal
		cfg.ColorOffline = PaletteCharcoal
		cfg.ColorOnline = PaletteSlateTeal
		cfg.ColorHighlight = PaletteMint
		cfg.ColorSlow = PaletteTerracotta
		return cfg, nil

	case "linen", "cream":
		// Linen theme: Warm Linen frame, Light Gray border, Off-White offline, Moss online, Slate Teal highlight
		cfg.ColorFrame = PaletteWarmLinen
		cfg.ColorBorder = PaletteLightGray
		cfg.ColorOffline = PaletteOffWhite
		cfg.ColorOnline = PaletteMossForest
		cfg.ColorHighlight = PaletteSlateTeal
		cfg.ColorSlow = PaletteTerracotta
		return cfg, nil

	default:
		return cfg, fmt.Errorf("unknown scheme %q: available schemes are: %s", name, strings.Join(AvailableSchemes(), ", "))
	}
}

// Render generates the RGBA image from the given config and host results.
func Render(cfg GridConfig, results []scanner.HostResult) *image.RGBA {
	if cfg.Rows <= 0 {
		cfg.Rows = DefaultRows
	}
	if cfg.Cols <= 0 {
		cfg.Cols = DefaultCols
	}
	if cfg.Width <= 0 {
		cfg.Width = DefaultWidth
	}
	if cfg.Height <= 0 {
		cfg.Height = DefaultHeight
	}
	if cfg.BorderWidth <= 0 {
		cfg.BorderWidth = DefaultBorderWidth
	}

	img := image.NewRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))

	// Fill canvas with frame color
	draw.Draw(img, img.Bounds(), &image.Uniform{C: cfg.ColorFrame}, image.Point{}, draw.Src)

	var (
		cellW   int
		cellH   int
		borderW = cfg.BorderWidth
		padLeft int
		padTop  int
	)

	// If using exact grid.png dimensions, reproduce the exact pixel layout
	if cfg.Width == DefaultWidth && cfg.Height == DefaultHeight && cfg.Rows == DefaultRows && cfg.Cols == DefaultCols {
		cellW = 8
		cellH = 8
		borderW = 1
		padLeft = 3
		padTop = 1
	} else {
		// Custom geometry: calculate cell dimensions to fit canvas
		availW := cfg.Width - (cfg.Cols+1)*borderW
		if availW < cfg.Cols {
			availW = cfg.Cols
		}
		cellW = availW / cfg.Cols
		if cellW < 1 {
			cellW = 1
		}

		availH := cfg.Height - (cfg.Rows+1)*borderW
		if availH < cfg.Rows {
			availH = cfg.Rows
		}
		cellH = availH / cfg.Rows
		if cellH < 1 {
			cellH = 1
		}

		totalGridW := cfg.Cols*cellW + (cfg.Cols+1)*borderW
		totalGridH := cfg.Rows*cellH + (cfg.Rows+1)*borderW

		padLeft = (cfg.Width - totalGridW) / 2
		if padLeft < 0 {
			padLeft = 0
		}
		padTop = (cfg.Height - totalGridH) / 2
		if padTop < 0 {
			padTop = 0
		}
	}

	// Draw grid border area
	totalW := cfg.Cols*cellW + (cfg.Cols+1)*borderW
	totalH := cfg.Rows*cellH + (cfg.Rows+1)*borderW
	gridRect := image.Rect(padLeft, padTop, padLeft+totalW, padTop+totalH).Intersect(img.Bounds())
	draw.Draw(img, gridRect, &image.Uniform{C: cfg.ColorBorder}, image.Point{}, draw.Src)

	// Draw each cell
	for r := 0; r < cfg.Rows; r++ {
		for c := 0; c < cfg.Cols; c++ {
			idx := r*cfg.Cols + c
			cellColor := cfg.ColorOffline

			if idx < len(results) {
				switch results[idx].Status {
				case scanner.StatusOnline:
					cellColor = cfg.ColorOnline
				case scanner.StatusHighlight:
					cellColor = cfg.ColorHighlight
				case scanner.StatusSlow:
					cellColor = cfg.ColorSlow
				default:
					cellColor = cfg.ColorOffline
				}
			}

			x0 := padLeft + borderW + c*(cellW+borderW)
			y0 := padTop + borderW + r*(cellH+borderW)
			x1 := x0 + cellW
			y1 := y0 + cellH

			cellBounds := image.Rect(x0, y0, x1, y1).Intersect(img.Bounds())
			if !cellBounds.Empty() {
				draw.Draw(img, cellBounds, &image.Uniform{C: cellColor}, image.Point{}, draw.Src)
			}
		}
	}

	return img
}

// SavePNG writes the RGBA image to a PNG file at the specified path.
func SavePNG(img *image.RGBA, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	return png.Encode(f, img)
}

// WritePNG encodes the image into any io.Writer.
func WritePNG(img *image.RGBA, w io.Writer) error {
	return png.Encode(w, img)
}
