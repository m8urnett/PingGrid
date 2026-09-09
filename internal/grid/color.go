package grid

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
)

// Allowed palette colors specified by design standards.
var (
	PaletteCharcoal   = color.RGBA{R: 0x2C, G: 0x2C, B: 0x2C, A: 0xFF} // #2c2c2c
	PaletteLightGray  = color.RGBA{R: 0xEF, G: 0xEF, B: 0xEF, A: 0xFF} // #efefef
	PaletteWarmLinen  = color.RGBA{R: 0xF4, G: 0xEE, B: 0xEB, A: 0xFF} // #897e79
	PaletteSage       = color.RGBA{R: 0xB1, G: 0xB9, B: 0xA0, A: 0xFF} // #b1b9a0
	PaletteSlateTeal  = color.RGBA{R: 0x4D, G: 0x86, B: 0xA2, A: 0xFF} // #4d86a2
	PaletteTerracotta = color.RGBA{R: 0xAB, G: 0x75, B: 0x50, A: 0xFF} // #ab7550
	PaletteMossForest = color.RGBA{R: 0x40, G: 0x4E, B: 0x41, A: 0xFF} // #404e41
	PaletteOffWhite   = color.RGBA{R: 0xFC, G: 0xFC, B: 0xFC, A: 0xFF} // #fcfcfc
	PaletteMint       = color.RGBA{R: 0xE4, G: 0xF9, B: 0xD4, A: 0xFF} // #e4f9d4

	// Additional theme specific colors
	PaletteMossCellBg  = color.RGBA{R: 0x46, G: 0x5A, B: 0x47, A: 0xFF} // #465a47
	PaletteEarthCellBg = color.RGBA{R: 0x88, G: 0x6D, B: 0x5B, A: 0xFF} // #886d5b
	PaletteLinenFg     = color.RGBA{R: 0x89, G: 0x7E, B: 0x79, A: 0xFF} // #C7BEAE
)

// AllowedPaletteHex returns the list of all allowed palette hex strings.
var AllowedPaletteHex = []string{
	"#2c2c2c",
	"#efefef",
	"#f4eeeb",
	"#b1b9a0",
	"#4d86a2",
	"#ab7550",
	"#404e41",
	"#fcfcfc",
	"#e4f9d4",
	"#465a47",
	"#886d5b",
	"#897e79",
}

// ParseHexColor parses a hex color string into a color.RGBA.
// Supports formats: "#RRGGBB", "RRGGBB", "#RGB", "RGB", "0xRRGGBB" (with or without leading '#').
func ParseHexColor(s string) (color.RGBA, error) {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"'`)
	s = strings.TrimPrefix(s, "#")
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")

	switch len(s) {
	case 6:
		r, err := strconv.ParseUint(s[0:2], 16, 8)
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid red component in %q: %w", s, err)
		}
		g, err := strconv.ParseUint(s[2:4], 16, 8)
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid green component in %q: %w", s, err)
		}
		b, err := strconv.ParseUint(s[4:6], 16, 8)
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid blue component in %q: %w", s, err)
		}
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 0xFF}, nil

	case 3:
		r, err := strconv.ParseUint(string(s[0])+string(s[0]), 16, 8)
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid red component in %q: %w", s, err)
		}
		g, err := strconv.ParseUint(string(s[1])+string(s[1]), 16, 8)
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid green component in %q: %w", s, err)
		}
		b, err := strconv.ParseUint(string(s[2])+string(s[2]), 16, 8)
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid blue component in %q: %w", s, err)
		}
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 0xFF}, nil

	default:
		return color.RGBA{}, fmt.Errorf("invalid hex color format %q: expected 3 or 6 hex digits", s)
	}
}

// HexString formats a color.RGBA into a standard #RRGGBB hex string (lowercase).
func HexString(c color.RGBA) string {
	return strings.ToLower(fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B))
}
