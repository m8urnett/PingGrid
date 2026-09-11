//go:build windows

package scanner

import "testing"

func TestIPAdapterInfoCurrentIPAddressIsPointer(t *testing.T) {
	var info ipAdapterInfo
	current := info.CurrentIPAddress
	if current != nil {
		t.Fatal("zero-value CurrentIPAddress pointer is non-nil")
	}
}
