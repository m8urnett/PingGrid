package grid

import (
	"image/color"
	"testing"
)

func TestParseHexColor(t *testing.T) {
	tests := []struct {
		input   string
		want    color.RGBA
		wantErr bool
	}{
		{input: "#324050", want: color.RGBA{R: 0x32, G: 0x40, B: 0x50, A: 0xFF}, wantErr: false},
		{input: "324050", want: color.RGBA{R: 0x32, G: 0x40, B: 0x50, A: 0xFF}, wantErr: false},
		{input: "#7DBEFF", want: color.RGBA{R: 0x7D, G: 0xBE, B: 0xFF, A: 0xFF}, wantErr: false},
		{input: "#9BD296", want: color.RGBA{R: 0x9B, G: 0xD2, B: 0x96, A: 0xFF}, wantErr: false},
		{input: "#000", want: color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xFF}, wantErr: false},
		{input: "#FFF", want: color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}, wantErr: false},
		{input: "#F00", want: color.RGBA{R: 0xFF, G: 0x00, B: 0x00, A: 0xFF}, wantErr: false},
		{input: "invalid", wantErr: true},
		{input: "#12345", wantErr: true},
		{input: "#1234567", wantErr: true},
		{input: "#GGFFFF", wantErr: true},
	}

	for _, tc := range tests {
		got, err := ParseHexColor(tc.input)
		if (err != nil) != tc.wantErr {
			t.Errorf("ParseHexColor(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
			continue
		}
		if !tc.wantErr && got != tc.want {
			t.Errorf("ParseHexColor(%q) = %+v, want %+v", tc.input, got, tc.want)
		}
	}
}

func TestHexString(t *testing.T) {
	c := color.RGBA{R: 0x32, G: 0x40, B: 0x50, A: 0xFF}
	if got := HexString(c); got != "#324050" {
		t.Errorf("HexString() = %q, want #324050", got)
	}
}
