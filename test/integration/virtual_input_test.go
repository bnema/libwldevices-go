package integration

import (
	"os"
	"os/exec"
	"testing"
)

// TestVirtualInputSmoke opts in to an isolated NeferWL session via LIBWLDEVICES_HEADLESS.
func TestVirtualInputSmoke(t *testing.T) {
	if os.Getenv("LIBWLDEVICES_HEADLESS") == "" {
		t.Skip("set LIBWLDEVICES_HEADLESS to a NeferWL binary")
	}
	cmd := exec.Command("bash", "run.sh")
	out, err := cmd.CombinedOutput()
	t.Logf("NeferWL smoke:\n%s", out)
	if err != nil {
		t.Fatalf("headless smoke: %v", err)
	}
}
