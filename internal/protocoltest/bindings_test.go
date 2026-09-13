package protocoltest

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/bnema/libwldevices-go/internal/testcompositor"
	"github.com/bnema/wlturbo"
	"github.com/bnema/wlturbo/wl"
)

// Global names announced by the fake compositor.
const (
	seatGlobalName    = 1
	managerGlobalName = 2
)

// newFixture connects to a fake compositor announcing the fixture globals and
// binds wl_seat and the fixture manager.
func newFixture(t *testing.T) (*testcompositor.Server, *wl.Display, *FixtureManager, int) {
	t.Helper()

	srv := testcompositor.Start(t,
		testcompositor.Global{Name: seatGlobalName, Interface: "wl_seat", Version: 5},
		testcompositor.Global{Name: managerGlobalName, Interface: FixtureManagerInterface, Version: 2},
	)
	srv.Env(t)

	display, err := wl.Connect("")
	if err != nil {
		t.Fatalf("wl.Connect: %v", err)
	}
	t.Cleanup(func() { _ = display.Close() })

	ctx := display.Context()
	registry := display.GetRegistry()

	if err := display.Roundtrip(); err != nil {
		t.Fatalf("Roundtrip: %v", err)
	}
	if _, ok := registry.FindGlobal(FixtureManagerInterface); !ok {
		t.Fatalf("compositor did not announce %s", FixtureManagerInterface)
	}

	seat := wl.NewSeat(ctx)
	seatID, err := registry.BindID(seatGlobalName, "wl_seat", 5)
	if err != nil {
		t.Fatalf("BindID(wl_seat): %v", err)
	}
	seat.SetID(seatID)
	ctx.Register(seat)

	manager := NewFixtureManager(ctx)
	if err := registry.Bind(managerGlobalName, FixtureManagerInterface, 2, manager); err != nil {
		t.Fatalf("Bind(%s): %v", FixtureManagerInterface, err)
	}

	return srv, display, manager, int(seatID)
}

// waitForRequest waits for a recorded request on an object and opcode.
func waitForRequest(t *testing.T, srv *testcompositor.Server, object uint32, opcode uint16) testcompositor.Request {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		for _, req := range srv.Requests() {
			if req.Object == object && req.Opcode == opcode {
				return req
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("no request object=%d opcode=%d; recorded %+v", object, opcode, srv.Requests())
		}
		time.Sleep(time.Millisecond)
	}
}

func TestBindSendsDeclaredNameVersionAndInterface(t *testing.T) {
	srv, _, manager, _ := newFixture(t)

	registryID := srv.RegistryID()
	deadline := time.Now().Add(2 * time.Second)
	var bind testcompositor.Request
	for {
		for _, req := range srv.Requests() {
			if req.Object != registryID || req.Opcode != 0 {
				continue
			}
			if iface, _ := req.String(4); iface == FixtureManagerInterface {
				bind = req
			}
		}
		if bind.Body != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("no bind request for %s: %+v", FixtureManagerInterface, srv.Requests())
		}
		time.Sleep(time.Millisecond)
	}

	if got := bind.Uint32(0); got != managerGlobalName {
		t.Errorf("bind name = %d, want %d", got, managerGlobalName)
	}
	iface, consumed := bind.String(4)
	if iface != FixtureManagerInterface {
		t.Fatalf("bind interface = %q, want %q", iface, FixtureManagerInterface)
	}
	if got := bind.Uint32(4 + consumed); got != 2 {
		t.Errorf("bind version = %d, want 2", got)
	}
	if manager.ID() == 0 {
		t.Error("bound manager has object ID 0")
	}
}

func TestNewIDRequestSendsSeatAndAllocatesChild(t *testing.T) {
	srv, _, manager, seatID := newFixture(t)

	thing, err := manager.CreateThing(nil)
	if err != nil {
		t.Fatalf("CreateThing: %v", err)
	}
	if thing.ID() == 0 {
		t.Fatal("CreateThing returned an object with ID 0")
	}
	if thing.ID() == manager.ID() {
		t.Fatal("CreateThing reused the manager object ID")
	}

	req := waitForRequest(t, srv, manager.ID(), 0)
	if got := req.Uint32(0); got != 0 {
		t.Errorf("nil seat encoded as object ID %d, want 0", got)
	}
	if got := req.Uint32(4); got != thing.ID() {
		t.Errorf("new_id = %d, want the allocated child ID %d", got, thing.ID())
	}

	// A bound seat must be sent as its real object ID.
	seat := wl.NewSeat(manager.Context())
	seat.SetID(uint32(seatID))
	manager.Context().Register(seat)

	second, err := manager.CreateThing(seat)
	if err != nil {
		t.Fatalf("CreateThing(seat): %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		var found bool
		for _, req := range srv.Requests() {
			if req.Object == manager.ID() && req.Opcode == 0 && req.Uint32(4) == second.ID() {
				if got := req.Uint32(0); got != uint32(seatID) {
					t.Fatalf("seat encoded as %d, want %d", got, seatID)
				}
				found = true
			}
		}
		if found {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the second CreateThing request was never recorded")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestScalarRequestsEncodeProtocolValues(t *testing.T) {
	srv, _, manager, _ := newFixture(t)

	if err := manager.SetRatio(wl.Fixed(640)); err != nil {
		t.Fatalf("SetRatio: %v", err)
	}
	ratio := waitForRequest(t, srv, manager.ID(), 1)
	if got := int32(ratio.Uint32(0)); got != 640 {
		t.Errorf("ratio = %d, want 640", got)
	}

	if err := manager.SetLabel("hello"); err != nil {
		t.Fatalf("SetLabel: %v", err)
	}
	label := waitForRequest(t, srv, manager.ID(), 2)
	got, _ := label.String(0)
	if got != "hello" {
		t.Errorf("label = %q, want %q", got, "hello")
	}

	// A nil object argument is encoded as object ID 0, never as a typed nil.
	if err := manager.SetRegion(nil); err != nil {
		t.Fatalf("SetRegion(nil): %v", err)
	}
	region := waitForRequest(t, srv, manager.ID(), 3)
	if got := region.Uint32(0); got != 0 {
		t.Errorf("nil region encoded as %d, want 0", got)
	}

	if err := manager.AttachRaw([]byte{0xde, 0xad, 0xbe, 0xef}); err != nil {
		t.Fatalf("AttachRaw: %v", err)
	}
	blob := waitForRequest(t, srv, manager.ID(), 6)
	if length := blob.Uint32(0); length != 4 {
		t.Errorf("array length = %d, want 4", length)
	}
	if blob.Body[4] != 0xde || blob.Body[5] != 0xad || blob.Body[6] != 0xbe || blob.Body[7] != 0xef {
		t.Errorf("array bytes = % x, want de ad be ef", blob.Body[4:8])
	}
}

func TestFdRequestAttachesDescriptorOutOfBand(t *testing.T) {
	srv, _, manager, _ := newFixture(t)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	if err := manager.PassFd(int(w.Fd())); err != nil {
		t.Fatalf("PassFd: %v", err)
	}

	req := waitForRequest(t, srv, manager.ID(), 7)
	if len(req.FDs) != 1 {
		t.Fatalf("request carried %d descriptors, want 1", len(req.FDs))
	}
	// The descriptor travels out of band, so the body carries no value for it.
	if len(req.Body) != 0 {
		t.Errorf("request body = %d bytes, want none", len(req.Body))
	}
}

func TestEventsDispatchToRegisteredHandlers(t *testing.T) {
	srv, display, manager, _ := newFixture(t)

	thing, err := manager.CreateThing(nil)
	if err != nil {
		t.Fatalf("CreateThing: %v", err)
	}

	var (
		serial   uint32
		label    string
		geometry [4]int64
		scaled   wl.Fixed
		peerID   uint32
		blob     []byte
		mode     *FixtureMode
	)

	thing.OnReady(func(value uint32) { serial = value })
	thing.OnLabelChanged(func(value string) { label = value })
	thing.OnGeometry(func(x, y int32, width, height uint32) {
		geometry = [4]int64{int64(x), int64(y), int64(width), int64(height)}
	})
	thing.OnScaled(func(value wl.Fixed) { scaled = value })
	thing.OnPeer(func(surfaceID uint32) { peerID = surfaceID })
	thing.OnBlob(func(data []byte) { blob = data })
	thing.OnMode(func(object *FixtureMode) { mode = object })

	events := []struct {
		opcode uint16
		args   []any
	}{
		{0, []any{uint32(7)}},
		{1, []any{"changed"}},
		{2, []any{int32(-3), int32(5), uint32(800), uint32(600)}},
		{5, []any{int32(384)}},
		{4, []any{uint32(99)}},
		{6, []any{[]byte{1, 2, 3, 4}}},
		{3, []any{uint32(4242)}},
	}
	for _, event := range events {
		if err := srv.SendEvent(thing.ID(), event.opcode, event.args...); err != nil {
			t.Fatalf("SendEvent(opcode=%d): %v", event.opcode, err)
		}
	}

	if err := display.Roundtrip(); err != nil {
		t.Fatalf("Roundtrip: %v", err)
	}

	if serial != 7 {
		t.Errorf("ready serial = %d, want 7", serial)
	}
	if label != "changed" {
		t.Errorf("label = %q, want %q", label, "changed")
	}
	if geometry != [4]int64{-3, 5, 800, 600} {
		t.Errorf("geometry = %v, want [-3 5 800 600]", geometry)
	}
	if scaled != wl.Fixed(384) {
		t.Errorf("scale = %d, want 384", scaled)
	}
	if peerID != 99 {
		t.Errorf("peer object ID = %d, want 99", peerID)
	}
	if len(blob) != 4 || blob[0] != 1 || blob[3] != 4 {
		t.Errorf("blob = % x, want 01 02 03 04", blob)
	}
	if mode == nil {
		t.Fatal("mode event did not create a FixtureMode object")
	}
	if mode.ID() != 4242 {
		t.Fatalf("mode object ID = %d, want 4242", mode.ID())
	}

	// The object created by the event is registered, so it can be used.
	if err := mode.Destroy(); err != nil {
		t.Fatalf("mode.Destroy: %v", err)
	}
	waitForRequest(t, srv, 4242, 0)
}

func TestDestructorUnregistersObject(t *testing.T) {
	srv, display, manager, _ := newFixture(t)

	thing, err := manager.CreateThing(nil)
	if err != nil {
		t.Fatalf("CreateThing: %v", err)
	}
	thingID := thing.ID()

	if err := thing.Destroy(); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
	waitForRequest(t, srv, thingID, 1)

	// The destroyed object must no longer receive events.
	if err := srv.SendEvent(thingID, 0, uint32(1)); err != nil {
		t.Fatalf("SendEvent: %v", err)
	}
	err = display.Roundtrip()
	if err == nil {
		t.Fatal("dispatch of an event for a destroyed object succeeded")
	}

	var perr *wlturbo.ProtocolError
	if !errors.As(err, &perr) {
		t.Fatalf("error = %v (%T), want a protocol error", err, err)
	}
	if perr.Kind != "unknown_object" {
		t.Fatalf("error kind = %q, want %q", perr.Kind, "unknown_object")
	}
}
