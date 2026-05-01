package tron

import (
	"net/http"
	"testing"
	"time"
)

func TestIsLikelyTronAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		want    bool
	}{
		{name: "valid tron address", address: "TJRyWwFs9wTFGZg3JbrVriFbNfCug5tDeC", want: true},
		{name: "too short", address: "231", want: false},
		{name: "wrong prefix", address: "AJRyWwFs9wTFGZg3JbrVriFbNfCug5tDeC", want: false},
		{name: "empty", address: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isLikelyTronAddress(tc.address); got != tc.want {
				t.Fatalf("isLikelyTronAddress(%q) = %v, want %v", tc.address, got, tc.want)
			}
		})
	}
}

func TestTronGridCooldownLifecycle(t *testing.T) {
	resetTronGridCooldown()
	t.Cleanup(resetTronGridCooldown)

	now := time.Unix(1_700_000_000, 0)
	if shouldSkipTronGrid(now) {
		t.Fatal("expected cooldown to be disabled initially")
	}

	resp := &http.Response{
		Header: http.Header{
			"Retry-After": []string{"90"},
		},
	}

	until := updateTronGridCooldown(now, resp)
	if got, want := until, now.Add(90*time.Second); !got.Equal(want) {
		t.Fatalf("updateTronGridCooldown() = %v, want %v", got, want)
	}

	if !shouldSkipTronGrid(now.Add(30 * time.Second)) {
		t.Fatal("expected cooldown to skip requests inside retry window")
	}

	if shouldSkipTronGrid(now.Add(91 * time.Second)) {
		t.Fatal("expected cooldown to expire after retry window")
	}

	resetTronGridCooldown()
	if shouldSkipTronGrid(now) {
		t.Fatal("expected cooldown reset to clear state")
	}
}
