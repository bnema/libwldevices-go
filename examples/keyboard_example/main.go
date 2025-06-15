// Package main demonstrates how to use the virtual_keyboard package to simulate keyboard input.
//
// This example shows how to:
// - Create a virtual keyboard manager
// - Create a virtual keyboard
// - Type text and perform keyboard operations
// - Handle modifiers and key combinations
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

	"github.com/bnema/wayland-virtual-input-go/virtual_keyboard"
)

func main() {
	fmt.Println("Virtual Keyboard Example - Keyboard Input Simulation")
	fmt.Println("====================================================")

	// Create a context for the application
	ctx := context.Background()

	// Create a virtual keyboard manager
	fmt.Println("1. Creating virtual keyboard manager...")
	manager, err := virtual_keyboard.NewVirtualKeyboardManager(ctx)
	if err != nil {
		log.Fatalf("Failed to create virtual keyboard manager: %v", err)
	}
	defer func() {
		fmt.Println("9. Destroying virtual keyboard manager...")
		if err := manager.Destroy(); err != nil {
			log.Printf("Error destroying manager: %v", err)
		}
	}()

	// Create a virtual keyboard
	fmt.Println("2. Creating virtual keyboard...")
	keyboard, err := manager.CreateVirtualKeyboard(nil)
	if err != nil {
		log.Fatalf("Failed to create virtual keyboard: %v", err)
	}
	defer func() {
		fmt.Println("8. Destroying virtual keyboard...")
		if err := keyboard.Destroy(); err != nil {
			log.Printf("Error destroying keyboard: %v", err)
		}
	}()

	// Set up keymap (in a real implementation, you'd provide an actual keymap file)
	fmt.Println("3. Setting up keymap...")
	if err := keyboard.Keymap(virtual_keyboard.KEYMAP_FORMAT_NO_KEYMAP, nil, 0); err != nil {
		log.Printf("Warning: Failed to set keymap: %v", err)
	}

	// Demonstrate basic key typing
	fmt.Println("4. Typing individual keys...")
	keys := []struct {
		key  uint32
		desc string
	}{
		{virtual_keyboard.KEY_H, "H"},
		{virtual_keyboard.KEY_E, "e"},
		{virtual_keyboard.KEY_L, "l"},
		{virtual_keyboard.KEY_L, "l"},
		{virtual_keyboard.KEY_O, "o"},
		{virtual_keyboard.KEY_SPACE, "Space"},
		{virtual_keyboard.KEY_W, "W"},
		{virtual_keyboard.KEY_O, "o"},
		{virtual_keyboard.KEY_R, "r"},
		{virtual_keyboard.KEY_L, "l"},
		{virtual_keyboard.KEY_D, "d"},
	}

	for _, key := range keys {
		fmt.Printf("   - Typing: %s\n", key.desc)
		if err := virtual_keyboard.TypeKey(keyboard, key.key); err != nil {
			log.Printf("Error typing key %s: %v", key.desc, err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("   - Pressing Enter")
	if err := virtual_keyboard.TypeKey(keyboard, virtual_keyboard.KEY_ENTER); err != nil {
		log.Printf("Error typing Enter: %v", err)
	}

	// Demonstrate string typing
	fmt.Println("5. Typing strings...")
	strings := []string{
		"Hello, Wayland!",
		"This is a test of virtual keyboard input.",
		"Special characters: !@#$%^&*()",
		"Numbers: 1234567890",
		"Mixed case: AbCdEfGhIjKlMnOpQrStUvWxYz",
	}

	for _, str := range strings {
		fmt.Printf("   - Typing string: \"%s\"\n", str)
		if err := virtual_keyboard.TypeString(keyboard, str); err != nil {
			log.Printf("Error typing string: %v", err)
		}
		// Press Enter after each string
		if err := virtual_keyboard.TypeKey(keyboard, virtual_keyboard.KEY_ENTER); err != nil {
			log.Printf("Error typing Enter: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Demonstrate modifier keys
	fmt.Println("6. Testing modifier keys...")
	modifierTests := []struct {
		modifier uint32
		key      uint32
		desc     string
	}{
		{virtual_keyboard.MOD_CTRL, virtual_keyboard.KEY_C, "Ctrl+C (Copy)"},
		{virtual_keyboard.MOD_CTRL, virtual_keyboard.KEY_V, "Ctrl+V (Paste)"},
		{virtual_keyboard.MOD_CTRL, virtual_keyboard.KEY_Z, "Ctrl+Z (Undo)"},
		{virtual_keyboard.MOD_CTRL, virtual_keyboard.KEY_S, "Ctrl+S (Save)"},
		{virtual_keyboard.MOD_ALT, virtual_keyboard.KEY_TAB, "Alt+Tab (Switch)"},
		{virtual_keyboard.MOD_CTRL | virtual_keyboard.MOD_SHIFT, virtual_keyboard.KEY_Z, "Ctrl+Shift+Z (Redo)"},
	}

	for _, test := range modifierTests {
		fmt.Printf("   - Key combination: %s\n", test.desc)
		if err := virtual_keyboard.KeyCombo(keyboard, test.modifier, test.key); err != nil {
			log.Printf("Error with key combination: %v", err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	// Demonstrate function keys
	fmt.Println("7. Testing function keys...")
	functionKeys := []struct {
		key  uint32
		desc string
	}{
		{virtual_keyboard.KEY_F1, "F1"},
		{virtual_keyboard.KEY_F2, "F2"},
		{virtual_keyboard.KEY_F5, "F5 (Refresh)"},
		{virtual_keyboard.KEY_F11, "F11 (Fullscreen)"},
		{virtual_keyboard.KEY_F12, "F12"},
	}

	for _, fkey := range functionKeys {
		fmt.Printf("   - Function key: %s\n", fkey.desc)
		if err := virtual_keyboard.TypeKey(keyboard, fkey.key); err != nil {
			log.Printf("Error typing function key: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Demonstrate arrow keys and navigation
	fmt.Println("   - Arrow keys and navigation...")
	navKeys := []struct {
		key  uint32
		desc string
	}{
		{virtual_keyboard.KEY_UP, "Up Arrow"},
		{virtual_keyboard.KEY_DOWN, "Down Arrow"},
		{virtual_keyboard.KEY_LEFT, "Left Arrow"},
		{virtual_keyboard.KEY_RIGHT, "Right Arrow"},
		{virtual_keyboard.KEY_HOME, "Home"},
		{virtual_keyboard.KEY_END, "End"},
		{virtual_keyboard.KEY_PAGEUP, "Page Up"},
		{virtual_keyboard.KEY_PAGEDOWN, "Page Down"},
	}

	for _, navKey := range navKeys {
		fmt.Printf("   - Navigation key: %s\n", navKey.desc)
		if err := virtual_keyboard.TypeKey(keyboard, navKey.key); err != nil {
			log.Printf("Error typing navigation key: %v", err)
		}
		time.Sleep(150 * time.Millisecond)
	}

	// Demonstrate more complex operations
	fmt.Println("7. Performing complex keyboard operations...")

	// Simulate typing a paragraph with proper formatting
	fmt.Println("   - Typing formatted text with tabs and newlines")
	formattedText := []struct {
		action func() error
		desc   string
	}{
		{func() error { return virtual_keyboard.TypeString(keyboard, "Dear User,") }, "Type greeting"},
		{func() error { return virtual_keyboard.TypeKey(keyboard, virtual_keyboard.KEY_ENTER) }, "New line"},
		{func() error { return virtual_keyboard.TypeKey(keyboard, virtual_keyboard.KEY_ENTER) }, "Blank line"},
		{func() error { return virtual_keyboard.TypeKey(keyboard, virtual_keyboard.KEY_TAB) }, "Tab indent"},
		{func() error { return virtual_keyboard.TypeString(keyboard, "This is a demonstration of virtual keyboard input.") }, "Type paragraph"},
		{func() error { return virtual_keyboard.TypeKey(keyboard, virtual_keyboard.KEY_ENTER) }, "New line"},
		{func() error { return virtual_keyboard.TypeKey(keyboard, virtual_keyboard.KEY_TAB) }, "Tab indent"},
		{func() error { return virtual_keyboard.TypeString(keyboard, "The virtual keyboard can simulate complex typing patterns.") }, "Type second paragraph"},
		{func() error { return virtual_keyboard.TypeKey(keyboard, virtual_keyboard.KEY_ENTER) }, "New line"},
		{func() error { return virtual_keyboard.TypeKey(keyboard, virtual_keyboard.KEY_ENTER) }, "Blank line"},
		{func() error { return virtual_keyboard.TypeString(keyboard, "Best regards,") }, "Type closing"},
		{func() error { return virtual_keyboard.TypeKey(keyboard, virtual_keyboard.KEY_ENTER) }, "New line"},
		{func() error { return virtual_keyboard.TypeString(keyboard, "Virtual Keyboard Example") }, "Type signature"},
	}

	for _, action := range formattedText {
		fmt.Printf("     - %s\n", action.desc)
		if err := action.action(); err != nil {
			log.Printf("Error in formatted text action: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Demonstrate rapid typing
	fmt.Println("   - Rapid typing test")
	rapidText := "The quick brown fox jumps over the lazy dog. "
	for i := 0; i < 3; i++ {
		if err := virtual_keyboard.TypeString(keyboard, rapidText); err != nil {
			log.Printf("Error in rapid typing: %v", err)
		}
	}

	// Demonstrate modifier state management
	fmt.Println("   - Testing modifier state management")
	if err := demonstrateModifierStates(keyboard); err != nil {
		log.Printf("Error in modifier state demo: %v", err)
	}

	fmt.Println("\nExample completed successfully!")
	fmt.Println("Note: In a real Wayland environment, these operations would")
	fmt.Println("actually send keyboard input to the focused application.")
}

// demonstrateModifierStates shows how to manage modifier key states
func demonstrateModifierStates(keyboard virtual_keyboard.VirtualKeyboard) error {
	fmt.Println("     - Pressing and holding Shift")
	if err := keyboard.KeyPress(virtual_keyboard.KEY_LEFTSHIFT); err != nil {
		return fmt.Errorf("failed to press shift: %v", err)
	}

	// Type some letters while shift is held
	shiftedText := "UPPERCASE TEXT"
	for _, char := range shiftedText {
		if char == ' ' {
			if err := virtual_keyboard.TypeKey(keyboard, virtual_keyboard.KEY_SPACE); err != nil {
				return fmt.Errorf("failed to type space: %v", err)
			}
		} else if char >= 'A' && char <= 'Z' {
			key := virtual_keyboard.KEY_A + uint32(char - 'A')
			if err := virtual_keyboard.TypeKey(keyboard, key); err != nil {
				return fmt.Errorf("failed to type character: %v", err)
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	fmt.Println("     - Releasing Shift")
	if err := keyboard.KeyRelease(virtual_keyboard.KEY_LEFTSHIFT); err != nil {
		return fmt.Errorf("failed to release shift: %v", err)
	}

	// Type some text without shift
	if err := virtual_keyboard.TypeString(keyboard, " lowercase text"); err != nil {
		return fmt.Errorf("failed to type lowercase: %v", err)
	}

	fmt.Println("     - Testing Caps Lock")
	if err := virtual_keyboard.TypeKey(keyboard, virtual_keyboard.KEY_CAPSLOCK); err != nil {
		return fmt.Errorf("failed to press caps lock: %v", err)
	}

	if err := virtual_keyboard.TypeString(keyboard, " CAPS LOCK TEXT "); err != nil {
		return fmt.Errorf("failed to type caps lock text: %v", err)
	}

	// Turn off caps lock
	if err := virtual_keyboard.TypeKey(keyboard, virtual_keyboard.KEY_CAPSLOCK); err != nil {
		return fmt.Errorf("failed to release caps lock: %v", err)
	}

	if err := virtual_keyboard.TypeString(keyboard, "normal text again"); err != nil {
		return fmt.Errorf("failed to type normal text: %v", err)
	}

	return nil
}

// demonstrateAdvancedFeatures shows more advanced virtual keyboard features
func demonstrateAdvancedFeatures(keyboard virtual_keyboard.VirtualKeyboard) {
	fmt.Println("Advanced Keyboard Features:")

	// Demonstrate setting modifier state directly
	fmt.Println("   - Setting modifier states directly")
	modifierStates := []struct {
		mods uint32
		desc string
	}{
		{virtual_keyboard.MOD_SHIFT, "Shift only"},
		{virtual_keyboard.MOD_CTRL, "Ctrl only"},
		{virtual_keyboard.MOD_ALT, "Alt only"},
		{virtual_keyboard.MOD_CTRL | virtual_keyboard.MOD_SHIFT, "Ctrl+Shift"},
		{virtual_keyboard.MOD_CTRL | virtual_keyboard.MOD_ALT, "Ctrl+Alt"},
		{0, "No modifiers"},
	}

	for _, state := range modifierStates {
		fmt.Printf("     Setting modifiers: %s\n", state.desc)
		if err := virtual_keyboard.SetModifiers(keyboard, state.mods); err != nil {
			log.Printf("Error setting modifiers: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Demonstrate keypad input
	fmt.Println("   - Numeric keypad input")
	keypadKeys := []struct {
		key  uint32
		desc string
	}{
		{virtual_keyboard.KEY_NUMLOCK, "Num Lock"},
		{virtual_keyboard.KEY_KP7, "Keypad 7"},
		{virtual_keyboard.KEY_KP8, "Keypad 8"},
		{virtual_keyboard.KEY_KP9, "Keypad 9"},
		{virtual_keyboard.KEY_KPMINUS, "Keypad Minus"},
		{virtual_keyboard.KEY_KP4, "Keypad 4"},
		{virtual_keyboard.KEY_KP5, "Keypad 5"},
		{virtual_keyboard.KEY_KP6, "Keypad 6"},
		{virtual_keyboard.KEY_KPPLUS, "Keypad Plus"},
		{virtual_keyboard.KEY_KP1, "Keypad 1"},
		{virtual_keyboard.KEY_KP2, "Keypad 2"},
		{virtual_keyboard.KEY_KP3, "Keypad 3"},
		{virtual_keyboard.KEY_KP0, "Keypad 0"},
		{virtual_keyboard.KEY_KPDOT, "Keypad Dot"},
		{virtual_keyboard.KEY_KPENTER, "Keypad Enter"},
	}

	for _, key := range keypadKeys {
		fmt.Printf("     Keypad key: %s\n", key.desc)
		if err := virtual_keyboard.TypeKey(keyboard, key.key); err != nil {
			log.Printf("Error typing keypad key: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}