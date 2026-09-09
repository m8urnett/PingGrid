package grid

import (
	"bytes"
	"image/png"
	"net"
	"testing"

	"github.com/m8urnett/PingGrid/internal/scanner"
)

func TestRenderDefaultDimensions(t *testing.T) {
	cfg := DefaultConfig()
	results := make([]scanner.HostResult, 256)
	for i := range results {
		results[i] = scanner.HostResult{
			IP:     net.IPv4(192, 168, 1, byte(i)),
			Status: scanner.StatusOffline,
		}
	}
	// Set a couple to online / highlight
	results[1].Status = scanner.StatusHighlight
	results[2].Status = scanner.StatusOnline

	img := Render(cfg, results)

	if img.Bounds().Dx() != DefaultWidth {
		t.Errorf("expected width %d, got %d", DefaultWidth, img.Bounds().Dx())
	}
	if img.Bounds().Dy() != DefaultHeight {
		t.Errorf("expected height %d, got %d", DefaultHeight, img.Bounds().Dy())
	}

	// Verify sample pixel colors:
	// (0, 0) should be frame color PaletteCharcoal #2c2c2c
	c00 := img.RGBAAt(0, 0)
	if c00 != DefaultColorFrame {
		t.Errorf("at (0,0) expected %+v, got %+v", DefaultColorFrame, c00)
	}

	// (3, 1) should be border PaletteCharcoal #2c2c2c
	cBorder := img.RGBAAt(3, 1)
	if cBorder != DefaultColorBorder {
		t.Errorf("at (3,1) expected border %+v, got %+v", DefaultColorBorder, cBorder)
	}

	// Cell (0, 0) is from x=4..11, y=2..9. Status is Offline (#404e41)
	cCell0 := img.RGBAAt(4, 2)
	if cCell0 != DefaultColorOffline {
		t.Errorf("cell 0 expected offline %+v, got %+v", DefaultColorOffline, cCell0)
	}

	// Cell (0, 1) [result 1] is Highlight (#e4f9d4). x = 3 + 1 + 9 = 13
	cCell1 := img.RGBAAt(13, 2)
	if cCell1 != DefaultColorHighlight {
		t.Errorf("cell 1 expected highlight %+v, got %+v", DefaultColorHighlight, cCell1)
	}

	// Cell (0, 2) [result 2] is Online (#4d86a2). x = 3 + 1 + 18 = 22
	cCell2 := img.RGBAAt(22, 2)
	if cCell2 != DefaultColorOnline {
		t.Errorf("cell 2 expected online %+v, got %+v", DefaultColorOnline, cCell2)
	}

	// Verify rightmost cell (0, 31) [result 31] is not clipped (x = 3 + 1 + 31*9 = 283..290)
	cCell31 := img.RGBAAt(283, 2)
	if cCell31 != DefaultColorOffline {
		t.Errorf("cell 31 expected offline %+v, got %+v", DefaultColorOffline, cCell31)
	}

	// Right outer border at x=291
	cRightBorder := img.RGBAAt(291, 2)
	if cRightBorder != DefaultColorBorder {
		t.Errorf("right border expected %+v, got %+v", DefaultColorBorder, cRightBorder)
	}

	// Right outer frame at x=294
	cRightFrame := img.RGBAAt(294, 2)
	if cRightFrame != DefaultColorFrame {
		t.Errorf("right frame expected %+v, got %+v", DefaultColorFrame, cRightFrame)
	}

	// Test PNG encoding
	var buf bytes.Buffer
	if err := WritePNG(img, &buf); err != nil {
		t.Fatalf("WritePNG failed: %v", err)
	}
	decoded, err := png.Decode(&buf)
	if err != nil {
		t.Fatalf("png.Decode failed: %v", err)
	}
	if decoded.Bounds().Dx() != DefaultWidth || decoded.Bounds().Dy() != DefaultHeight {
		t.Errorf("decoded PNG bounds mismatch: %v", decoded.Bounds())
	}
}

func TestRenderCustomDimensions(t *testing.T) {
	cfg := GridConfig{
		Rows:           4,
		Cols:           4,
		Width:          100,
		Height:         100,
		BorderWidth:    2,
		ColorOffline:   DefaultColorOffline,
		ColorOnline:    DefaultColorOnline,
		ColorHighlight: DefaultColorHighlight,
		ColorBorder:    DefaultColorBorder,
		ColorFrame:     DefaultColorFrame,
	}

	results := make([]scanner.HostResult, 16)
	results[0].Status = scanner.StatusOnline

	img := Render(cfg, results)
	if img.Bounds().Dx() != 100 || img.Bounds().Dy() != 100 {
		t.Errorf("custom bounds mismatch: %v", img.Bounds())
	}
}

func TestGetScheme(t *testing.T) {
	for _, scheme := range AvailableSchemes() {
		cfg, err := GetScheme(scheme)
		if err != nil {
			t.Errorf("GetScheme(%q) returned error: %v", scheme, err)
		}
		if cfg.Width != DefaultWidth || cfg.Height != DefaultHeight {
			t.Errorf("GetScheme(%q) dimensions mismatch", scheme)
		}
	}

	if _, err := GetScheme("invalid-scheme"); err == nil {
		t.Errorf("expected error for invalid scheme, got nil")
	}
}
