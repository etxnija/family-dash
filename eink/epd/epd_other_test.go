//go:build !(linux && arm)

package epd

import "testing"

func TestNew_Unsupported(t *testing.T) {
	p, err := New(CableLeft)
	if err == nil {
		t.Fatal("New(CableLeft) = nil error, want error off linux/arm")
	}
	if p != nil {
		t.Errorf("New(CableLeft) panel = %v, want nil", p)
	}
}
