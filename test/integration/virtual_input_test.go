// Package integration runs the headless wlroots compositor smoke harness as a
// Go test so it participates in `go test ./test/integration/`.
package integration

import (
	"os/exec"
	"testing"
)

// TestVirtualInputSmoke runs test/integration/run.sh, which boots a pinned
// headless sway compositor in Docker and drives the standalone consumer module
// against it. It fails (never skips) when docker is unavailable, because this
// gate is required whenever the daemon is present.
func TestVirtualInputSmoke(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Fatalf("docker CLI is required for the headless compositor smoke test but was not found in PATH: %v", err)
	}

	cmd := exec.Command("bash", "run.sh")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	t.Logf("run.sh output:\n%s", out)
	if err != nil {
		t.Fatalf("test/integration/run.sh failed: %v", err)
	}
}
