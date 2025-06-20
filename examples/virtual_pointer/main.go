// Package main demonstrates how to use the virtual_pointer package to simulate mouse movements.
//
// This example shows how to:
// - Create a virtual pointer manager
// - Create a virtual pointer
// - Perform various mouse operations (move, click, scroll)
// - Clean up resources properly
//
// Note: This is a demonstration of the API. In a real Wayland environment,
// you would need actual Wayland client library bindings.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bnema/libwldevices-go/virtual_pointer"
)

func main() {
	fmt.Println("Virtual Pointer Example - Mouse Movement and Control")
	fmt.Println("=====================================================")

	// Create a context for the application
	ctx := context.Background()

	// Create a virtual pointer manager
	fmt.Println("1. Creating virtual pointer manager...")
	manager, err := virtual_pointer.NewVirtualPointerManager(ctx)
	if err != nil {
		log.Fatalf("Failed to create virtual pointer manager: %v", err)
	}
	defer func() {
		fmt.Println("9. Closing virtual pointer manager...")
		if err := manager.Close(); err != nil {
			log.Printf("Error closing manager: %v", err)
		}
	}()

	// Create a virtual pointer
	fmt.Println("2. Creating virtual pointer...")
	pointer, err := manager.CreatePointer()
	if err != nil {
		log.Printf("Failed to create virtual pointer: %v", err)
		return
	}
	defer func() {
		fmt.Println("8. Closing virtual pointer...")
		if err := pointer.Close(); err != nil {
			log.Printf("Error closing pointer: %v", err)
		}
	}()

	// Demonstrate relative mouse movement
	fmt.Println("3. Performing relative mouse movements...")
	movements := []struct {
		dx, dy float64
		desc   string
	}{
		{100.0, 0.0, "Move right 100 pixels"},
		{0.0, 50.0, "Move down 50 pixels"},
		{-50.0, -25.0, "Move left 50 pixels and up 25 pixels"},
	}

	for _, move := range movements {
		fmt.Printf("   - %s\n", move.desc)
		if err := pointer.MoveRelative(move.dx, move.dy); err != nil {
			log.Printf("Error moving pointer: %v", err)
		}
		time.Sleep(500 * time.Millisecond) // Brief pause between movements
	}

	// Demonstrate absolute mouse movement
	fmt.Println("4. Performing absolute mouse movements...")
	absMovements := []struct {
		x, y, xExtent, yExtent uint32
		desc                   string
	}{
		{960, 540, 1920, 1080, "Move to center of 1920x1080 screen"},
		{100, 100, 1920, 1080, "Move to top-left area"},
		{1820, 980, 1920, 1080, "Move to bottom-right area"},
	}

	for _, move := range absMovements {
		fmt.Printf("   - %s\n", move.desc)
		// Note: MoveAbsolute is not a helper method, using MotionAbsolute directly
		if err := pointer.MotionAbsolute(time.Now(), move.x, move.y, move.xExtent, move.yExtent); err != nil {
			log.Printf("Error moving pointer absolutely: %v", err)
			continue
		}
		if err := pointer.Frame(); err != nil {
			log.Printf("Error moving pointer absolutely: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Demonstrate mouse clicks
	fmt.Println("5. Performing mouse clicks...")
	clicks := []struct {
		button uint32
		desc   string
	}{
		{virtual_pointer.BTN_LEFT, "Left click"},
		{virtual_pointer.BTN_RIGHT, "Right click"},
		{virtual_pointer.BTN_MIDDLE, "Middle click"},
	}

	for _, click := range clicks {
		fmt.Printf("   - %s\n", click.desc)
		// Generic click helper
		now := time.Now()
		if err := pointer.Button(now, click.button, virtual_pointer.ButtonStatePressed); err != nil {
			log.Printf("Error pressing button: %v", err)
			continue
		}
		if err := pointer.Button(now, click.button, virtual_pointer.ButtonStateReleased); err != nil {
			log.Printf("Error releasing button: %v", err)
			continue
		}
		if err := pointer.Frame(); err != nil {
			log.Printf("Error clicking button: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Demonstrate scrolling
	fmt.Println("6. Performing scroll operations...")
	scrolls := []struct {
		axis  uint32
		value float64
		desc  string
	}{
		{virtual_pointer.AXIS_VERTICAL_SCROLL, 10.0, "Scroll up"},
		{virtual_pointer.AXIS_VERTICAL_SCROLL, -10.0, "Scroll down"},
		{virtual_pointer.AXIS_HORIZONTAL_SCROLL, 5.0, "Scroll right"},
		{virtual_pointer.AXIS_HORIZONTAL_SCROLL, -5.0, "Scroll left"},
	}

	for _, scroll := range scrolls {
		fmt.Printf("   - %s\n", scroll.desc)
		if err := pointer.Axis(time.Now(), virtual_pointer.Axis(scroll.axis), scroll.value); err != nil {
			log.Printf("Error scrolling: %v", err)
			continue
		}
		if err := pointer.Frame(); err != nil {
			log.Printf("Error scrolling: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Demonstrate more complex operations
	fmt.Println("7. Performing complex mouse operations...")

	// Simulate drawing a square by moving and clicking
	fmt.Println("   - Drawing a virtual square with clicks")
	square := []struct {
		dx, dy float64
		click  bool
		desc   string
	}{
		{0, 0, true, "Click at starting position"},
		{100, 0, true, "Move right and click"},
		{0, 100, true, "Move down and click"},
		{-100, 0, true, "Move left and click"},
		{0, -100, true, "Move up and click (complete square)"},
	}

	for _, step := range square {
		if step.dx != 0 || step.dy != 0 {
			fmt.Printf("     Moving by (%.0f, %.0f)\n", step.dx, step.dy)
			if err := pointer.MoveRelative(step.dx, step.dy); err != nil {
				log.Printf("Error moving: %v", err)
			}
		}
		if step.click {
			fmt.Printf("     Clicking\n")
			now := time.Now()
			if err := pointer.Button(now, virtual_pointer.BTN_LEFT, virtual_pointer.ButtonStatePressed); err != nil {
				log.Printf("Error clicking: %v", err)
				continue
			}
			if err := pointer.Button(now, virtual_pointer.BTN_LEFT, virtual_pointer.ButtonStateReleased); err != nil {
				log.Printf("Error clicking: %v", err)
				continue
			}
			if err := pointer.Frame(); err != nil {
				log.Printf("Error clicking: %v", err)
			}
		}
		time.Sleep(300 * time.Millisecond)
	}

	// Demonstrate drag operation
	fmt.Println("   - Simulating drag operation (press, move, release)")
	if err := pointer.Button(time.Now(), virtual_pointer.BTN_LEFT, virtual_pointer.ButtonStatePressed); err != nil {
		log.Printf("Error pressing button for drag: %v", err)
	} else {
		// Move while button is held down
		for i := 0; i < 5; i++ {
			if err := pointer.Motion(time.Now(), 20.0, 10.0); err != nil {
				log.Printf("Error during drag motion: %v", err)
				break
			}
			if err := pointer.Frame(); err != nil {
				log.Printf("Error sending frame during drag: %v", err)
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		// Release the button
		if err := pointer.Button(time.Now(), virtual_pointer.BTN_LEFT, virtual_pointer.ButtonStateReleased); err != nil {
			log.Printf("Error releasing button after drag: %v", err)
		}
		if err := pointer.Frame(); err != nil {
			log.Printf("Error sending frame after drag: %v", err)
		}
	}

	fmt.Println("\nExample completed successfully!")
}

// demonstrateAdvancedFeatures shows more advanced virtual pointer features
func demonstrateAdvancedFeatures(pointer *virtual_pointer.VirtualPointer) {
	fmt.Println("Advanced Features:")

	// Set different axis sources
	fmt.Println("   - Setting different axis sources")
	sources := []struct {
		source uint32
		desc   string
	}{
		{virtual_pointer.AXIS_SOURCE_WHEEL, "Mouse wheel"},
		{virtual_pointer.AXIS_SOURCE_FINGER, "Touchpad finger"},
		{virtual_pointer.AXIS_SOURCE_CONTINUOUS, "Continuous scroll"},
		{virtual_pointer.AXIS_SOURCE_WHEEL_TILT, "Wheel tilt"},
	}

	for _, src := range sources {
		fmt.Printf("     Setting axis source: %s\n", src.desc)
		if err := pointer.AxisSource(virtual_pointer.AxisSource(src.source)); err != nil {
			log.Printf("Error setting axis source: %v", err)
		}

		// Send a small scroll with this source
		if err := pointer.Axis(time.Now(), virtual_pointer.AxisVertical, 1.0); err != nil {
			log.Printf("Error sending axis event: %v", err)
		}
		if err := pointer.Frame(); err != nil {
			log.Printf("Error sending frame: %v", err)
		}

		time.Sleep(200 * time.Millisecond)
	}

	// Demonstrate discrete scrolling
	fmt.Println("   - Discrete scrolling (scroll wheel clicks)")
	for i := 0; i < 3; i++ {
		if err := pointer.AxisDiscrete(time.Now(), virtual_pointer.AxisVertical, 10.0, 1); err != nil {
			log.Printf("Error with discrete scroll: %v", err)
		}
		if err := pointer.Frame(); err != nil {
			log.Printf("Error sending frame: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Demonstrate axis stop
	fmt.Println("   - Stopping scroll axis")
	if err := pointer.AxisStop(time.Now(), virtual_pointer.AxisVertical); err != nil {
		log.Printf("Error stopping axis: %v", err)
	}
	if err := pointer.Frame(); err != nil {
		log.Printf("Error sending frame: %v", err)
	}
}
