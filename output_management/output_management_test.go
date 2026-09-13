package output_management

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/bnema/libwldevices-go/internal/client"
	"github.com/bnema/libwldevices-go/internal/protocols"
	"github.com/bnema/libwldevices-go/internal/testcompositor"
	"github.com/bnema/wlturbo"
	"github.com/bnema/wlturbo/wl"
)

// Unit tests that don't require a compositor

// TestOutputMode tests the OutputMode struct
func TestOutputMode(t *testing.T) {
	mode := &OutputMode{
		Width:     1920,
		Height:    1080,
		Refresh:   60000,
		Preferred: true,
	}

	if mode.Width != 1920 {
		t.Errorf("Expected Width=1920, got %d", mode.Width)
	}
	if mode.Height != 1080 {
		t.Errorf("Expected Height=1080, got %d", mode.Height)
	}
	if mode.Refresh != 60000 {
		t.Errorf("Expected Refresh=60000, got %d", mode.Refresh)
	}
	if !mode.Preferred {
		t.Error("Expected Preferred=true")
	}

	// Test refresh rate conversion
	refreshHz := mode.GetRefreshRate()
	if math.Abs(refreshHz-60.0) > 0.001 {
		t.Errorf("Expected refresh rate ~60Hz, got %f", refreshHz)
	}
}

// TestOutputHead tests the OutputHead struct
func TestOutputHead(t *testing.T) {
	head := &OutputHead{
		ID:           1,
		Name:         "DP-1",
		Description:  "Dell Monitor",
		Make:         "Dell",
		Model:        "U2415",
		SerialNumber: "ABC123",
		PhysicalSize: Size{Width: 518, Height: 324},
		Position:     Position{X: 0, Y: 0},
		Transform:    TransformNormal,
		Scale:        1.0,
		Enabled:      true,
		CurrentMode:  &OutputMode{Width: 1920, Height: 1200, Refresh: 59997},
		modes: []*OutputMode{
			{Width: 1920, Height: 1200, Refresh: 59997, Preferred: true},
			{Width: 1920, Height: 1080, Refresh: 60000},
		},
	}

	// Test basic properties
	if head.Name != "DP-1" {
		t.Errorf("Expected Name='DP-1', got '%s'", head.Name)
	}
	if head.Make != "Dell" {
		t.Errorf("Expected Make='Dell', got '%s'", head.Make)
	}

	// Test physical size
	if head.PhysicalSize.Width != 518 {
		t.Errorf("Expected PhysicalSize.Width=518, got %d", head.PhysicalSize.Width)
	}

	// Test modes
	modes := head.GetModes()
	if len(modes) != 2 {
		t.Errorf("Expected 2 modes, got %d", len(modes))
	}

	// Test current mode
	if head.CurrentMode == nil {
		t.Error("Expected CurrentMode to be set")
	} else if head.CurrentMode.Width != 1920 {
		t.Errorf("Expected CurrentMode.Width=1920, got %d", head.CurrentMode.Width)
	}

	// Test bounds calculation
	x1, y1, x2, y2 := head.Bounds()
	if x1 != 0 || y1 != 0 {
		t.Errorf("Expected bounds origin (0,0), got (%d,%d)", x1, y1)
	}
	if x2 != 1920 || y2 != 1200 {
		t.Errorf("Expected bounds end (1920,1200), got (%d,%d)", x2, y2)
	}

	// Test contains point
	if !head.Contains(100, 100) {
		t.Error("Expected head to contain point (100,100)")
	}
	if head.Contains(-10, -10) {
		t.Error("Expected head to not contain point (-10,-10)")
	}
	if head.Contains(2000, 2000) {
		t.Error("Expected head to not contain point (2000,2000)")
	}
}

// TestPosition tests the Position struct
func TestPosition(t *testing.T) {
	tests := []struct {
		name string
		pos  Position
		x    int32
		y    int32
	}{
		{"origin", Position{0, 0}, 0, 0},
		{"positive", Position{100, 200}, 100, 200},
		{"negative", Position{-100, -200}, -100, -200},
		{"mixed", Position{100, -200}, 100, -200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pos.X != tt.x {
				t.Errorf("Expected X=%d, got %d", tt.x, tt.pos.X)
			}
			if tt.pos.Y != tt.y {
				t.Errorf("Expected Y=%d, got %d", tt.y, tt.pos.Y)
			}
		})
	}
}

// TestSize tests the Size struct
func TestSize(t *testing.T) {
	tests := []struct {
		name   string
		size   Size
		width  int32
		height int32
	}{
		{"zero", Size{0, 0}, 0, 0},
		{"standard", Size{1920, 1080}, 1920, 1080},
		{"square", Size{1000, 1000}, 1000, 1000},
		{"portrait", Size{1080, 1920}, 1080, 1920},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.size.Width != tt.width {
				t.Errorf("Expected Width=%d, got %d", tt.width, tt.size.Width)
			}
			if tt.size.Height != tt.height {
				t.Errorf("Expected Height=%d, got %d", tt.height, tt.size.Height)
			}
		})
	}
}

// TestTransform tests the Transform enum
func TestTransform(t *testing.T) {
	tests := []struct {
		transform Transform
		expected  string
	}{
		{TransformNormal, "normal"},
		{Transform90, "90"},
		{Transform180, "180"},
		{Transform270, "270"},
		{TransformFlipped, "flipped"},
		{TransformFlipped90, "flipped-90"},
		{TransformFlipped180, "flipped-180"},
		{TransformFlipped270, "flipped-270"},
		{Transform(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.transform.String()
			if got != tt.expected {
				t.Errorf("Transform.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestOutputManagerGetHeads tests retrieving output heads (without requiring compositor)
func TestOutputManagerGetHeads(t *testing.T) {
	manager := &OutputManager{
		heads: map[uint32]*OutputHead{
			1: {ID: 1, Name: "DP-1", Enabled: true},
			2: {ID: 2, Name: "DP-2", Enabled: false},
			3: {ID: 3, Name: "HDMI-1", Enabled: true},
		},
	}

	heads := manager.GetHeads()
	if len(heads) != 3 {
		t.Errorf("Expected 3 heads, got %d", len(heads))
	}

	// Verify all heads are present
	found := make(map[string]bool)
	for _, head := range heads {
		found[head.Name] = true
	}

	expectedNames := []string{"DP-1", "DP-2", "HDMI-1"}
	for _, name := range expectedNames {
		if !found[name] {
			t.Errorf("Expected head %s not found", name)
		}
	}
}

// TestOutputManagerGetEnabledHeads tests retrieving only enabled heads
func TestOutputManagerGetEnabledHeads(t *testing.T) {
	manager := &OutputManager{
		heads: map[uint32]*OutputHead{
			1: {ID: 1, Name: "DP-1", Enabled: true},
			2: {ID: 2, Name: "DP-2", Enabled: false},
			3: {ID: 3, Name: "HDMI-1", Enabled: true},
		},
	}

	heads := manager.GetEnabledHeads()
	if len(heads) != 2 {
		t.Errorf("Expected 2 enabled heads, got %d", len(heads))
	}

	// Verify only enabled heads are returned
	for _, head := range heads {
		if !head.Enabled {
			t.Errorf("Head %s is not enabled but was returned", head.Name)
		}
	}
}

// TestOutputManagerGetHeadByName tests finding a head by name
func TestOutputManagerGetHeadByName(t *testing.T) {
	manager := &OutputManager{
		heads: map[uint32]*OutputHead{
			1: {ID: 1, Name: "DP-1"},
			2: {ID: 2, Name: "DP-2"},
			3: {ID: 3, Name: "HDMI-1"},
		},
	}

	tests := []struct {
		name      string
		searchFor string
		expectNil bool
	}{
		{"existing DP-1", "DP-1", false},
		{"existing DP-2", "DP-2", false},
		{"existing HDMI-1", "HDMI-1", false},
		{"non-existing", "VGA-1", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			head := manager.GetHeadByName(tt.searchFor)
			if tt.expectNil && head != nil {
				t.Errorf("Expected nil for %s, got %v", tt.searchFor, head)
			}
			if !tt.expectNil && head == nil {
				t.Errorf("Expected head for %s, got nil", tt.searchFor)
			}
			if head != nil && head.Name != tt.searchFor {
				t.Errorf("Expected head name %s, got %s", tt.searchFor, head.Name)
			}
		})
	}
}

// TestOutputManagerGetHeadAtPoint tests spatial queries
func TestOutputManagerGetHeadAtPoint(t *testing.T) {
	manager := &OutputManager{
		heads: map[uint32]*OutputHead{
			1: {
				ID:       1,
				Name:     "DP-1",
				Position: Position{X: 0, Y: 0},
				CurrentMode: &OutputMode{
					Width:  1920,
					Height: 1080,
				},
				Enabled: true,
			},
			2: {
				ID:       2,
				Name:     "DP-2",
				Position: Position{X: 1920, Y: 0},
				CurrentMode: &OutputMode{
					Width:  1920,
					Height: 1080,
				},
				Enabled: true,
			},
			3: {
				ID:       3,
				Name:     "HDMI-1",
				Position: Position{X: 0, Y: 1080},
				CurrentMode: &OutputMode{
					Width:  1920,
					Height: 1080,
				},
				Enabled: false, // Disabled
			},
		},
	}

	tests := []struct {
		name      string
		x, y      int32
		expected  string
		expectNil bool
	}{
		{"top-left DP-1", 0, 0, "DP-1", false},
		{"center DP-1", 960, 540, "DP-1", false},
		{"edge DP-1", 1919, 1079, "DP-1", false},
		{"top-left DP-2", 1920, 0, "DP-2", false},
		{"center DP-2", 2880, 540, "DP-2", false},
		{"disabled monitor", 960, 1620, "", true}, // HDMI-1 is disabled
		{"outside any monitor", -100, -100, "", true},
		{"far outside", 5000, 5000, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			head := manager.GetHeadAtPoint(tt.x, tt.y)
			if tt.expectNil && head != nil {
				t.Errorf("Expected nil at (%d,%d), got %v", tt.x, tt.y, head.Name)
			}
			if !tt.expectNil && head == nil {
				t.Errorf("Expected head at (%d,%d), got nil", tt.x, tt.y)
			}
			if head != nil && head.Name != tt.expected {
				t.Errorf("Expected head %s at (%d,%d), got %s", tt.expected, tt.x, tt.y, head.Name)
			}
		})
	}
}

// TestOutputManagerThreadSafety tests concurrent access
func TestOutputManagerThreadSafety(t *testing.T) {
	manager := &OutputManager{
		heads: make(map[uint32]*OutputHead),
	}

	// Add initial heads
	for i := uint32(1); i <= 5; i++ {
		manager.heads[i] = &OutputHead{
			ID:      i,
			Name:    fmt.Sprintf("DP-%d", i),
			Enabled: i%2 == 0,
		}
	}

	// Run concurrent operations
	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Readers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = manager.GetHeads()
				_ = manager.GetEnabledHeads()
				_ = manager.GetHeadByName("DP-1")
				_ = manager.GetHeadAtPoint(100, 100)
			}
		}()
	}

	// Writers
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				// Simulate adding/updating heads
				manager.mu.Lock()
				headID := uint32(100 + id)
				manager.heads[headID] = &OutputHead{
					ID:   headID,
					Name: fmt.Sprintf("Dynamic-%d", id),
				}
				manager.mu.Unlock()

				// Simulate removing heads
				if j%10 == 0 {
					manager.mu.Lock()
					delete(manager.heads, headID)
					manager.mu.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()
}

// TestOutputHandlers tests event handler registration
func TestOutputHandlers(t *testing.T) {
	manager := &OutputManager{
		heads: make(map[uint32]*OutputHead),
	}

	headAddedCalled := false
	headRemovedCalled := false
	configurationChangedCalled := false

	handlers := OutputHandlers{
		OnHeadAdded: func(head *OutputHead) {
			headAddedCalled = true
		},
		OnHeadRemoved: func(head *OutputHead) {
			headRemovedCalled = true
		},
		OnConfigurationChanged: func(heads []*OutputHead) {
			configurationChangedCalled = true
		},
	}

	manager.SetHandlers(handlers)

	// Simulate events
	if manager.handlers.OnHeadAdded != nil {
		manager.handlers.OnHeadAdded(&OutputHead{})
	}
	if manager.handlers.OnHeadRemoved != nil {
		manager.handlers.OnHeadRemoved(&OutputHead{})
	}
	if manager.handlers.OnConfigurationChanged != nil {
		manager.handlers.OnConfigurationChanged([]*OutputHead{})
	}

	if !headAddedCalled {
		t.Error("OnHeadAdded handler not called")
	}
	if !headRemovedCalled {
		t.Error("OnHeadRemoved handler not called")
	}
	if !configurationChangedCalled {
		t.Error("OnConfigurationChanged handler not called")
	}
}

// TestOutputModeComparison tests mode comparison
func TestOutputModeComparison(t *testing.T) {
	mode1 := &OutputMode{Width: 1920, Height: 1080, Refresh: 60000}
	mode2 := &OutputMode{Width: 1920, Height: 1080, Refresh: 60000}
	mode3 := &OutputMode{Width: 2560, Height: 1440, Refresh: 144000}

	// Same values should be considered equal
	if mode1.Width != mode2.Width || mode1.Height != mode2.Height || mode1.Refresh != mode2.Refresh {
		t.Error("Expected mode1 and mode2 to have equal values")
	}

	// Different values
	if mode1.Width == mode3.Width || mode1.Height == mode3.Height || mode1.Refresh == mode3.Refresh {
		t.Error("Expected mode1 and mode3 to have different values")
	}
}

// TestNilSafety tests nil pointer safety
func TestNilSafety(t *testing.T) {
	var manager *OutputManager

	// These should not panic
	heads := manager.GetHeads()
	if heads != nil {
		t.Error("Expected nil heads from nil manager")
	}

	enabled := manager.GetEnabledHeads()
	if enabled != nil {
		t.Error("Expected nil enabled heads from nil manager")
	}

	head := manager.GetHeadByName("test")
	if head != nil {
		t.Error("Expected nil head from nil manager")
	}

	point := manager.GetHeadAtPoint(0, 0)
	if point != nil {
		t.Error("Expected nil head at point from nil manager")
	}

	err := manager.Close()
	if err != nil {
		t.Error("Expected nil error from closing nil manager")
	}
}

// TestOutputManagerNilClose tests closing with nil components
func TestOutputManagerNilClose(t *testing.T) {
	manager := &OutputManager{}
	err := manager.Close()
	if err != nil {
		t.Errorf("Close() on nil components returned error: %v", err)
	}
}

// TestHeadBounds tests the bounds calculation
func TestHeadBounds(t *testing.T) {
	tests := []struct {
		name   string
		head   *OutputHead
		x1, y1 int32
		x2, y2 int32
	}{
		{
			name: "origin head",
			head: &OutputHead{
				Position:    Position{X: 0, Y: 0},
				CurrentMode: &OutputMode{Width: 1920, Height: 1080},
			},
			x1: 0, y1: 0, x2: 1920, y2: 1080,
		},
		{
			name: "offset head",
			head: &OutputHead{
				Position:    Position{X: 100, Y: 200},
				CurrentMode: &OutputMode{Width: 1280, Height: 720},
			},
			x1: 100, y1: 200, x2: 1380, y2: 920,
		},
		{
			name: "negative position",
			head: &OutputHead{
				Position:    Position{X: -100, Y: -50},
				CurrentMode: &OutputMode{Width: 800, Height: 600},
			},
			x1: -100, y1: -50, x2: 700, y2: 550,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x1, y1, x2, y2 := tt.head.Bounds()
			if x1 != tt.x1 || y1 != tt.y1 || x2 != tt.x2 || y2 != tt.y2 {
				t.Errorf("Bounds() = (%d,%d,%d,%d), want (%d,%d,%d,%d)",
					x1, y1, x2, y2, tt.x1, tt.y1, tt.x2, tt.y2)
			}
		})
	}
}

// Benchmark tests
func BenchmarkGetHeadAtPoint(b *testing.B) {
	manager := &OutputManager{
		heads: make(map[uint32]*OutputHead),
	}

	// Create a grid of monitors
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			id := uint32(i*3 + j + 1)
			manager.heads[id] = &OutputHead{
				ID:       id,
				Name:     fmt.Sprintf("Monitor-%d-%d", i, j),
				Position: Position{X: int32(i) * 1920, Y: int32(j) * 1080},
				CurrentMode: &OutputMode{
					Width:  1920,
					Height: 1080,
				},
				Enabled: true,
			}
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Test various points
		x := int32(i % 5760) // 3 * 1920
		y := int32(i % 3240) // 3 * 1080
		_ = manager.GetHeadAtPoint(x, y)
	}
}

func BenchmarkGetEnabledHeads(b *testing.B) {
	manager := &OutputManager{
		heads: make(map[uint32]*OutputHead),
	}

	// Create many heads, half enabled
	for i := uint32(1); i <= 100; i++ {
		manager.heads[i] = &OutputHead{
			ID:      i,
			Name:    fmt.Sprintf("Head-%d", i),
			Enabled: i%2 == 0,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.GetEnabledHeads()
	}
}

// TestMemoryAllocation tests for memory leaks
func TestMemoryAllocation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory allocation test in short mode")
	}

	manager := &OutputManager{
		heads: make(map[uint32]*OutputHead),
	}

	// Get initial memory stats
	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Perform many allocations and deallocations
	for i := 0; i < 1000; i++ {
		// Add heads
		for j := uint32(0); j < 100; j++ {
			manager.mu.Lock()
			manager.heads[j] = &OutputHead{
				ID:   j,
				Name: fmt.Sprintf("Head-%d-%d", i, j),
				modes: []*OutputMode{
					{Width: 1920, Height: 1080},
					{Width: 2560, Height: 1440},
				},
			}
			manager.mu.Unlock()
		}

		// Remove heads
		manager.mu.Lock()
		for k := range manager.heads {
			delete(manager.heads, k)
		}
		manager.mu.Unlock()
	}

	// Get final memory stats
	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// Check that we haven't leaked too much memory
	// Allow for some allocation due to runtime overhead
	// Safe conversion: runtime memory stats fit in int64
	leaked := int64(m2.Alloc) - int64(m1.Alloc)
	maxAllowed := int64(10 * 1024 * 1024) // 10MB tolerance

	if leaked > maxAllowed {
		t.Errorf("Possible memory leak detected: %d bytes leaked (max allowed: %d)", leaked, maxAllowed)
	}
}

// ---------------------------------------------------------------------------
// Wire-level tests driving the in-process test compositor. These replace any
// dependency on a live Wayland session.
// ---------------------------------------------------------------------------

const (
	outputGlobalName = 1

	// Object IDs chosen by the test compositor for the objects it creates.
	// They are deliberately far above the client's own allocation range.
	testHeadObjectID = 0x7F000001
	testModeObjectID = 0x7F000002

	testSerial = 42
)

// newOutputCompositor starts a compositor announcing zwlr_output_manager_v1 and
// points the client library at it.
func newOutputCompositor(t *testing.T) *testcompositor.Server {
	t.Helper()

	srv := testcompositor.Start(t, testcompositor.Global{
		Name:      outputGlobalName,
		Interface: protocols.OutputManagerInterface,
		Version:   4,
	})
	srv.Env(t)
	return srv
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

// pushHeadAndDone waits for the client to bind the output manager global and
// then pushes one complete head + mode + done sequence describing a single
// 1920x1200@59.997Hz virtual output.
func pushHeadAndDone(srv *testcompositor.Server) error {
	deadline := time.Now().Add(2 * time.Second)
	var managerID uint32
	for managerID == 0 {
		managerID = srv.ObjectID(protocols.OutputManagerInterface)
		if managerID == 0 {
			if time.Now().After(deadline) {
				return errors.New("zwlr_output_manager_v1 was never bound")
			}
			time.Sleep(time.Millisecond)
		}
	}

	events := []struct {
		object uint32
		opcode uint16
		args   []any
	}{
		{managerID, 0, []any{uint32(testHeadObjectID)}},        // head
		{testHeadObjectID, 0, []any{"DP-1"}},                   // name
		{testHeadObjectID, 1, []any{"Dell U2415 (DP-1)"}},      // description
		{testHeadObjectID, 2, []any{int32(518), int32(324)}},   // physical_size
		{testHeadObjectID, 3, []any{uint32(testModeObjectID)}}, // mode
		{testModeObjectID, 0, []any{int32(1920), int32(1200)}}, // mode size
		{testModeObjectID, 1, []any{int32(59997)}},             // mode refresh
		{testModeObjectID, 2, []any{}},                         // mode preferred
		{testHeadObjectID, 4, []any{int32(1)}},                 // enabled
		{testHeadObjectID, 6, []any{int32(100), int32(50)}},    // position
		{testHeadObjectID, 7, []any{int32(TransformNormal)}},   // transform
		{testHeadObjectID, 8, []any{uint32(256)}},              // scale 1.0
		{testHeadObjectID, 10, []any{"Dell"}},                  // make
		{testHeadObjectID, 11, []any{"U2415"}},                 // model
		{testHeadObjectID, 12, []any{"ABC123"}},                // serial_number
		{managerID, 1, []any{uint32(testSerial)}},              // done
	}

	for _, event := range events {
		if err := srv.SendEvent(event.object, event.opcode, event.args...); err != nil {
			return err
		}
	}
	return nil
}

// newConfiguredOutputManager creates an output manager whose initial
// configuration is delivered by the test compositor.
func newConfiguredOutputManager(t *testing.T, srv *testcompositor.Server) *OutputManager {
	t.Helper()

	sent := make(chan error, 1)
	go func() { sent <- pushHeadAndDone(srv) }()

	manager, err := NewOutputManager(context.Background())
	if err != nil {
		t.Fatalf("NewOutputManager: %v", err)
	}
	t.Cleanup(func() { _ = manager.Close() })

	if err := <-sent; err != nil {
		t.Fatalf("push initial configuration: %v", err)
	}
	return manager
}

func TestManagerBindsOutputManagerGlobal(t *testing.T) {
	srv := newOutputCompositor(t)
	manager := newConfiguredOutputManager(t, srv)

	bind := waitForBind(t, srv, protocols.OutputManagerInterface)
	if got := bind.Uint32(0); got != outputGlobalName {
		t.Errorf("bind name = %d, want %d", got, outputGlobalName)
	}
	iface, consumed := bind.String(4)
	if iface != protocols.OutputManagerInterface {
		t.Errorf("bind interface = %q, want %q", iface, protocols.OutputManagerInterface)
	}
	if got := bind.Uint32(4 + consumed); got != 4 {
		t.Errorf("bind version = %d, want 4", got)
	}
	bindID := bind.Uint32(8 + consumed)
	if bindID == 0 {
		t.Fatal("bind used object ID 0")
	}
	if got := srv.ObjectID(protocols.OutputManagerInterface); got != bindID {
		t.Errorf("bound object ID = %d, want %d", got, bindID)
	}

	// Closing stops event delivery: zwlr_output_manager_v1.stop is opcode 1.
	if err := manager.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	stop := waitForRequest(t, srv, bindID, 1)
	if len(stop.Body) != 0 {
		t.Errorf("stop carried %d bytes of arguments, want none", len(stop.Body))
	}
}

func TestManagerFailsWithoutOutputManagerGlobal(t *testing.T) {
	srv := testcompositor.Start(t)
	srv.Env(t)

	manager, err := NewOutputManager(context.Background())
	if err == nil {
		_ = manager.Close()
		t.Fatal("NewOutputManager succeeded without zwlr_output_manager_v1")
	}
	if got := err.Error(); got != "zwlr_output_manager_v1 not available - compositor may not support wlr-output-management protocol" {
		t.Fatalf("error = %q, want the missing-global message", got)
	}
}

func TestManagerFailsOnCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	manager, err := NewOutputManager(ctx)
	if err == nil {
		_ = manager.Close()
		t.Fatal("NewOutputManager succeeded with a cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestInitialConfigurationIsParsedFromEvents(t *testing.T) {
	srv := newOutputCompositor(t)
	manager := newConfiguredOutputManager(t, srv)
	managerID := srv.ObjectID(protocols.OutputManagerInterface)

	heads := manager.GetHeads()
	if len(heads) != 1 {
		t.Fatalf("GetHeads() = %d heads, want 1", len(heads))
	}
	head := heads[0]

	if head.ID != testHeadObjectID {
		t.Errorf("head.ID = %d, want %d", head.ID, testHeadObjectID)
	}
	if head.Name != "DP-1" {
		t.Errorf("head.Name = %q, want %q", head.Name, "DP-1")
	}
	if head.Description != "Dell U2415 (DP-1)" {
		t.Errorf("head.Description = %q, want the description event value", head.Description)
	}
	if head.PhysicalSize != (Size{Width: 518, Height: 324}) {
		t.Errorf("head.PhysicalSize = %+v, want 518x324", head.PhysicalSize)
	}
	if head.Position != (Position{X: 100, Y: 50}) {
		t.Errorf("head.Position = %+v, want (100,50)", head.Position)
	}
	if !head.Enabled {
		t.Error("head.Enabled = false, want true")
	}
	if head.Transform != TransformNormal {
		t.Errorf("head.Transform = %v, want %v", head.Transform, TransformNormal)
	}
	if head.Scale != 1.0 {
		t.Errorf("head.Scale = %v, want 1.0", head.Scale)
	}
	if head.Make != "Dell" || head.Model != "U2415" || head.SerialNumber != "ABC123" {
		t.Errorf("head make/model/serial = %q/%q/%q, want Dell/U2415/ABC123", head.Make, head.Model, head.SerialNumber)
	}

	modes := head.GetModes()
	if len(modes) != 1 {
		t.Fatalf("head.GetModes() = %d modes, want 1", len(modes))
	}
	mode := modes[0]
	if mode.Width != 1920 || mode.Height != 1200 || mode.Refresh != 59997 || !mode.Preferred {
		t.Errorf("mode = %dx%d@%d preferred=%v, want 1920x1200@59997 preferred", mode.Width, mode.Height, mode.Refresh, mode.Preferred)
	}
	if head.Mode != mode {
		t.Errorf("head.Mode = %p, want the preferred mode %p", head.Mode, mode)
	}

	if !manager.hasSerial || manager.serial != testSerial {
		t.Errorf("manager serial = %d (set=%v), want %d", manager.serial, manager.hasSerial, testSerial)
	}

	// A later done event must notify OnConfigurationChanged with the known heads.
	changed := make(chan []*OutputHead, 1)
	manager.SetHandlers(OutputHandlers{
		OnConfigurationChanged: func(heads []*OutputHead) { changed <- heads },
	})
	if err := srv.SendEvent(managerID, 1, uint32(testSerial+1)); err != nil {
		t.Fatalf("SendEvent(done): %v", err)
	}
	select {
	case heads := <-changed:
		if len(heads) != 1 || heads[0].Name != "DP-1" {
			t.Fatalf("OnConfigurationChanged heads = %+v, want the DP-1 head", heads)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("OnConfigurationChanged was not called for the done event")
	}
}

func TestHeadFinishedRemovesHead(t *testing.T) {
	srv := newOutputCompositor(t)
	manager := newConfiguredOutputManager(t, srv)

	removed := make(chan *OutputHead, 1)
	manager.SetHandlers(OutputHandlers{
		OnHeadRemoved: func(head *OutputHead) { removed <- head },
	})

	if err := srv.SendEvent(testHeadObjectID, 9); err != nil {
		t.Fatalf("SendEvent(finished): %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for len(manager.GetHeads()) != 0 {
		if time.Now().After(deadline) {
			t.Fatalf("head still present after the finished event: %+v", manager.GetHeads())
		}
		time.Sleep(time.Millisecond)
	}

	select {
	case head := <-removed:
		if head.Name != "DP-1" {
			t.Errorf("OnHeadRemoved head.Name = %q, want %q", head.Name, "DP-1")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("OnHeadRemoved was not called for the finished event")
	}
}

func TestConfigurationRequestsUseProtocolOpcodes(t *testing.T) {
	srv := newOutputCompositor(t)
	manager := newConfiguredOutputManager(t, srv)
	managerID := srv.ObjectID(protocols.OutputManagerInterface)
	head := manager.GetHeads()[0]

	config, err := manager.manager.CreateConfiguration(manager.serial)
	if err != nil {
		t.Fatalf("CreateConfiguration: %v", err)
	}
	create := waitForRequest(t, srv, managerID, 0)
	configID := create.Uint32(0)
	if configID == 0 {
		t.Fatal("create_configuration new_id = 0, want a fresh object ID")
	}
	if got := create.Uint32(4); got != testSerial {
		t.Errorf("create_configuration serial = %d, want %d", got, testSerial)
	}

	configHead, err := config.EnableHead(head.head)
	if err != nil {
		t.Fatalf("EnableHead: %v", err)
	}
	enable := waitForRequest(t, srv, configID, 0)
	configHeadID := enable.Uint32(0)
	if configHeadID == 0 {
		t.Fatal("enable_head new_id = 0, want a fresh object ID")
	}
	if got := enable.Uint32(4); got != testHeadObjectID {
		t.Errorf("enable_head head = %d, want %d", got, testHeadObjectID)
	}

	if err := configHead.SetMode(head.modes[0].mode); err != nil {
		t.Fatalf("SetMode: %v", err)
	}
	setMode := waitForRequest(t, srv, configHeadID, 0)
	if got := setMode.Uint32(0); got != testModeObjectID {
		t.Errorf("set_mode mode = %d, want %d", got, testModeObjectID)
	}

	if err := configHead.SetCustomMode(1280, 720, 60000); err != nil {
		t.Fatalf("SetCustomMode: %v", err)
	}
	custom := waitForRequest(t, srv, configHeadID, 1)
	if custom.Uint32(0) != 1280 || custom.Uint32(4) != 720 || custom.Uint32(8) != 60000 {
		t.Errorf("set_custom_mode = %d,%d,%d, want 1280,720,60000", custom.Uint32(0), custom.Uint32(4), custom.Uint32(8))
	}

	if err := configHead.SetPosition(1920, 0); err != nil {
		t.Fatalf("SetPosition: %v", err)
	}
	position := waitForRequest(t, srv, configHeadID, 2)
	if position.Uint32(0) != 1920 || position.Uint32(4) != 0 {
		t.Errorf("set_position = %d,%d, want 1920,0", position.Uint32(0), position.Uint32(4))
	}

	if err := configHead.SetTransform(int32(Transform90)); err != nil {
		t.Fatalf("SetTransform: %v", err)
	}
	transform := waitForRequest(t, srv, configHeadID, 3)
	if got := transform.Uint32(0); got != uint32(Transform90) {
		t.Errorf("set_transform = %d, want %d", got, Transform90)
	}

	if err := configHead.SetScale(wl.Fixed(256)); err != nil {
		t.Fatalf("SetScale: %v", err)
	}
	scale := waitForRequest(t, srv, configHeadID, 4)
	if got := scale.Uint32(0); got != 256 {
		t.Errorf("set_scale = %d, want 256 (1.0 in 24.8 fixed point)", got)
	}

	if err := configHead.SetAdaptiveSync(1); err != nil {
		t.Fatalf("SetAdaptiveSync: %v", err)
	}
	adaptive := waitForRequest(t, srv, configHeadID, 5)
	if got := adaptive.Uint32(0); got != 1 {
		t.Errorf("set_adaptive_sync = %d, want 1", got)
	}

	if err := config.DisableHead(head.head); err != nil {
		t.Fatalf("DisableHead: %v", err)
	}
	disable := waitForRequest(t, srv, configID, 1)
	if got := disable.Uint32(0); got != testHeadObjectID {
		t.Errorf("disable_head head = %d, want %d", got, testHeadObjectID)
	}

	if err := config.Test(); err != nil {
		t.Fatalf("Test: %v", err)
	}
	if test := waitForRequest(t, srv, configID, 3); len(test.Body) != 0 {
		t.Errorf("test carried %d bytes of arguments, want none", len(test.Body))
	}

	if err := config.Apply(); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if apply := waitForRequest(t, srv, configID, 2); len(apply.Body) != 0 {
		t.Errorf("apply carried %d bytes of arguments, want none", len(apply.Body))
	}

	if err := config.Destroy(); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
	if destroy := waitForRequest(t, srv, configID, 4); len(destroy.Body) != 0 {
		t.Errorf("destroy carried %d bytes of arguments, want none", len(destroy.Body))
	}
}

// A compositor error must surface through the connection instead of being
// swallowed. The OutputManager's own dispatcher intentionally stops on the
// first connection error, so this test drives the manager's display directly
// to prove the error reaches the caller of Roundtrip.
func TestDisplayErrorSurfacesThroughManagerConnection(t *testing.T) {
	srv := newOutputCompositor(t)

	c, err := client.NewClient()
	if err != nil {
		t.Fatalf("client.NewClient: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	manager := protocols.NewOutputManager(c.GetContext())
	if err := c.GetRegistry().Bind(c.GetOutputManagerName(), protocols.OutputManagerInterface, 4, manager); err != nil {
		t.Fatalf("bind output manager: %v", err)
	}

	managerID := manager.ID()
	if managerID == 0 {
		t.Fatal("output manager was not bound")
	}
	if err := srv.SendDisplayError(managerID, 1, "denied"); err != nil {
		t.Fatalf("SendDisplayError: %v", err)
	}

	err = c.GetDisplay().Roundtrip()
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
