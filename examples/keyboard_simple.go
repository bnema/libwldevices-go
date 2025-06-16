// Simple keyboard input example
//
// This example demonstrates basic virtual keyboard functionality:
// - Creating a virtual keyboard manager and device
// - Typing individual keys
// - Typing strings
// - Using modifier keys
//
// Run with: go run examples/keyboard_simple.go
// Make sure to click in a text field or terminal to see the output!
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bnema/wayland-virtual-input-go/virtual_keyboard"
)

func main() {
	fmt.Println("Simple Keyboard Input Example")
	fmt.Println("=============================")
	
	ctx := context.Background()
	
	// Create virtual keyboard manager
	fmt.Print("Creating virtual keyboard manager... ")
	manager, err := virtual_keyboard.NewVirtualKeyboardManager(ctx)
	if err != nil {
		log.Fatalf("FAILED: %v", err)
	}
	defer manager.Close()
	fmt.Println("OK")
	
	// Create virtual keyboard
	fmt.Print("Creating virtual keyboard... ")
	keyboard, err := manager.CreateKeyboard()
	if err != nil {
		log.Fatalf("FAILED: %v", err)
	}
	defer keyboard.Close()
	fmt.Println("OK")
	
	fmt.Println("\n⚠️  IMPORTANT: Click in a text field or terminal window to see the keyboard input!")
	fmt.Println("Starting keyboard demonstrations in 3 seconds...")
	
	for i := 3; i > 0; i-- {
		fmt.Printf("%d... ", i)
		time.Sleep(1 * time.Second)
	}
	fmt.Println("GO!")
	
	// Demonstration 1: Type a simple string
	fmt.Println("\n1. Typing: 'Hello Wayland!'")
	err = keyboard.TypeString("Hello Wayland!")
	if err != nil {
		log.Printf("TypeString failed: %v", err)
	}
	time.Sleep(500 * time.Millisecond)
	
	// Demonstration 2: Press Enter
	fmt.Println("\n2. Pressing Enter key")
	err = keyboard.TypeKey(virtual_keyboard.KEY_ENTER)
	if err != nil {
		log.Printf("Enter key failed: %v", err)
	}
	time.Sleep(500 * time.Millisecond)
	
	// Demonstration 3: Type mixed case
	fmt.Println("\n3. Typing: 'MiXeD CaSe TeXt'")
	err = keyboard.TypeString("MiXeD CaSe TeXt")
	if err != nil {
		log.Printf("Mixed case failed: %v", err)
	}
	time.Sleep(500 * time.Millisecond)
	
	// Demonstration 4: Numbers and symbols
	fmt.Println("\n4. Typing numbers: '12345'")
	err = keyboard.TypeString("12345")
	if err != nil {
		log.Printf("Numbers failed: %v", err)
	}
	time.Sleep(500 * time.Millisecond)
	
	// Demonstration 5: Individual key presses
	fmt.Println("\n5. Individual key presses: 'abc' (one by one)")
	keys := []uint32{virtual_keyboard.KEY_A, virtual_keyboard.KEY_B, virtual_keyboard.KEY_C}
	for _, key := range keys {
		err = keyboard.TypeKey(key)
		if err != nil {
			log.Printf("Key %d failed: %v", key, err)
		}
		time.Sleep(200 * time.Millisecond)
	}
	
	// Demonstration 6: Space and punctuation
	fmt.Println("\n6. Adding space and punctuation")
	keyboard.TypeKey(virtual_keyboard.KEY_SPACE)
	time.Sleep(100 * time.Millisecond)
	keyboard.TypeKey(virtual_keyboard.KEY_ENTER)
	
	fmt.Println("\nExample completed! All keyboard input was sent to the active window.")
	fmt.Println("If you didn't see any text input, make sure:")
	fmt.Println("- You clicked in a text field or terminal")
	fmt.Println("- Your compositor supports zwp_virtual_keyboard_v1 protocol")
	fmt.Println("- The application has focus and can receive keyboard input")
}