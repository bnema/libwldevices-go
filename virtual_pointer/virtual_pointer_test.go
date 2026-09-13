package virtual_pointer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bnema/libwldevices-go/internal/protocols"
	"github.com/bnema/libwldevices-go/internal/testcompositor"
	"github.com/bnema/wlturbo"
)

// Global names announced by the test compositor.
const (
	seatGlobalName    = 1
	pointerGlobalName = 2
)

// newCompositor starts a compositor announcing the globals the virtual pointer
// protocol needs and points the client library at it.
func newCompositor(t *testing.T) *testcompositor.Server {
	t.Helper()

	srv := testcompositor.Start(t,
		testcompositor.Global{Name: seatGlobalName, Interface: "wl_seat", Version: 5},
		testcompositor.Global{Name: pointerGlobalName, Interface: protocols.VirtualPointerManagerInterface, Version: 2},
	)
	srv.Env(t)
	return srv
}

// newManager creates a manager against the test compositor.
func newManager(t *testing.T, srv *testcompositor.Server) *VirtualPointerManager {
	t.Helper()

	manager, err := NewVirtualPointerManager(context.Background())
	if err != nil {
		t.Fatalf("NewVirtualPointerManager: %v", err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	return manager
}

// waitForRequest waits for a request on one object and opcode.
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
			t.Fatalf("no request object=%d opcode=%d; recorded: %+v", object, opcode, srv.Requests())
		}
		time.Sleep(time.Millisecond)
	}
}

// waitForBind waits for a wl_registry.bind request for the given interface.
func waitForBind(t *testing.T, srv *testcompositor.Server, iface string) testcompositor.Request {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		for _, req := range srv.Requests() {
			if req.Object != srv.RegistryID() || req.Opcode != 0 {
				continue
			}
			name, _ := req.String(4)
			if name == iface {
				return req
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("no bind request for %q; recorded: %+v", iface, srv.Requests())
		}
		time.Sleep(time.Millisecond)
	}
}

func TestManagerBindsVirtualPointerGlobal(t *testing.T) {
	srv := newCompositor(t)
	newManager(t, srv)

	bind := waitForBind(t, srv, protocols.VirtualPointerManagerInterface)

	if got := bind.Uint32(0); got != pointerGlobalName {
		t.Errorf("bind name = %d, want %d", got, pointerGlobalName)
	}
	iface, consumed := bind.String(4)
	if iface != protocols.VirtualPointerManagerInterface {
		t.Errorf("bind interface = %q, want %q", iface, protocols.VirtualPointerManagerInterface)
	}
	if got := bind.Uint32(4 + consumed); got != 1 {
		t.Errorf("bind version = %d, want 1", got)
	}
	bindID := bind.Uint32(8 + consumed)
	if bindID == 0 {
		t.Fatal("bind used object ID 0")
	}
	if got := srv.ObjectID(protocols.VirtualPointerManagerInterface); got != bindID {
		t.Errorf("bound object ID = %d, want %d", got, bindID)
	}
}

func TestManagerFailsWithoutVirtualPointerGlobal(t *testing.T) {
	srv := testcompositor.Start(t,
		testcompositor.Global{Name: seatGlobalName, Interface: "wl_seat", Version: 5},
	)
	srv.Env(t)

	manager, err := NewVirtualPointerManager(context.Background())
	if err == nil {
		_ = manager.Close()
		t.Fatal("NewVirtualPointerManager succeeded without zwlr_virtual_pointer_manager_v1")
	}
	if got := err.Error(); got != "zwlr_virtual_pointer_manager_v1 not available" {
		t.Fatalf("error = %q, want the missing-global message", got)
	}
}

func TestManagerFailsOnCancelledContext(t *testing.T) {
	srv := newCompositor(t)
	_ = srv

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	manager, err := NewVirtualPointerManager(ctx)
	if err == nil {
		_ = manager.Close()
		t.Fatal("NewVirtualPointerManager succeeded with a cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestCreatePointerSendsSeatAndNewID(t *testing.T) {
	srv := newCompositor(t)
	manager := newManager(t, srv)

	pointer, err := manager.CreatePointer()
	if err != nil {
		t.Fatalf("CreatePointer: %v", err)
	}

	managerID := srv.ObjectID(protocols.VirtualPointerManagerInterface)
	req := waitForRequest(t, srv, managerID, 0)

	seatID := srv.ObjectID("wl_seat")
	if seatID == 0 {
		t.Fatal("client never bound wl_seat")
	}
	if got := req.Uint32(0); got != seatID {
		t.Errorf("create_virtual_pointer seat = %d, want %d", got, seatID)
	}

	pointerID := req.Uint32(4)
	if pointerID == 0 || pointerID == seatID || pointerID == managerID {
		t.Fatalf("create_virtual_pointer new_id = %d, want a fresh object ID", pointerID)
	}

	// Destroy must target the object that was created.
	if err := pointer.Close(); err != nil {
		t.Fatalf("pointer.Close: %v", err)
	}
	destroy := waitForRequest(t, srv, pointerID, 8)
	if len(destroy.Body) != 0 {
		t.Errorf("destroy carried %d bytes of arguments, want none", len(destroy.Body))
	}
}

func TestPointerEventsUseProtocolOpcodesAndArguments(t *testing.T) {
	srv := newCompositor(t)
	manager := newManager(t, srv)

	pointer, err := manager.CreatePointer()
	if err != nil {
		t.Fatalf("CreatePointer: %v", err)
	}

	managerID := srv.ObjectID(protocols.VirtualPointerManagerInterface)
	pointerID := waitForRequest(t, srv, managerID, 0).Uint32(4)

	when := time.UnixMilli(1234)

	if err := pointer.Motion(when, 1.5, -2.0); err != nil {
		t.Fatalf("Motion: %v", err)
	}
	motion := waitForRequest(t, srv, pointerID, 0)
	if got := motion.Uint32(0); got != 1234 {
		t.Errorf("motion time = %d, want 1234", got)
	}
	if got := int32(motion.Uint32(4)); got != 384 {
		t.Errorf("motion dx = %d, want 384 (1.5 in 24.8 fixed point)", got)
	}
	if got := int32(motion.Uint32(8)); got != -512 {
		t.Errorf("motion dy = %d, want -512 (-2.0 in 24.8 fixed point)", got)
	}

	if err := pointer.MotionAbsolute(when, 10, 20, 1920, 1080); err != nil {
		t.Fatalf("MotionAbsolute: %v", err)
	}
	absolute := waitForRequest(t, srv, pointerID, 1)
	wantAbsolute := []uint32{1234, 10, 20, 1920, 1080}
	for i, want := range wantAbsolute {
		if got := absolute.Uint32(i * 4); got != want {
			t.Errorf("motion_absolute arg %d = %d, want %d", i, got, want)
		}
	}

	if err := pointer.Button(when, BTN_LEFT, ButtonStatePressed); err != nil {
		t.Fatalf("Button: %v", err)
	}
	button := waitForRequest(t, srv, pointerID, 2)
	if got := button.Uint32(4); got != BTN_LEFT {
		t.Errorf("button = %#x, want %#x", got, BTN_LEFT)
	}
	if got := button.Uint32(8); got != BUTTON_STATE_PRESSED {
		t.Errorf("button state = %d, want %d", got, BUTTON_STATE_PRESSED)
	}

	if err := pointer.Axis(when, AxisVertical, 2.0); err != nil {
		t.Fatalf("Axis: %v", err)
	}
	axis := waitForRequest(t, srv, pointerID, 3)
	if got := axis.Uint32(4); got != uint32(AxisVertical) {
		t.Errorf("axis = %d, want %d", got, AxisVertical)
	}
	if got := int32(axis.Uint32(8)); got != 512 {
		t.Errorf("axis value = %d, want 512", got)
	}

	if err := pointer.Frame(); err != nil {
		t.Fatalf("Frame: %v", err)
	}
	frame := waitForRequest(t, srv, pointerID, 4)
	if len(frame.Body) != 0 {
		t.Errorf("frame carried %d bytes of arguments, want none", len(frame.Body))
	}

	if err := pointer.AxisSource(AxisSourceWheel); err != nil {
		t.Fatalf("AxisSource: %v", err)
	}
	source := waitForRequest(t, srv, pointerID, 5)
	if got := source.Uint32(0); got != uint32(AxisSourceWheel) {
		t.Errorf("axis source = %d, want %d", got, AxisSourceWheel)
	}

	if err := pointer.AxisStop(when, AxisVertical); err != nil {
		t.Fatalf("AxisStop: %v", err)
	}
	stop := waitForRequest(t, srv, pointerID, 6)
	if got := stop.Uint32(4); got != uint32(AxisVertical) {
		t.Errorf("axis stop axis = %d, want %d", got, AxisVertical)
	}

	if err := pointer.AxisDiscrete(when, AxisHorizontal, 1.0, 3); err != nil {
		t.Fatalf("AxisDiscrete: %v", err)
	}
	discrete := waitForRequest(t, srv, pointerID, 7)
	if got := discrete.Uint32(4); got != uint32(AxisHorizontal) {
		t.Errorf("axis discrete axis = %d, want %d", got, AxisHorizontal)
	}
	if got := int32(discrete.Uint32(8)); got != 256 {
		t.Errorf("axis discrete value = %d, want 256", got)
	}
	if got := int32(discrete.Uint32(12)); got != 3 {
		t.Errorf("axis discrete steps = %d, want 3", got)
	}
}

func TestConvenienceHelpersEmitFrame(t *testing.T) {
	srv := newCompositor(t)
	manager := newManager(t, srv)

	pointer, err := manager.CreatePointer()
	if err != nil {
		t.Fatalf("CreatePointer: %v", err)
	}

	managerID := srv.ObjectID(protocols.VirtualPointerManagerInterface)
	pointerID := waitForRequest(t, srv, managerID, 0).Uint32(4)

	if err := pointer.LeftClick(); err != nil {
		t.Fatalf("LeftClick: %v", err)
	}
	waitForRequest(t, srv, pointerID, 4) // frame

	var press, release bool
	for _, req := range srv.Requests() {
		if req.Object != pointerID || req.Opcode != 2 {
			continue
		}
		switch req.Uint32(8) {
		case BUTTON_STATE_PRESSED:
			press = true
		case BUTTON_STATE_RELEASED:
			release = true
		}
	}
	if !press || !release {
		t.Fatalf("LeftClick press=%v release=%v, want both", press, release)
	}

	if err := pointer.ScrollVertical(-1.0); err != nil {
		t.Fatalf("ScrollVertical: %v", err)
	}
	axis := waitForRequest(t, srv, pointerID, 3)
	if got := int32(axis.Uint32(8)); got != -256 {
		t.Errorf("scroll value = %d, want -256", got)
	}
}

func TestManagerCloseDestroysManager(t *testing.T) {
	srv := newCompositor(t)
	manager := newManager(t, srv)

	if err := manager.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	managerID := srv.ObjectID(protocols.VirtualPointerManagerInterface)
	waitForRequest(t, srv, managerID, 1) // destroy
}

// A compositor error must surface through the connection instead of being
// swallowed by the protocol layer.
func TestDisplayErrorPropagates(t *testing.T) {
	srv := newCompositor(t)
	manager := newManager(t, srv)

	managerID := srv.ObjectID(protocols.VirtualPointerManagerInterface)
	if err := srv.SendDisplayError(managerID, 1, "denied"); err != nil {
		t.Fatalf("SendDisplayError: %v", err)
	}

	// The error is queued before the roundtrip's sync reply, so the roundtrip
	// must surface it rather than reporting success.
	err := manager.client.GetDisplay().Roundtrip()
	if err == nil {
		t.Fatal("Roundtrip succeeded, want the compositor error")
	}

	var displayErr *wlturbo.DisplayError
	if !errors.As(err, &displayErr) {
		t.Fatalf("error = %v (%T), want *wlturbo.DisplayError", err, err)
	}
	if displayErr.ObjectID != managerID || displayErr.Code != 1 || displayErr.Message != "denied" {
		t.Fatalf("DisplayError = %+v, want object=%d code=1 message=denied", displayErr, managerID)
	}
}
