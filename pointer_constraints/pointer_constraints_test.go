package pointer_constraints

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bnema/libwldevices-go/internal/protocols"
	"github.com/bnema/libwldevices-go/internal/testcompositor"
	"github.com/bnema/wlturbo"
	"github.com/bnema/wlturbo/protocol/core"
)

// Global name announced by the test compositor.
const constraintsGlobalName = 1

// newCompositor starts a compositor announcing the globals the pointer
// constraints protocol needs and points the client library at it.
func newCompositor(t *testing.T) *testcompositor.Server {
	t.Helper()

	srv := testcompositor.Start(t, testcompositor.Global{
		Name:      constraintsGlobalName,
		Interface: protocols.PointerConstraintsInterface,
		Version:   1,
	})
	srv.Env(t)
	return srv
}

// newManager creates a manager against the test compositor.
func newManager(t *testing.T, srv *testcompositor.Server) *PointerConstraintsManager {
	t.Helper()

	manager, err := NewPointerConstraintsManager(context.Background())
	if err != nil {
		t.Fatalf("NewPointerConstraintsManager: %v", err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	return manager
}

// newSurfacePointerRegion creates client-side proxies with fresh object IDs so
// the manager has concrete objects to reference in its requests.
func newSurfacePointerRegion(t *testing.T, manager *PointerConstraintsManager) (*core.Surface, *core.Pointer, *core.Region) {
	t.Helper()

	ctx := manager.client.GetContext()

	surface := core.NewSurface(ctx)
	surface.SetID(ctx.AllocateID())
	ctx.Register(surface)

	pointer := &core.Pointer{}
	pointer.SetID(ctx.AllocateID())
	pointer.SetContext(ctx)
	ctx.Register(pointer)

	region := &core.Region{}
	region.SetID(ctx.AllocateID())
	region.SetContext(ctx)
	ctx.Register(region)

	return surface, pointer, region
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

	// The server records a request before it processes it, so poll the bound
	// object map instead of assuming the bind has already been handled.
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

func TestManagerBindsPointerConstraintsGlobal(t *testing.T) {
	srv := newCompositor(t)
	newManager(t, srv)

	bind := waitForBind(t, srv, protocols.PointerConstraintsInterface)

	if got := bind.Uint32(0); got != constraintsGlobalName {
		t.Errorf("bind name = %d, want %d", got, constraintsGlobalName)
	}
	iface, consumed := bind.String(4)
	if iface != protocols.PointerConstraintsInterface {
		t.Errorf("bind interface = %q, want %q", iface, protocols.PointerConstraintsInterface)
	}
	if got := bind.Uint32(4 + consumed); got != 1 {
		t.Errorf("bind version = %d, want 1", got)
	}
	bindID := bind.Uint32(8 + consumed)
	if bindID == 0 {
		t.Fatal("bind used object ID 0")
	}
	if got := boundObject(t, srv, protocols.PointerConstraintsInterface); got != bindID {
		t.Errorf("bound object ID = %d, want %d", got, bindID)
	}
}

func TestManagerFailsWithoutPointerConstraintsGlobal(t *testing.T) {
	srv := testcompositor.Start(t)
	srv.Env(t)

	manager, err := NewPointerConstraintsManager(context.Background())
	if err == nil {
		_ = manager.Close()
		t.Fatal("NewPointerConstraintsManager succeeded without zwp_pointer_constraints_v1")
	}
	if got := err.Error(); got != "zwp_pointer_constraints_v1 not available - compositor may not support pointer-constraints protocol" {
		t.Fatalf("error = %q, want the missing-global message", got)
	}
}

func TestManagerFailsOnCancelledContext(t *testing.T) {
	srv := newCompositor(t)
	_ = srv

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	manager, err := NewPointerConstraintsManager(ctx)
	if err == nil {
		_ = manager.Close()
		t.Fatal("NewPointerConstraintsManager succeeded with a cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestLockPointerRequestEncoding(t *testing.T) {
	srv := newCompositor(t)
	manager := newManager(t, srv)
	surface, pointer, region := newSurfacePointerRegion(t, manager)

	managerID := boundObject(t, srv, protocols.PointerConstraintsInterface)

	locked, err := manager.LockPointer(surface, pointer, region, LifetimePersistent)
	if err != nil {
		t.Fatalf("LockPointer: %v", err)
	}
	_ = locked

	req := waitForRequest(t, srv, managerID, 1)
	if got := req.Uint32(0); got == 0 {
		t.Fatal("lock_pointer new_id = 0, want a fresh object ID")
	}
	if got := req.Uint32(4); got != surface.ID() {
		t.Errorf("lock_pointer surface = %d, want %d", got, surface.ID())
	}
	if got := req.Uint32(8); got != pointer.ID() {
		t.Errorf("lock_pointer pointer = %d, want %d", got, pointer.ID())
	}
	if got := req.Uint32(12); got != region.ID() {
		t.Errorf("lock_pointer region = %d, want %d", got, region.ID())
	}
	if got := req.Uint32(16); got != LIFETIME_PERSISTENT {
		t.Errorf("lock_pointer lifetime = %d, want %d", got, LIFETIME_PERSISTENT)
	}

	// Optional arguments travel as object ID 0 when omitted.
	lockedNil, err := manager.LockPointer(nil, nil, nil, LifetimeOneshot)
	if err != nil {
		t.Fatalf("LockPointer with nil objects: %v", err)
	}
	if lockedNil == nil {
		t.Fatal("LockPointer with nil objects returned a nil locked pointer")
	}
	nilReq := waitForRequests(t, srv, managerID, 1, 2)[1]
	if got := nilReq.Uint32(0); got == 0 {
		t.Error("lock_pointer new_id = 0 for the nil-argument variant")
	}
	for _, offset := range []int{4, 8, 12} {
		if got := nilReq.Uint32(offset); got != 0 {
			t.Errorf("lock_pointer optional arg at %d = %d, want 0", offset, got)
		}
	}
	if got := nilReq.Uint32(16); got != LIFETIME_ONESHOT {
		t.Errorf("lock_pointer lifetime = %d, want %d", got, LIFETIME_ONESHOT)
	}
}

func TestLockedPointerRequestsUseProtocolOpcodes(t *testing.T) {
	srv := newCompositor(t)
	manager := newManager(t, srv)
	surface, pointer, region := newSurfacePointerRegion(t, manager)

	managerID := boundObject(t, srv, protocols.PointerConstraintsInterface)
	locked, err := manager.LockPointer(surface, pointer, nil, LifetimeOneshot)
	if err != nil {
		t.Fatalf("LockPointer: %v", err)
	}
	lockedID := waitForRequest(t, srv, managerID, 1).Uint32(0)

	if err := locked.SetCursorPositionHint(1.5, -2.0); err != nil {
		t.Fatalf("SetCursorPositionHint: %v", err)
	}
	hint := waitForRequest(t, srv, lockedID, 1)
	if got := int32(hint.Uint32(0)); got != 384 {
		t.Errorf("hint x = %d, want 384 (1.5 in 24.8 fixed point)", got)
	}
	if got := int32(hint.Uint32(4)); got != -512 {
		t.Errorf("hint y = %d, want -512 (-2.0 in 24.8 fixed point)", got)
	}

	if err := locked.SetRegion(region); err != nil {
		t.Fatalf("SetRegion: %v", err)
	}
	setRegion := waitForRequest(t, srv, lockedID, 2)
	if got := setRegion.Uint32(0); got != region.ID() {
		t.Errorf("set_region region = %d, want %d", got, region.ID())
	}

	if err := locked.SetRegion(nil); err != nil {
		t.Fatalf("SetRegion(nil): %v", err)
	}
	nilRegion := waitForRequests(t, srv, lockedID, 2, 2)[1]
	if got := nilRegion.Uint32(0); got != 0 {
		t.Errorf("set_region with nil region = %d, want 0", got)
	}

	if err := locked.Destroy(); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
	destroy := waitForRequest(t, srv, lockedID, 0)
	if len(destroy.Body) != 0 {
		t.Errorf("destroy carried %d bytes of arguments, want none", len(destroy.Body))
	}
}

func TestConfinePointerRequestEncoding(t *testing.T) {
	srv := newCompositor(t)
	manager := newManager(t, srv)
	surface, pointer, region := newSurfacePointerRegion(t, manager)

	managerID := boundObject(t, srv, protocols.PointerConstraintsInterface)

	confined, err := manager.ConfinePointer(surface, pointer, region, LifetimeOneshot)
	if err != nil {
		t.Fatalf("ConfinePointer: %v", err)
	}

	req := waitForRequest(t, srv, managerID, 2)
	confinedID := req.Uint32(0)
	if confinedID == 0 {
		t.Fatal("confine_pointer new_id = 0, want a fresh object ID")
	}
	if got := req.Uint32(4); got != surface.ID() {
		t.Errorf("confine_pointer surface = %d, want %d", got, surface.ID())
	}
	if got := req.Uint32(8); got != pointer.ID() {
		t.Errorf("confine_pointer pointer = %d, want %d", got, pointer.ID())
	}
	if got := req.Uint32(12); got != region.ID() {
		t.Errorf("confine_pointer region = %d, want %d", got, region.ID())
	}
	if got := req.Uint32(16); got != LIFETIME_ONESHOT {
		t.Errorf("confine_pointer lifetime = %d, want %d", got, LIFETIME_ONESHOT)
	}

	if err := confined.SetRegion(region); err != nil {
		t.Fatalf("SetRegion: %v", err)
	}
	setRegion := waitForRequest(t, srv, confinedID, 1)
	if got := setRegion.Uint32(0); got != region.ID() {
		t.Errorf("set_region region = %d, want %d", got, region.ID())
	}

	if err := confined.Destroy(); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
	destroy := waitForRequest(t, srv, confinedID, 0)
	if len(destroy.Body) != 0 {
		t.Errorf("destroy carried %d bytes of arguments, want none", len(destroy.Body))
	}
}

func TestInvalidArgumentTypes(t *testing.T) {
	srv := newCompositor(t)
	manager := newManager(t, srv)
	_, _, region := newSurfacePointerRegion(t, manager)

	tests := []struct {
		name string
		call func() error
		want string
	}{
		{
			name: "surface",
			call: func() error {
				_, err := manager.LockPointer("surface", nil, nil, LifetimeOneshot)
				return err
			},
			want: "pointer constraints error -1: surface must be a *core.Surface",
		},
		{
			name: "pointer",
			call: func() error {
				_, err := manager.ConfinePointer(nil, "pointer", nil, LifetimeOneshot)
				return err
			},
			want: "pointer constraints error -1: pointer must be a *core.Pointer",
		},
		{
			name: "region",
			call: func() error {
				_, err := manager.LockPointer(nil, nil, "region", LifetimeOneshot)
				return err
			},
			want: "pointer constraints error -1: region must be a *core.Region",
		},
		{
			name: "invalid lifetime",
			call: func() error {
				_, err := manager.LockPointer(nil, nil, nil, 999)
				return err
			},
			want: "pointer constraints error -1: invalid lifetime value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil {
				t.Fatal("call succeeded, want an error")
			}
			if got := err.Error(); got != tt.want {
				t.Fatalf("error = %q, want %q", got, tt.want)
			}
		})
	}

	locked, err := manager.LockPointer(nil, nil, nil, LifetimeOneshot)
	if err != nil {
		t.Fatalf("LockPointer: %v", err)
	}
	if err := locked.SetRegion(region); err != nil {
		t.Fatalf("SetRegion: %v", err)
	}
	if err := locked.SetRegion("region"); err == nil {
		t.Error("SetRegion succeeded with a non-region argument")
	} else if got := err.Error(); got != "pointer constraints error -1: region must be a *core.Region" {
		t.Errorf("SetRegion error = %q, want the non-region message", got)
	}

	confined, err := manager.ConfinePointer(nil, nil, nil, LifetimeOneshot)
	if err != nil {
		t.Fatalf("ConfinePointer: %v", err)
	}
	if err := confined.SetRegion("region"); err == nil {
		t.Error("SetRegion succeeded with a non-region argument")
	} else if got := err.Error(); got != "pointer constraints error -1: region must be a *core.Region" {
		t.Errorf("SetRegion error = %q, want the non-region message", got)
	}
}

func TestManagerCloseDestroysManager(t *testing.T) {
	srv := newCompositor(t)
	manager := newManager(t, srv)

	managerID := boundObject(t, srv, protocols.PointerConstraintsInterface)

	if err := manager.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	destroy := waitForRequest(t, srv, managerID, 0)
	if len(destroy.Body) != 0 {
		t.Errorf("destroy carried %d bytes of arguments, want none", len(destroy.Body))
	}
}

// A compositor error must surface through the connection instead of being
// swallowed by the protocol layer.
func TestDisplayErrorPropagates(t *testing.T) {
	srv := newCompositor(t)
	manager := newManager(t, srv)

	managerID := boundObject(t, srv, protocols.PointerConstraintsInterface)
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

// Test lifetime constants
func TestLifetimeConstants(t *testing.T) {
	// Verify that constants have different values
	if LifetimeOneshot == LifetimePersistent {
		t.Fatal("LifetimeOneshot and LifetimePersistent should have different values")
	}

	// Verify correct values
	if LifetimeOneshot != 1 {
		t.Errorf("LifetimeOneshot should be 1, got %d", LifetimeOneshot)
	}

	if LifetimePersistent != 2 {
		t.Errorf("LifetimePersistent should be 2, got %d", LifetimePersistent)
	}

	// Test alternative names
	if LIFETIME_ONESHOT != LifetimeOneshot {
		t.Fatal("LIFETIME_ONESHOT should equal LifetimeOneshot")
	}

	if LIFETIME_PERSISTENT != LifetimePersistent {
		t.Fatal("LIFETIME_PERSISTENT should equal LifetimePersistent")
	}
}

// Test error constants
func TestErrorConstants(t *testing.T) {
	if ERROR_ALREADY_CONSTRAINED != 1 {
		t.Errorf("ERROR_ALREADY_CONSTRAINED should be 1, got %d", ERROR_ALREADY_CONSTRAINED)
	}
}

// Test PointerConstraintsError
func TestPointerConstraintsError(t *testing.T) {
	err := &PointerConstraintsError{
		Code:    ERROR_ALREADY_CONSTRAINED,
		Message: "test error",
	}

	expected := "pointer constraints error 1: test error"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

// Test manager operations with invalid arguments
func TestManagerInvalidArguments(t *testing.T) {
	// Construct a manager whose internal protocol object was never bound.
	manager := &PointerConstraintsManager{}

	// Test LockPointer with nil manager
	_, err := manager.LockPointer(nil, nil, nil, LifetimeOneshot)
	if err == nil {
		t.Fatal("LockPointer should fail with nil internal manager")
	}

	// Test ConfinePointer with nil manager
	_, err = manager.ConfinePointer(nil, nil, nil, LifetimeOneshot)
	if err == nil {
		t.Fatal("ConfinePointer should fail with nil internal manager")
	}

	// Test invalid lifetime
	manager.manager = nil // Ensure it's nil for this test
	_, err = manager.LockPointer(nil, nil, nil, 999)
	if err == nil {
		t.Fatal("LockPointer should fail with invalid lifetime")
	}

	_, err = manager.ConfinePointer(nil, nil, nil, 999)
	if err == nil {
		t.Fatal("ConfinePointer should fail with invalid lifetime")
	}
}

// Test LockedPointer operations with nil pointer
func TestLockedPointerNilOperations(t *testing.T) {
	lp := &LockedPointer{}

	// Test Destroy with nil locked pointer
	err := lp.Destroy()
	if err != nil {
		t.Errorf("Destroy should handle nil locked pointer gracefully, got error: %v", err)
	}

	// Test SetCursorPositionHint with nil locked pointer
	err = lp.SetCursorPositionHint(100.0, 200.0)
	if err == nil {
		t.Fatal("SetCursorPositionHint should fail with nil locked pointer")
	}

	// Test SetRegion with nil locked pointer
	err = lp.SetRegion(nil)
	if err == nil {
		t.Fatal("SetRegion should fail with nil locked pointer")
	}
}

// Test ConfinedPointer operations with nil pointer
func TestConfinedPointerNilOperations(t *testing.T) {
	cp := &ConfinedPointer{}

	// Test Destroy with nil confined pointer
	err := cp.Destroy()
	if err != nil {
		t.Errorf("Destroy should handle nil confined pointer gracefully, got error: %v", err)
	}

	// Test SetRegion with nil confined pointer
	err = cp.SetRegion(nil)
	if err == nil {
		t.Fatal("SetRegion should fail with nil confined pointer")
	}
}

// Test convenience functions
func TestConvenienceFunctions(t *testing.T) {
	manager := &PointerConstraintsManager{} // nil internal manager for testing

	// Test LockPointerAtCurrentPosition
	_, err := LockPointerAtCurrentPosition(manager, nil, nil)
	if err == nil {
		t.Fatal("LockPointerAtCurrentPosition should fail with nil internal manager")
	}

	// Test LockPointerPersistent
	_, err = LockPointerPersistent(manager, nil, nil)
	if err == nil {
		t.Fatal("LockPointerPersistent should fail with nil internal manager")
	}

	// Test ConfinePointerToRegion
	_, err = ConfinePointerToRegion(manager, nil, nil, nil)
	if err == nil {
		t.Fatal("ConfinePointerToRegion should fail with nil internal manager")
	}
}

// Test manager Close operations
func TestManagerClose(t *testing.T) {
	manager := &PointerConstraintsManager{}

	// Test close with nil components
	err := manager.Close()
	if err != nil {
		t.Errorf("Close should handle nil components gracefully, got error: %v", err)
	}

	// Test Destroy (which calls Close)
	err = manager.Destroy()
	if err != nil {
		t.Errorf("Destroy should handle nil components gracefully, got error: %v", err)
	}
}

// Benchmark basic operations
func BenchmarkLifetimeConstants(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = LifetimeOneshot
		_ = LifetimePersistent
	}
}

func BenchmarkErrorCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := &PointerConstraintsError{
			Code:    ERROR_ALREADY_CONSTRAINED,
			Message: "benchmark error",
		}
		_ = err.Error()
	}
}
