// Comprehensive integration test for virtual input functionality
//
// This test demonstrates both virtual pointer and virtual keyboard functionality
// in a single program. It performs a series of mouse movements, clicks, scrolls,
// and keyboard input to verify that all protocols are working correctly.
//
// Prerequisites:
// - Wayland compositor with virtual input support (Sway, Hyprland, etc.)
// - Active Wayland session
// - Focus on a text input field for keyboard tests
//
// Usage: go run tests/inject/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/bnema/wayland-virtual-input-go/virtual_pointer"
	"github.com/bnema/wayland-virtual-input-go/virtual_keyboard"
)

func main() {
	fmt.Println("=== Wayland Virtual Input Injection Test ===")
	fmt.Printf("WAYLAND_DISPLAY: %s\n", os.Getenv("WAYLAND_DISPLAY"))
	fmt.Printf("XDG_SESSION_TYPE: %s\n\n", os.Getenv("XDG_SESSION_TYPE"))

	ctx := context.Background()

	// Test Virtual Pointer
	fmt.Println("Testing Virtual Pointer...")
	if err := testVirtualPointer(ctx); err != nil {
		log.Printf("Virtual pointer test failed: %v", err)
	}

	fmt.Println("\n" + strings.Repeat("-", 50) + "\n")

	// Test Virtual Keyboard
	fmt.Println("Testing Virtual Keyboard...")
	if err := testVirtualKeyboard(ctx); err != nil {
		log.Printf("Virtual keyboard test failed: %v", err)
	}
}

func testVirtualPointer(ctx context.Context) error {
	// Create manager
	manager, err := virtual_pointer.NewVirtualPointerManager(ctx)
	if err != nil {
		return fmt.Errorf("failed to create pointer manager: %w", err)
	}
	defer manager.Close()
	fmt.Println("✓ Pointer manager created")

	// Create pointer
	pointer, err := manager.CreatePointer()
	if err != nil {
		return fmt.Errorf("failed to create virtual pointer: %w", err)
	}
	defer pointer.Close()
	fmt.Println("✓ Virtual pointer created")

	fmt.Println("\nStarting pointer tests in 2 seconds...")
	time.Sleep(2 * time.Second)

	// Test 1: Simple relative movement
	fmt.Println("\n1. Testing relative movement (100px right, 100px down)")
	if err := pointer.Motion(time.Now(), 100.0, 100.0); err != nil {
		fmt.Printf("   ✗ Motion failed: %v\n", err)
	} else {
		fmt.Println("   ✓ Motion sent")
	}
	if err := pointer.Frame(); err != nil {
		fmt.Printf("   ✗ Frame failed: %v\n", err)
	} else {
		fmt.Println("   ✓ Frame sent")
	}
	time.Sleep(500 * time.Millisecond)

	// Test 2: Series of small movements
	fmt.Println("\n2. Testing series of small movements")
	for i := 1; i <= 5; i++ {
		if err := pointer.Motion(time.Now(), 20.0, 20.0); err != nil {
			fmt.Printf("   ✗ Movement %d failed: %v\n", i, err)
		} else {
			fmt.Printf("   ✓ Movement %d sent\n", i)
		}
		pointer.Frame()
		time.Sleep(200 * time.Millisecond)
	}

	// Test 3: Mouse click
	fmt.Println("\n3. Testing left mouse button click")
	if err := pointer.Button(time.Now(), virtual_pointer.BTN_LEFT, virtual_pointer.ButtonStatePressed); err != nil {
		fmt.Printf("   ✗ Button press failed: %v\n", err)
	} else {
		fmt.Println("   ✓ Button pressed")
	}
	time.Sleep(100 * time.Millisecond)
	if err := pointer.Button(time.Now(), virtual_pointer.BTN_LEFT, virtual_pointer.ButtonStateReleased); err != nil {
		fmt.Printf("   ✗ Button release failed: %v\n", err)
	} else {
		fmt.Println("   ✓ Button released")
	}
	pointer.Frame()

	// Test 4: Scrolling
	fmt.Println("\n4. Testing mouse scroll")
	// Scroll down
	if err := pointer.Axis(time.Now(), virtual_pointer.AxisVertical, 5.0); err != nil {
		fmt.Printf("   ✗ Scroll down failed: %v\n", err)
	} else {
		fmt.Println("   ✓ Scroll down sent")
	}
	pointer.Frame()
	time.Sleep(500 * time.Millisecond)

	// Scroll up
	if err := pointer.Axis(time.Now(), virtual_pointer.AxisVertical, -5.0); err != nil {
		fmt.Printf("   ✗ Scroll up failed: %v\n", err)
	} else {
		fmt.Println("   ✓ Scroll up sent")
	}
	pointer.Frame()

	// Test 5: Use convenience functions
	fmt.Println("\n5. Testing convenience functions")
	if err := pointer.MoveRelative(50.0, 50.0); err != nil {
		fmt.Printf("   ✗ MoveRelative failed: %v\n", err)
	} else {
		fmt.Println("   ✓ MoveRelative succeeded")
	}
	time.Sleep(500 * time.Millisecond)

	if err := pointer.RightClick(); err != nil {
		fmt.Printf("   ✗ Right click failed: %v\n", err)
	} else {
		fmt.Println("   ✓ Right click succeeded")
	}

	return nil
}

func testVirtualKeyboard(ctx context.Context) error {
	// Create manager
	manager, err := virtual_keyboard.NewVirtualKeyboardManager(ctx)
	if err != nil {
		return fmt.Errorf("failed to create keyboard manager: %w", err)
	}
	defer manager.Close()
	fmt.Println("✓ Keyboard manager created")

	// Create keyboard
	keyboard, err := manager.CreateKeyboard()
	if err != nil {
		return fmt.Errorf("failed to create virtual keyboard: %w", err)
	}
	defer keyboard.Close()
	fmt.Println("✓ Virtual keyboard created")

	fmt.Println("\nStarting keyboard tests in 2 seconds...")
	fmt.Println("Click on a text field or terminal to see the input!")
	time.Sleep(2 * time.Second)

	// Test 1: Type "hello"
	fmt.Println("\n1. Typing 'hello'")
	keys := []struct {
		keycode uint32
		char    string
	}{
		{virtual_keyboard.KEY_H, "h"},
		{virtual_keyboard.KEY_E, "e"},
		{virtual_keyboard.KEY_L, "l"},
		{virtual_keyboard.KEY_L, "l"},
		{virtual_keyboard.KEY_O, "o"},
	}

	for _, k := range keys {
		if err := keyboard.Key(time.Now(), k.keycode, virtual_keyboard.KeyStatePressed); err != nil {
			fmt.Printf("   ✗ Failed to press '%s': %v\n", k.char, err)
		} else {
			fmt.Printf("   ✓ Pressed '%s'\n", k.char)
		}
		time.Sleep(50 * time.Millisecond)
		keyboard.Key(time.Now(), k.keycode, virtual_keyboard.KeyStateReleased)
		time.Sleep(50 * time.Millisecond)
	}

	// Test 2: Special keys
	fmt.Println("\n2. Testing special keys")
	fmt.Println("   Testing space key...")
	keyboard.Key(time.Now(), virtual_keyboard.KEY_SPACE, virtual_keyboard.KeyStatePressed)
	time.Sleep(50 * time.Millisecond)
	keyboard.Key(time.Now(), virtual_keyboard.KEY_SPACE, virtual_keyboard.KeyStateReleased)
	time.Sleep(200 * time.Millisecond)

	fmt.Println("   Testing enter key...")
	keyboard.Key(time.Now(), virtual_keyboard.KEY_ENTER, virtual_keyboard.KeyStatePressed)
	time.Sleep(50 * time.Millisecond)
	keyboard.Key(time.Now(), virtual_keyboard.KEY_ENTER, virtual_keyboard.KeyStateReleased)

	return nil
}