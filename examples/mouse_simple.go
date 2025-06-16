// Simple mouse control example
//
// This example demonstrates basic virtual pointer functionality:
// - Creating a virtual pointer manager and device
// - Moving the mouse cursor
// - Clicking mouse buttons
// - Scrolling
//
// Run with: go run examples/mouse_simple.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bnema/wayland-virtual-input-go/virtual_pointer"
)

func main() {
	fmt.Println("Simple Mouse Control Example")
	fmt.Println("============================")
	
	ctx := context.Background()
	
	// Create virtual pointer manager
	fmt.Print("Creating virtual pointer manager... ")
	manager, err := virtual_pointer.NewVirtualPointerManager(ctx)
	if err != nil {
		log.Fatalf("FAILED: %v", err)
	}
	defer manager.Close()
	fmt.Println("OK")
	
	// Create virtual pointer
	fmt.Print("Creating virtual pointer... ")
	pointer, err := manager.CreatePointer()
	if err != nil {
		log.Fatalf("FAILED: %v", err)
	}
	defer pointer.Close()
	fmt.Println("OK")
	
	fmt.Println("\nStarting demonstrations in 2 seconds...")
	fmt.Println("Watch your mouse cursor!")
	time.Sleep(2 * time.Second)
	
	// Demonstration 1: Simple movement
	fmt.Println("\n1. Moving mouse 100px right, 50px down")
	err = pointer.MoveRelative(100.0, 50.0)
	if err != nil {
		log.Printf("Movement failed: %v", err)
	}
	time.Sleep(1 * time.Second)
	
	// Demonstration 2: Circle movement
	fmt.Println("\n2. Drawing a small circle with mouse movement")
	for i := 0; i < 8; i++ {
		angle := float64(i) * 3.14159 / 4 // 8 points around circle
		dx := 20.0 * float64(i%2*2-1) // Simple pattern
		dy := 20.0 * float64((i/2)%2*2-1)
		pointer.MoveRelative(dx, dy)
		time.Sleep(200 * time.Millisecond)
	}
	
	// Demonstration 3: Mouse clicks
	fmt.Println("\n3. Left click")
	err = pointer.LeftClick()
	if err != nil {
		log.Printf("Left click failed: %v", err)
	}
	time.Sleep(500 * time.Millisecond)
	
	fmt.Println("4. Right click") 
	err = pointer.RightClick()
	if err != nil {
		log.Printf("Right click failed: %v", err)
	}
	time.Sleep(500 * time.Millisecond)
	
	// Demonstration 4: Scrolling
	fmt.Println("\n5. Scroll down")
	err = pointer.ScrollVertical(3.0)
	if err != nil {
		log.Printf("Scroll down failed: %v", err)
	}
	time.Sleep(500 * time.Millisecond)
	
	fmt.Println("6. Scroll up")
	err = pointer.ScrollVertical(-3.0)
	if err != nil {
		log.Printf("Scroll up failed: %v", err)
	}
	
	fmt.Println("\nExample completed! All mouse operations were sent to the compositor.")
	fmt.Println("If you didn't see mouse movement, your compositor may not support")
	fmt.Println("the zwlr_virtual_pointer_v1 protocol.")
}