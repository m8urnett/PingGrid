package grid

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"
	"path/filepath"
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
		// Earth theme: Charcoal frame/border, Earth brown offline (#886d5b), Sage online (#b1b9a0), Terracotta highlight (#ab7550)
		cfg.ColorFrame = PaletteCharcoal
		cfg.ColorBorder = PaletteCharcoal
		cfg.ColorOffline = PaletteEarthCellBg
		cfg.ColorOnline = PaletteSage
		cfg.ColorHighlight = PaletteTerracotta
		cfg.ColorSlow = PaletteSlateTeal
		return cfg, nil

	case "moss", "forest":
		// Moss theme: Charcoal frame/border, Moss green offline (#465a47), Sage online (#b1b9a0), Mint highlight
		cfg.ColorFrame = PaletteCharcoal
		cfg.ColorBorder = PaletteCharcoal
		cfg.ColorOffline = PaletteMossCellBg
		cfg.ColorOnline = PaletteSage
		cfg.ColorHighlight = PaletteMint
		cfg.ColorSlow = PaletteTerracotta
		return cfg, nil

	case "linen", "cream":
		// Linen theme: Warm Linen frame & offline (#f4eeeb), Light Gray border, Linen taupe online/foreground (#897e79), Teal highlight
		cfg.ColorFrame = PaletteWarmLinen
		cfg.ColorBorder = PaletteLightGray
		cfg.ColorOffline = PaletteWarmLinen
		cfg.ColorOnline = PaletteLinenFg
		cfg.ColorHighlight = PaletteSlateTeal
		cfg.ColorSlow = PaletteTerracotta
		return cfg, nil

	default:
		return cfg, fmt.Errorf("unknown scheme %q: available schemes are: %s", name, strings.Join(AvailableSchemes(), ", "))
	}
}

// AutoLayout calculates optimal row and column counts based on total host count.
// It prioritizes standard networking dimensions (powers of 2 for subnets), clean
// decimal groupings (e.g. 5x10 for 50 hosts, 10x10 for 100 hosts), and terminal-friendly
// aspect ratios without unnecessary empty rows.
func AutoLayout(totalHosts int) (int, int) {
	if totalHosts <= 0 {
		return DefaultRows, DefaultCols
	}
	if totalHosts == 1 {
		return 1, 1
	}

	// Standard CIDR subnet power-of-two mappings
	switch totalHosts {
	case 2:
		return 1, 2
	case 3:
		return 1, 3
	case 4:
		return 2, 2
	case 8:
		return 2, 4
	case 16:
		return 4, 4
	case 32:
		return 4, 8
	case 64:
		return 4, 16
	case 128:
		return 8, 16
	case 256:
		return 8, 32
	}

	// Clean decimal multiples (e.g. 50 -> 5x10, 100 -> 10x10)
	if totalHosts <= 100 && totalHosts%10 == 0 {
		return totalHosts / 10, 10
	}
	if totalHosts <= 50 && totalHosts%5 == 0 {
		return totalHosts / 5, 5
	}
	if totalHosts <= 200 && totalHosts%20 == 0 {
		return totalHosts / 20, 20
	}

	// For small counts <= 10
	if totalHosts <= 10 {
		if totalHosts <= 5 {
			return 1, totalHosts
		}
		return 2, (totalHosts + 1) / 2
	}

	// Candidate column counts to evaluate for arbitrary N
	var candidates []int
	if totalHosts <= 32 {
		candidates = []int{8, 10, 6, 4, 16}
	} else if totalHosts <= 128 {
		candidates = []int{10, 16, 12, 8, 20}
	} else if totalHosts <= 256 {
		candidates = []int{16, 20, 32, 24, 10}
	} else {
		candidates = []int{32, 24, 16, 20}
	}

	bestCols := 16
	bestRows := (totalHosts + bestCols - 1) / bestCols
	minWaste := bestRows*bestCols - totalHosts

	for _, c := range candidates {
		r := (totalHosts + c - 1) / c
		waste := r*c - totalHosts
		if waste < minWaste || (waste == minWaste && c >= r && (bestCols < bestRows || c < bestCols)) {
			minWaste = waste
			bestCols = c
			bestRows = r
		}
	}

	return bestRows, bestCols
}

// AutosizeDimensions calculates canvas width and height for a given row and column count,
// preserving the standard 8x8 square cell size and frame padding proportions.
func AutosizeDimensions(rows, cols, borderW int) (int, int) {
	if rows <= 0 {
		rows = DefaultRows
	}
	if cols <= 0 {
		cols = DefaultCols
	}
	if borderW <= 0 {
		borderW = DefaultBorderWidth
	}
	totalGridW := cols*8 + (cols+1)*borderW
	totalGridH := rows*8 + (rows+1)*borderW
	return totalGridW + 6, totalGridH + 4
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

	// If using exact default dimensions, reproduce the reference pixel layout
	if cfg.Width == DefaultWidth && cfg.Height == DefaultHeight && cfg.Rows == DefaultRows && cfg.Cols == DefaultCols {
		cellW = 8
		cellH = 8
		borderW = 1
		padLeft = 3
		padTop = 1
	} else {
		// Custom geometry: calculate cell dimensions to fit canvas while preserving square cell size
		availW := cfg.Width - (cfg.Cols+1)*borderW
		if availW < cfg.Cols {
			availW = cfg.Cols
		}
		availH := cfg.Height - (cfg.Rows+1)*borderW
		if availH < cfg.Rows {
			availH = cfg.Rows
		}
		cW := availW / cfg.Cols
		cH := availH / cfg.Rows
		cellSize := cW
		if cH < cellSize {
			cellSize = cH
		}
		if cellSize < 1 {
			cellSize = 1
		}
		cellW = cellSize
		cellH = cellSize

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

			x0 := padLeft + borderW + c*(cellW+borderW)
			y0 := padTop + borderW + r*(cellH+borderW)
			x1 := x0 + cellW
			y1 := y0 + cellH
			cellBounds := image.Rect(x0, y0, x1, y1).Intersect(img.Bounds())
			if cellBounds.Empty() {
				continue
			}

			if idx >= len(results) {
				// When range and grid sizes don't match (extra cells), leave with no color (frame background)
				draw.Draw(img, cellBounds, &image.Uniform{C: cfg.ColorFrame}, image.Point{}, draw.Src)
				continue
			}

			var cellColor color.RGBA
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

			draw.Draw(img, cellBounds, &image.Uniform{C: cellColor}, image.Point{}, draw.Src)
		}
	}

	return img
}

// SavePNG writes the RGBA image to a PNG file at the specified path.
func SavePNG(img *image.RGBA, path string) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return err
		}
	}
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
