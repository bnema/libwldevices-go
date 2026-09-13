package virtual_keyboard

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
	seatGlobalName     = 1
	keyboardGlobalName = 2
)

// newKeyboardCompositor starts a compositor announcing the globals the virtual
// keyboard protocol needs and points the client library at it.
func newKeyboardCompositor(t *testing.T) *testcompositor.Server {
	t.Helper()

	srv := testcompositor.Start(t,
		testcompositor.Global{Name: seatGlobalName, Interface: "wl_seat", Version: 5},
		testcompositor.Global{Name: keyboardGlobalName, Interface: protocols.VirtualKeyboardManagerInterface, Version: 1},
	)
	srv.Env(t)
	return srv
}

// newManager creates a manager against the test compositor.
func newManager(t *testing.T, srv *testcompositor.Server) *VirtualKeyboardManager {
	t.Helper()

	manager, err := NewVirtualKeyboardManager(context.Background())
	if err != nil {
		t.Fatalf("NewVirtualKeyboardManager: %v", err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	return manager
}

// newKeyboard creates a virtual keyboard against the test compositor.
func newKeyboard(t *testing.T, manager *VirtualKeyboardManager) *VirtualKeyboard {
	t.Helper()

	keyboard, err := manager.CreateKeyboard()
	if err != nil {
		t.Fatalf("CreateKeyboard: %v", err)
	}
	t.Cleanup(func() { _ = keyboard.Close() })
	return keyboard
}

// waitForRequest waits for a request on one object and opcode.
func waitForRequest(t *testing.T, srv *testcompositor.Server, object uint32, opcode uint16) testcompositor.Request {
	t.Helper()

	requests := waitForRequests(t, srv, object, opcode, 1)
	return requests[0]
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

func TestManagerBindsVirtualKeyboardGlobal(t *testing.T) {
	srv := newKeyboardCompositor(t)
	newManager(t, srv)

	bind := waitForBind(t, srv, protocols.VirtualKeyboardManagerInterface)

	if got := bind.Uint32(0); got != keyboardGlobalName {
		t.Errorf("bind name = %d, want %d", got, keyboardGlobalName)
	}
	iface, consumed := bind.String(4)
	if iface != protocols.VirtualKeyboardManagerInterface {
		t.Errorf("bind interface = %q, want %q", iface, protocols.VirtualKeyboardManagerInterface)
	}
	if got := bind.Uint32(4 + consumed); got != 1 {
		t.Errorf("bind version = %d, want 1", got)
	}
	bindID := bind.Uint32(8 + consumed)
	if bindID == 0 {
		t.Fatal("bind used object ID 0")
	}
	if got := srv.ObjectID(protocols.VirtualKeyboardManagerInterface); got != bindID {
		t.Errorf("bound object ID = %d, want %d", got, bindID)
	}
}

func TestManagerFailsWithoutVirtualKeyboardGlobal(t *testing.T) {
	srv := testcompositor.Start(t,
		testcompositor.Global{Name: seatGlobalName, Interface: "wl_seat", Version: 5},
	)
	srv.Env(t)

	manager, err := NewVirtualKeyboardManager(context.Background())
	if err == nil {
		_ = manager.Close()
		t.Fatal("NewVirtualKeyboardManager succeeded without zwp_virtual_keyboard_manager_v1")
	}
	if got := err.Error(); got != "zwp_virtual_keyboard_manager_v1 not available" {
		t.Fatalf("error = %q, want the missing-global message", got)
	}
}

func TestManagerFailsOnCancelledContext(t *testing.T) {
	srv := newKeyboardCompositor(t)
	_ = srv

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	manager, err := NewVirtualKeyboardManager(ctx)
	if err == nil {
		_ = manager.Close()
		t.Fatal("NewVirtualKeyboardManager succeeded with a cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestCreateKeyboardSendsSeatAndNewID(t *testing.T) {
	srv := newKeyboardCompositor(t)
	manager := newManager(t, srv)

	keyboard := newKeyboard(t, manager)

	managerID := srv.ObjectID(protocols.VirtualKeyboardManagerInterface)
	req := waitForRequest(t, srv, managerID, 0)

	seatID := srv.ObjectID("wl_seat")
	if seatID == 0 {
		t.Fatal("client never bound wl_seat")
	}
	if got := req.Uint32(0); got != seatID {
		t.Errorf("create_virtual_keyboard seat = %d, want %d", got, seatID)
	}

	keyboardID := req.Uint32(4)
	if keyboardID == 0 || keyboardID == seatID || keyboardID == managerID {
		t.Fatalf("create_virtual_keyboard new_id = %d, want a fresh object ID", keyboardID)
	}

	// Destroy must target the object that was created.
	if err := keyboard.Close(); err != nil {
		t.Fatalf("keyboard.Close: %v", err)
	}
	destroy := waitForRequest(t, srv, keyboardID, 3)
	if len(destroy.Body) != 0 {
		t.Errorf("destroy carried %d bytes of arguments, want none", len(destroy.Body))
	}
}

func TestKeymapIsSentOutOfBand(t *testing.T) {
	srv := newKeyboardCompositor(t)
	manager := newManager(t, srv)
	newKeyboard(t, manager)

	managerID := srv.ObjectID(protocols.VirtualKeyboardManagerInterface)
	keyboardID := waitForRequest(t, srv, managerID, 0).Uint32(4)

	keymap := waitForRequest(t, srv, keyboardID, 0)
	// The keymap request carries format and size in the message; the keymap
	// bytes travel out of band in a SCM_RIGHTS descriptor, so the body must not
	// contain an fd placeholder.
	if len(keymap.Body) != 8 {
		t.Errorf("keymap body = %d bytes, want 8 (format + size only)", len(keymap.Body))
	}
	if got := keymap.Uint32(0); got != KEYMAP_FORMAT_XKB_V1 {
		t.Errorf("keymap format = %d, want %d", got, KEYMAP_FORMAT_XKB_V1)
	}
	if got := keymap.Uint32(4); got == 0 {
		t.Error("keymap size = 0, want the mapped keymap size")
	}
}

func TestKeyboardEventsUseProtocolOpcodesAndArguments(t *testing.T) {
	srv := newKeyboardCompositor(t)
	manager := newManager(t, srv)
	keyboard := newKeyboard(t, manager)

	managerID := srv.ObjectID(protocols.VirtualKeyboardManagerInterface)
	keyboardID := waitForRequest(t, srv, managerID, 0).Uint32(4)

	when := time.UnixMilli(1234)

	if err := keyboard.Key(when, KEY_A, KeyStatePressed); err != nil {
		t.Fatalf("Key: %v", err)
	}
	press := waitForRequest(t, srv, keyboardID, 1)
	if got := press.Uint32(0); got != 1234 {
		t.Errorf("key time = %d, want 1234", got)
	}
	if got := press.Uint32(4); got != KEY_A {
		t.Errorf("key code = %d, want %d", got, KEY_A)
	}
	if got := press.Uint32(8); got != KEY_STATE_PRESSED {
		t.Errorf("key state = %d, want %d", got, KEY_STATE_PRESSED)
	}

	if err := keyboard.Key(when, KEY_A, KeyStateReleased); err != nil {
		t.Fatalf("Key: %v", err)
	}
	release := waitForRequests(t, srv, keyboardID, 1, 2)[1]
	if got := release.Uint32(4); got != KEY_A {
		t.Errorf("release key code = %d, want %d", got, KEY_A)
	}
	if got := release.Uint32(8); got != KEY_STATE_RELEASED {
		t.Errorf("release key state = %d, want %d", got, KEY_STATE_RELEASED)
	}

	if err := keyboard.Modifiers(1|4, 0, 0, 0); err != nil {
		t.Fatalf("Modifiers: %v", err)
	}
	modifiers := waitForRequest(t, srv, keyboardID, 2)
	wantModifiers := []uint32{1 | 4, 0, 0, 0}
	for i, want := range wantModifiers {
		if got := modifiers.Uint32(i * 4); got != want {
			t.Errorf("modifiers arg %d = %d, want %d", i, got, want)
		}
	}

	if err := keyboard.PressKey(KEY_B); err != nil {
		t.Fatalf("PressKey: %v", err)
	}
	if err := keyboard.ReleaseKey(KEY_B); err != nil {
		t.Fatalf("ReleaseKey: %v", err)
	}
	if err := keyboard.TypeKey(KEY_C); err != nil {
		t.Fatalf("TypeKey: %v", err)
	}

	keys := waitForRequests(t, srv, keyboardID, 1, 6)
	wantKeys := []struct {
		key   uint32
		state uint32
	}{
		{KEY_A, KEY_STATE_PRESSED},
		{KEY_A, KEY_STATE_RELEASED},
		{KEY_B, KEY_STATE_PRESSED},
		{KEY_B, KEY_STATE_RELEASED},
		{KEY_C, KEY_STATE_PRESSED},
		{KEY_C, KEY_STATE_RELEASED},
	}
	for i, want := range wantKeys {
		if got := keys[i].Uint32(4); got != want.key {
			t.Errorf("key %d code = %d, want %d", i, got, want.key)
		}
		if got := keys[i].Uint32(8); got != want.state {
			t.Errorf("key %d state = %d, want %d", i, got, want.state)
		}
	}
}

func TestTypeStringPressesShiftForUppercase(t *testing.T) {
	srv := newKeyboardCompositor(t)
	manager := newManager(t, srv)
	keyboard := newKeyboard(t, manager)

	managerID := srv.ObjectID(protocols.VirtualKeyboardManagerInterface)
	keyboardID := waitForRequest(t, srv, managerID, 0).Uint32(4)

	if err := keyboard.TypeString("Hi"); err != nil {
		t.Fatalf("TypeString: %v", err)
	}

	keys := waitForRequests(t, srv, keyboardID, 1, 6)
	wantKeys := []struct {
		key   uint32
		state uint32
	}{
		{KEY_LEFTSHIFT, KEY_STATE_PRESSED},
		{KEY_H, KEY_STATE_PRESSED},
		{KEY_H, KEY_STATE_RELEASED},
		{KEY_LEFTSHIFT, KEY_STATE_RELEASED},
		{KEY_I, KEY_STATE_PRESSED},
		{KEY_I, KEY_STATE_RELEASED},
	}
	for i, want := range wantKeys {
		if got := keys[i].Uint32(4); got != want.key {
			t.Errorf("key %d code = %d, want %d", i, got, want.key)
		}
		if got := keys[i].Uint32(8); got != want.state {
			t.Errorf("key %d state = %d, want %d", i, got, want.state)
		}
	}
}

func TestKeyboardWithoutKeymapRejectsEvents(t *testing.T) {
	keyboard := &VirtualKeyboard{keyboard: &protocols.VirtualKeyboard{}}

	if err := keyboard.Key(time.Now(), KEY_A, KeyStatePressed); err == nil {
		t.Error("Key succeeded without a keymap")
	} else if got := err.Error(); got != "keymap not set" {
		t.Errorf("Key error = %q, want %q", got, "keymap not set")
	}

	if err := keyboard.Modifiers(1, 0, 0, 0); err == nil {
		t.Error("Modifiers succeeded without a keymap")
	} else if got := err.Error(); got != "keymap not set" {
		t.Errorf("Modifiers error = %q, want %q", got, "keymap not set")
	}
}

func TestManagerCloseLeavesNoWireRequest(t *testing.T) {
	srv := newKeyboardCompositor(t)
	manager := newManager(t, srv)

	managerID := srv.ObjectID(protocols.VirtualKeyboardManagerInterface)
	before := len(srv.Requests())

	if err := manager.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// zwp_virtual_keyboard_manager_v1 has no destroy request, so closing the
	// manager must not emit any protocol traffic.
	time.Sleep(20 * time.Millisecond)
	if got := len(srv.Requests()); got != before {
		t.Errorf("manager close emitted %d extra requests, want none (manager has no destroy request)", got-before)
	}
	for _, req := range srv.Requests() {
		if req.Object == managerID {
			t.Errorf("unexpected request on manager object: %+v", req)
		}
	}
}

// A compositor error must surface through the connection instead of being
// swallowed by the protocol layer.
func TestDisplayErrorPropagates(t *testing.T) {
	srv := newKeyboardCompositor(t)
	manager := newManager(t, srv)

	managerID := srv.ObjectID(protocols.VirtualKeyboardManagerInterface)
	if err := srv.SendDisplayError(managerID, 1, "denied"); err != nil {
		t.Fatalf("SendDisplayError: %v", err)
	}

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

func TestKeyConstants(t *testing.T) {
	// Test that key constants are defined and have reasonable values
	keys := []struct {
		key   uint32
		name  string
		value uint32
	}{
		{KEY_A, "KEY_A", 30},
		{KEY_Z, "KEY_Z", 44},
		{KEY_0, "KEY_0", 11},
		{KEY_9, "KEY_9", 10},
		{KEY_SPACE, "KEY_SPACE", 57},
		{KEY_ENTER, "KEY_ENTER", 28},
		{KEY_ESC, "KEY_ESC", 1},
		{KEY_LEFTSHIFT, "KEY_LEFTSHIFT", 42},
		{KEY_LEFTCTRL, "KEY_LEFTCTRL", 29},
		{KEY_LEFTALT, "KEY_LEFTALT", 56},
	}

	for _, test := range keys {
		if test.key != test.value {
			t.Fatalf("%s should be %d, got %d", test.name, test.value, test.key)
		}
	}

	// Test key states
	if KEY_STATE_RELEASED != 0 {
		t.Fatal("KEY_STATE_RELEASED should be 0")
	}
	if KEY_STATE_PRESSED != 1 {
		t.Fatal("KEY_STATE_PRESSED should be 1")
	}
}

func TestKeymapFormatConstants(t *testing.T) {
	if KEYMAP_FORMAT_NO_KEYMAP != 0 {
		t.Fatal("KEYMAP_FORMAT_NO_KEYMAP should be 0")
	}
	if KEYMAP_FORMAT_XKB_V1 != 1 {
		t.Fatal("KEYMAP_FORMAT_XKB_V1 should be 1")
	}
}

func TestKeyStateConstants(t *testing.T) {
	// Test that KeyState enum values match the raw constants
	if uint32(KeyStateReleased) != KEY_STATE_RELEASED {
		t.Fatal("KeyStateReleased should equal KEY_STATE_RELEASED")
	}
	if uint32(KeyStatePressed) != KEY_STATE_PRESSED {
		t.Fatal("KeyStatePressed should equal KEY_STATE_PRESSED")
	}
}
