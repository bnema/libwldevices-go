package scanner

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	fixtureXML      = "testdata/protocol_fixture.xml"
	committedOutput = "../internal/protocoltest/bindings.go"
)

// The committed fixture bindings must be exactly what the generator produces:
// they are the golden output that proves regeneration is reproducible.
func TestGenerateFixtureBindingsMatchCommittedFile(t *testing.T) {
	generated := generateFixture(t, fixtureXML, "protocoltest")

	committed, err := os.ReadFile(committedOutput)
	if err != nil {
		t.Fatalf("read committed bindings: %v", err)
	}

	if !bytes.Equal(generated, committed) {
		t.Fatalf("committed bindings are stale.\nRegenerate with:\n  go run ./scanner/cmd/wayland-scanner -p protocoltest -o %s %s\n",
			committedOutput, fixtureXML)
	}
}

// Generation must be deterministic and independent of the path the XML was
// read from, otherwise golden comparisons and CI diffs are unstable.
func TestGenerateIsDeterministicAndPathIndependent(t *testing.T) {
	first := generateFixture(t, fixtureXML, "protocoltest")
	second := generateFixture(t, fixtureXML, "protocoltest")
	if !bytes.Equal(first, second) {
		t.Fatal("two generations of the same protocol differ")
	}

	relative := filepath.Join("testdata", "..", "testdata", "protocol_fixture.xml")
	third := generateFixture(t, relative, "protocoltest")
	if !bytes.Equal(first, third) {
		t.Fatal("generated output depends on the input path")
	}
}

// The generated code must not silently drop a second new_id argument.
func TestGenerateRejectsMultipleNewIDArguments(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<protocol name="two_new_ids">
  <interface name="wlv_two_v1" version="1">
    <request name="make_pair">
      <arg name="first" type="new_id" interface="wlv_two_v1"/>
      <arg name="second" type="new_id" interface="wlv_two_v1"/>
    </request>
  </interface>
</protocol>`

	path := filepath.Join(t.TempDir(), "two_new_ids.xml")
	if err := os.WriteFile(path, []byte(xml), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	s := NewScanner()
	if err := s.ParseXML(path); err != nil {
		t.Fatalf("ParseXML: %v", err)
	}

	_, err := s.Generate("twoids")
	if err == nil {
		t.Fatal("Generate accepted a request with two new_id arguments")
	}
	if !strings.Contains(err.Error(), "new_id") {
		t.Fatalf("error = %v, want it to name the unsupported new_id argument", err)
	}
}

// Generated code must not carry transport responsibilities.
func TestGeneratedCodeHasNoTransportOwnership(t *testing.T) {
	generated := generateFixture(t, fixtureXML, "protocoltest")

	forbidden := []string{
		"net.Dial",
		"syscall.",
		"Sendmsg",
		"Recvmsg",
		"socket",
	}
	for _, needle := range forbidden {
		if bytes.Contains(generated, []byte(needle)) {
			t.Errorf("generated code contains %q; framing and descriptors belong to the transport", needle)
		}
	}

	if !bytes.Contains(generated, []byte("wl.BaseProxy")) {
		t.Error("generated code does not embed wl.BaseProxy")
	}
}

func generateFixture(t *testing.T, path, pkg string) []byte {
	t.Helper()

	s := NewScanner()
	if err := s.ParseXML(path); err != nil {
		t.Fatalf("ParseXML(%s): %v", path, err)
	}
	code, err := s.Generate(pkg)
	if err != nil {
		t.Fatalf("Generate(%s): %v", path, err)
	}
	return code
}
