package main

import "testing"

func TestClampFloat(t *testing.T) {
	if got := clampFloat(-1, 0, 10); got != 0 {
		t.Fatalf("clampFloat below min got %f, want 0", got)
	}
	if got := clampFloat(11, 0, 10); got != 10 {
		t.Fatalf("clampFloat above max got %f, want 10", got)
	}
	if got := clampFloat(4.5, 0, 10); got != 4.5 {
		t.Fatalf("clampFloat in range got %f, want 4.5", got)
	}
}

func TestClampInt(t *testing.T) {
	if got := clampInt(-1, 0, 10); got != 0 {
		t.Fatalf("clampInt below min got %d, want 0", got)
	}
	if got := clampInt(20, 0, 10); got != 10 {
		t.Fatalf("clampInt above max got %d, want 10", got)
	}
	if got := clampInt(7, 0, 10); got != 7 {
		t.Fatalf("clampInt in range got %d, want 7", got)
	}
}
