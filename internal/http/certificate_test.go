package http_test

import (
	"testing"

	"oras.land/oras/internal/http"
)

func Test_LoadCertPool(t *testing.T) {
	got, err := http.LoadCertPool("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected cert pool: %v, got: %v", nil, got)
	}
}
