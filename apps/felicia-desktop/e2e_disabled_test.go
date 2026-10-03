//go:build !e2e

package main

import (
	"flag"
	"net/http"
	"testing"
)

func TestNormalBuildHasNoE2ETransport(t *testing.T) {
	t.Setenv("FELICIA_DESKTOP_DEBUG_ADDR", "0.0.0.0:8125")
	t.Setenv("FELICIA_E2E_TOKEN", "synthetic-not-a-real-secret")
	if flag.Lookup("e2e-addr") != nil {
		t.Fatal("normal build registered e2e listener flag")
	}
	if handled, err := serveE2E(http.NotFoundHandler()); handled || err != nil {
		t.Fatalf("normal build activated test transport: handled=%v err=%v", handled, err)
	}
}
