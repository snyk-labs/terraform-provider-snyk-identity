package main

import "testing"

// Smoke test so the root module participates in `go test ./...` like other packages.
func TestRootModule(t *testing.T) {
	t.Parallel()
	t.Log("main package builds with tests")
}
