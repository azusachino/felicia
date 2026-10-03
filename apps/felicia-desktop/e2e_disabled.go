//go:build !e2e

package main

import "net/http"

// Production builds do not register test flags or expose authoring over TCP.
func validateE2EPaths(_, _, _ string) error { return nil }

func serveE2E(_ http.Handler) (bool, error) { return false, nil }
