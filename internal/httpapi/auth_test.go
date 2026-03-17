package httpapi

import "testing"

func TestResolveAccountHeader(t *testing.T) {
	v, err := ResolveAccountHeader("acc_123")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if v != "acc_123" {
		t.Fatalf("got %q", v)
	}
	if _, err := ResolveAccountHeader("  "); err == nil {
		t.Fatal("expected error for empty header")
	}
}
