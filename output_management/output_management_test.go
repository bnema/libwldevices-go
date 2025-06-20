package output_management

import (
	"fmt"
	"math"
	"runtime"
	"sync"
	"testing"
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
