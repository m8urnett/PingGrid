package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestVersionAndWindowsResourceContract(t *testing.T) {
	if !regexp.MustCompile(`^1\.[0-9]{2}\.[0-9]{3}$`).MatchString(version) {
		t.Fatalf("version %q does not match required 1.xx.NNN format", version)
	}
	resource, err := os.ReadFile("pg_windows_amd64.rc")
	if err != nil {
		t.Fatalf("read Windows resource source: %v", err)
	}
	text := string(resource)
	parts := strings.Split(version, ".")
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		t.Fatalf("parse minor version: %v", err)
	}
	build, err := strconv.Atoi(parts[2])
	if err != nil {
		t.Fatalf("parse build version: %v", err)
	}
	for _, required := range []string{
		fmt.Sprintf("FILEVERSION 1,%d,%d,0", minor, build),
		fmt.Sprintf("PRODUCTVERSION 1,%d,%d,0", minor, build),
		`VALUE "CompanyName", "Xato\0"`,
		`VALUE "LegalCopyright", "Mark Burnett (mb@xato.net)\0"`,
		`VALUE "FileVersion", "` + version + `\0"`,
		`VALUE "ProductVersion", "` + version + `\0"`,
	} {
		if !strings.Contains(text, required) {
			t.Errorf("Windows resource is missing or mismatches %q", required)
		}
	}
}
