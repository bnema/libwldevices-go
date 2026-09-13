package keyboard_shortcuts_inhibitor

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bnema/libwldevices-go/internal/protocols"
	"github.com/bnema/libwldevices-go/internal/testcompositor"
	"github.com/bnema/wlturbo"
	"github.com/bnema/wlturbo/wl"
)

// Globals announced by the test compositor. The manager name mirrors the value
// the headless sway fixture uses for zwp_keyboard_shortcuts_inhibit_manager_v1.
const (
	inhibitGlobalName    = 42
	seatGlobalName       = 2
	compositorGlobalName = 3
)

// newCompositor starts a compositor announcing the globals the keyboard
// shortcuts inhibit protocol needs and points the client library at it.
func newCompositor(t *testing.T) *testcompositor.Server {
	t.Helper()

	srv := testcompositor.Start(t,
		testcompositor.Global{Name: seatGlobalName, Interface: "wl_seat", Version: 7},
		testcompositor.Global{Name: compositorGlobalName, Interface: "wl_compositor", Version: 6},
		testcompositor.Global{
			Name:      inhibitGlobalName,
			Interface: protocols.KeyboardShortcutsInhibitManagerInterface,
			Version:   1,
		},
	)
	srv.Env(t)
	return srv
}

// newManager creates a manager against the test compositor.
func newManager(t *testing.T, srv *testcompositor.Server) *KeyboardShortcutsInhibitorManager {
	t.Helper()

	manager, err := NewKeyboardShortcutsInhibitorManager(context.Background())
	if err != nil {
		t.Fatalf("NewKeyboardShortcutsInhibitorManager: %v", err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	return manager
}

// newSurfaceSeat binds wl_compositor, creates a surface through it and returns
// the seat the client bound from the wl_seat global.
func newSurfaceSeat(t *testing.T, manager *KeyboardShortcutsInhibitorManager) (*wl.Surface, *wl.Seat) {
	t.Helper()

	ctx := manager.client.GetContext()

	compositorID, err := manager.client.GetRegistry().BindID(compositorGlobalName, "wl_compositor", 6)
	if err != nil {
		t.Fatalf("bind wl_compositor: %v", err)
	}
	compositor := wl.NewCompositor(ctx)
	compositor.SetID(compositorID)
	ctx.Register(compositor)

	surface, err := compositor.CreateSurface()
	if err != nil {
		t.Fatalf("CreateSurface: %v", err)
	}

	seat := manager.client.GetSeat()
	if seat == nil {
		t.Fatal("client did not bind wl_seat")
	}
	return surface, seat
}

// waitForRequests waits until at least n requests on one object and opcode were
// recorded, then returns them in wire order.
func waitForRequests(t *testing.T, srv *testcompositor.Server, object uint32, opcode uint16, n int) []testcompositor.Request {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		var matched []testcompositor.Request
		for _, req := range srv.Requests() {
			if req.Object == object && req.Opcode == opcode {
				matched = append(matched, req)
			}
		}
		if len(matched) >= n {
			return matched
		}
		if time.Now().After(deadline) {
			t.Fatalf("waited for %d requests object=%d opcode=%d, saw %d: %+v", n, object, opcode, len(matched), matched)
		}
		time.Sleep(time.Millisecond)
	}
}

// waitForRequest waits for a request on one object and opcode.
func waitForRequest(t *testing.T, srv *testcompositor.Server, object uint32, opcode uint16) testcompositor.Request {
	t.Helper()

	return waitForRequests(t, srv, object, opcode, 1)[0]
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

// boundObject waits until the interface has been bound, then returns the
// object ID the client used in that bind request.
func boundObject(t *testing.T, srv *testcompositor.Server, iface string) uint32 {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if id := srv.ObjectID(iface); id != 0 {
			return id
		}
		if time.Now().After(deadline) {
			t.Fatalf("interface %q was never bound", iface)
		}
		time.Sleep(time.Millisecond)
	}
}

// waitForEvent pumps roundtrips until cond reports true or the deadline expires.
func waitForEvent(t *testing.T, manager *KeyboardShortcutsInhibitorManager, cond func() bool, what string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if cond() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		_ = manager.Roundtrip()
		time.Sleep(time.Millisecond)
	}
}

func TestManagerBindsKeyboardShortcutsInhibitGlobal(t *testing.T) {
	srv := newCompositor(t)
	newManager(t, srv)

	bind := waitForBind(t, srv, protocols.KeyboardShortcutsInhibitManagerInterface)

	if got := bind.Uint32(0); got != inhibitGlobalName {
		t.Errorf("bind name = %d, want %d", got, inhibitGlobalName)
	}
	iface, consumed := bind.String(4)
	if iface != protocols.KeyboardShortcutsInhibitManagerInterface {
		t.Errorf("bind interface = %q, want %q", iface, protocols.KeyboardShortcutsInhibitManagerInterface)
	}
	if got := bind.Uint32(4 + consumed); got != 1 {
		t.Errorf("bind version = %d, want 1", got)
	}
	bindID := bind.Uint32(8 + consumed)
	if bindID == 0 {
		t.Fatal("bind used object ID 0")
	}
	if got := boundObject(t, srv, protocols.KeyboardShortcutsInhibitManagerInterface); got != bindID {
		t.Errorf("bound object ID = %d, want %d", got, bindID)
	}
}

func TestManagerFailsWithoutKeyboardShortcutsInhibitGlobal(t *testing.T) {
	srv := testcompositor.Start(t)
	srv.Env(t)

	manager, err := NewKeyboardShortcutsInhibitorManager(context.Background())
	if err == nil {
		_ = manager.Close()
		t.Fatal("NewKeyboardShortcutsInhibitorManager succeeded without zwp_keyboard_shortcuts_inhibit_manager_v1")
	}
	want := "zwp_keyboard_shortcuts_inhibit_manager_v1 not available - compositor may not support keyboard-shortcuts-inhibit protocol"
	if got := err.Error(); got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestManagerFailsOnCancelledContext(t *testing.T) {
	srv := newCompositor(t)
	_ = srv

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	manager, err := NewKeyboardShortcutsInhibitorManager(ctx)
	if err == nil {
		_ = manager.Close()
		t.Fatal("NewKeyboardShortcutsInhibitorManager succeeded with a cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

// inhibit_shortcuts is declared as (id, surface, seat) in the protocol XML.
func TestInhibitShortcutsRequestEncoding(t *testing.T) {
	srv := newCompositor(t)
	manager := newManager(t, srv)
	surface, seat := newSurfaceSeat(t, manager)

	managerID := boundObject(t, srv, protocols.KeyboardShortcutsInhibitManagerInterface)

	inhibitor, err := manager.InhibitShortcuts(surface, seat)
	if err != nil {
		t.Fatalf("InhibitShortcuts: %v", err)
	}

	// inhibit_shortcuts is the second manager request (opcode 1).
	req := waitForRequest(t, srv, managerID, 1)
	childID := req.Uint32(0)
	if childID == 0 {
		t.Fatal("inhibit_shortcuts new_id = 0, want a fresh object ID")
	}
	if got := req.Uint32(4); got != surface.ID() {
		t.Errorf("inhibit_shortcuts surface = %d, want %d", got, surface.ID())
	}
	if got := req.Uint32(8); got != seat.ID() {
		t.Errorf("inhibit_shortcuts seat = %d, want %d", got, seat.ID())
	}

	// The inhibitor's destroy is the object's first request (opcode 0).
	if err := inhibitor.Destroy(); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
	destroy := waitForRequest(t, srv, childID, 0)
	if len(destroy.Body) != 0 {
		t.Errorf("destroy carried %d bytes of arguments, want none", len(destroy.Body))
	}

	if err := inhibitor.Destroy(); err == nil {
		t.Fatal("second Destroy succeeded, want an error")
	}
}

func TestInhibitorActiveAndInactiveEvents(t *testing.T) {
	srv := newCompositor(t)
	manager := newManager(t, srv)
	surface, seat := newSurfaceSeat(t, manager)

	inhibitor, err := manager.InhibitShortcuts(surface, seat)
	if err != nil {
		t.Fatalf("InhibitShortcuts: %v", err)
	}
	if inhibitor.Active() {
		t.Fatal("inhibitor reports active before the compositor sent the active event")
	}

	var activated, deactivated atomic.Int32
	inhibitor.OnActive(func() { activated.Add(1) })
	inhibitor.OnInactive(func() { deactivated.Add(1) })

	managerID := boundObject(t, srv, protocols.KeyboardShortcutsInhibitManagerInterface)
	childID := waitForRequest(t, srv, managerID, 1).Uint32(0)

	// active (opcode 0)
	if err := srv.SendEvent(childID, 0); err != nil {
		t.Fatalf("SendEvent(active): %v", err)
	}
	waitForEvent(t, manager, func() bool {
		return inhibitor.Active() && activated.Load() == 1
	}, "the active event to be delivered")

	// inactive (opcode 1)
	if err := srv.SendEvent(childID, 1); err != nil {
		t.Fatalf("SendEvent(inactive): %v", err)
	}
	waitForEvent(t, manager, func() bool {
		return !inhibitor.Active() && deactivated.Load() == 1
	}, "the inactive event to be delivered")
}

func TestManagerCloseSendsDestroy(t *testing.T) {
	srv := newCompositor(t)
	manager := newManager(t, srv)

	managerID := boundObject(t, srv, protocols.KeyboardShortcutsInhibitManagerInterface)

	if err := manager.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// destroy is the manager's first request (opcode 0).
	destroy := waitForRequest(t, srv, managerID, 0)
	if len(destroy.Body) != 0 {
		t.Errorf("destroy carried %d bytes of arguments, want none", len(destroy.Body))
	}

	if err := manager.Close(); err == nil {
		t.Fatal("second Close succeeded, want an error")
	}
}

// A compositor protocol error (already_inhibited) must surface to the caller
// instead of being swallowed.
func TestDisplayErrorPropagates(t *testing.T) {
	srv := newCompositor(t)
	manager := newManager(t, srv)
	surface, seat := newSurfaceSeat(t, manager)

	if _, err := manager.InhibitShortcuts(surface, seat); err != nil {
		t.Fatalf("InhibitShortcuts: %v", err)
	}

	managerID := boundObject(t, srv, protocols.KeyboardShortcutsInhibitManagerInterface)
	childID := waitForRequest(t, srv, managerID, 1).Uint32(0)

	if err := srv.SendDisplayError(childID, uint32(ERROR_ALREADY_INHIBITED), "already inhibited"); err != nil {
		t.Fatalf("SendDisplayError: %v", err)
	}

	err := manager.Roundtrip()
	if err == nil {
		t.Fatal("Roundtrip succeeded, want the compositor protocol error")
	}

	var displayErr *wlturbo.DisplayError
	if !errors.As(err, &displayErr) {
		t.Fatalf("error = %v (%T), want *wlturbo.DisplayError", err, err)
	}
	if displayErr.ObjectID != childID {
		t.Errorf("DisplayError.ObjectID = %d, want %d", displayErr.ObjectID, childID)
	}
	if displayErr.Code != uint32(ERROR_ALREADY_INHIBITED) {
		t.Errorf("DisplayError.Code = %d, want %d", displayErr.Code, ERROR_ALREADY_INHIBITED)
	}
	if displayErr.Message != "already inhibited" {
		t.Errorf("DisplayError.Message = %q, want %q", displayErr.Message, "already inhibited")
	}
}

func TestInhibitShortcutsWithNilArguments(t *testing.T) {
	srv := newCompositor(t)
	manager := newManager(t, srv)
	surface, seat := newSurfaceSeat(t, manager)

	tests := []struct {
		name    string
		surface *wl.Surface
		seat    *wl.Seat
		want    string
	}{
		{name: "surface", surface: nil, seat: seat, want: "keyboard shortcuts inhibitor error -1: surface cannot be nil"},
		{name: "seat", surface: surface, seat: nil, want: "keyboard shortcuts inhibitor error -1: seat cannot be nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.InhibitShortcuts(tt.surface, tt.seat)
			if err == nil {
				t.Fatal("InhibitShortcuts succeeded, want an error")
			}
			if got := err.Error(); got != tt.want {
				t.Fatalf("error = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestManagerNotConnected(t *testing.T) {
	manager := &KeyboardShortcutsInhibitorManager{}

	_, err := manager.InhibitShortcuts(nil, nil)
	if err == nil {
		t.Fatal("InhibitShortcuts succeeded on an unconnected manager")
	}
	if got := err.Error(); got != "keyboard shortcuts inhibitor error -1: manager not connected" {
		t.Fatalf("error = %q, want the not-connected message", got)
	}

	if err := manager.Roundtrip(); err == nil {
		t.Fatal("Roundtrip succeeded on an unconnected manager")
	}
}

func TestInhibitShortcutsAfterManagerDestroy(t *testing.T) {
	srv := newCompositor(t)
	manager := newManager(t, srv)
	surface, seat := newSurfaceSeat(t, manager)

	if err := manager.Destroy(); err != nil {
		t.Fatalf("Destroy: %v", err)
	}

	if _, err := manager.InhibitShortcuts(surface, seat); err == nil {
		t.Fatal("InhibitShortcuts succeeded after the manager was destroyed")
	}
}

func TestCreateTemporaryInhibitor(t *testing.T) {
	srv := newCompositor(t)
	manager := newManager(t, srv)
	surface, seat := newSurfaceSeat(t, manager)

	inhibitor, err := CreateTemporaryInhibitor(manager, surface, seat)
	if err != nil {
		t.Fatalf("CreateTemporaryInhibitor: %v", err)
	}
	if inhibitor == nil {
		t.Fatal("CreateTemporaryInhibitor returned nil")
	}

	status := GetStatus(inhibitor)
	if status.Surface != surface || status.Seat != seat {
		t.Fatalf("GetStatus surface/seat = %v/%v, want %v/%v", status.Surface, status.Seat, surface, seat)
	}
	if status.Active {
		t.Fatal("GetStatus reports active before the compositor sent the active event")
	}

	if err := inhibitor.Destroy(); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
}

func TestGetStatusNilInhibitor(t *testing.T) {
	if status := GetStatus(nil); status.Active {
		t.Fatal("GetStatus(nil) should report inactive")
	}
}

// Keep the protocol constant/error cases the package already exposed.
func TestErrorConstants(t *testing.T) {
	if ERROR_ALREADY_INHIBITED != protocols.ERROR_ALREADY_INHIBITED {
		t.Errorf("ERROR_ALREADY_INHIBITED = %d, want %d", ERROR_ALREADY_INHIBITED, protocols.ERROR_ALREADY_INHIBITED)
	}
	if ERROR_ALREADY_INHIBITED != 0 {
		t.Errorf("ERROR_ALREADY_INHIBITED = %d, want 0 per keyboard-shortcuts-inhibit-unstable-v1", ERROR_ALREADY_INHIBITED)
	}
}

func TestKeyboardShortcutsInhibitorError(t *testing.T) {
	err := &KeyboardShortcutsInhibitorError{
		Code:    int(ERROR_ALREADY_INHIBITED),
		Message: "test error",
	}

	const expected = "keyboard shortcuts inhibitor error 0: test error"
	if err.Error() != expected {
		t.Errorf("error = %q, want %q", err.Error(), expected)
	}
}
