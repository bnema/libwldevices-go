// Minimal virtual pointer test for basic functionality verification
//
// This is the simplest possible test to verify that virtual pointer
// functionality is working. It only tests mouse movement to help
// debug protocol communication issues.
//
// Prerequisites:
// - Wayland compositor with virtual pointer support
// - Active Wayland session
//
// Usage: go run tests/minimal/main.go
// Debug: WAYLAND_DEBUG=1 go run tests/minimal/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bnema/wayland-virtual-input-go/virtual_pointer"
)

func main() {
	fmt.Println("Minimal Virtual Pointer Test")
	fmt.Printf("WAYLAND_DISPLAY: %s\n\n", os.Getenv("WAYLAND_DISPLAY"))

	// Set WAYLAND_DEBUG=1 to see protocol messages
	if os.Getenv("WAYLAND_DEBUG") == "1" {
		fmt.Println("WAYLAND_DEBUG is enabled - you'll see protocol messages")
	}

	ctx := context.Background()

	// Step 1: Create manager
	fmt.Print("Creating virtual pointer manager... ")
	manager, err := virtual_pointer.NewVirtualPointerManager(ctx)
	if err != nil {
		log.Fatalf("FAILED: %v", err)
	}
	fmt.Println("OK")
	defer manager.Close()

	// Step 2: Create pointer
	fmt.Print("Creating virtual pointer... ")
	pointer, err := manager.CreatePointer()
	if err != nil {
		log.Fatalf("FAILED: %v", err)
	}
	fmt.Println("OK")
	defer pointer.Close()

	// Step 3: Wait before moving
	fmt.Println("\nWaiting 2 seconds before moving mouse...")
	fmt.Println("Watch your cursor - it should move!")
	time.Sleep(2 * time.Second)

	// Step 4: Move mouse
	fmt.Print("Sending mouse movement (100, 100)... ")
	err = pointer.Motion(time.Now(), 100.0, 100.0)
	if err != nil {
		fmt.Printf("FAILED: %v\n", err)
	} else {
		fmt.Println("OK")
	}

	// Step 5: Frame the event
	fmt.Print("Sending frame... ")
	err = pointer.Frame()
	if err != nil {
		fmt.Printf("FAILED: %v\n", err)
	} else {
		fmt.Println("OK")
	}

	fmt.Println("\nDid the mouse move? If not:")
	fmt.Println("1. Check if your compositor supports zwlr_virtual_pointer_v1")
	fmt.Println("2. Run 'wayland-info | grep virtual_pointer' to verify")
	fmt.Println("3. Try running with WAYLAND_DEBUG=1 to see protocol messages")
	fmt.Println("4. Some compositors may require specific permissions")
}