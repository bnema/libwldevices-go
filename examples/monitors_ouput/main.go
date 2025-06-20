// Example program demonstrating output management functionality
// This program outputs monitor information in a format similar to wlr-randr
package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/bnema/libwldevices-go/output_management"
)

func main() {
	// Create context
	ctx := context.Background()

	// Create output manager
	manager, err := output_management.NewOutputManager(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create output manager: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = manager.Close() }()

	// Output events should be processed immediately - no sleep needed

	// Get all output heads
	heads := manager.GetHeads()
	if len(heads) == 0 {
		fmt.Println("No outputs found")
		return
	}
	

	// Sort heads by name for consistent output
	sort.Slice(heads, func(i, j int) bool {
		return heads[i].Name < heads[j].Name
	})

	// Print information for each head
	for i, head := range heads {
		// Print head name and description
		fmt.Printf("%s", head.Name)
		if head.Description != "" {
			fmt.Printf(" \"%s\"", head.Description)
		}
		fmt.Println()

		// Print make, model, serial if available
		if head.Make != "" {
			fmt.Printf("  Make: %s\n", head.Make)
		}
		if head.Model != "" {
			fmt.Printf("  Model: %s\n", head.Model)
		}
		if head.SerialNumber != "" {
			fmt.Printf("  Serial: %s\n", head.SerialNumber)
		}

		// Print physical size
		if head.PhysicalSize.Width > 0 && head.PhysicalSize.Height > 0 {
			fmt.Printf("  Physical size: %dx%d mm\n", head.PhysicalSize.Width, head.PhysicalSize.Height)
		}

		// Print enabled status
		fmt.Printf("  Enabled: %s\n", boolToYesNo(head.Enabled))

		// Print modes
		modes := head.GetModes()
		if len(modes) > 0 {
			fmt.Println("  Modes:")
			for _, mode := range modes {
				fmt.Printf("    %dx%d px, %.6f Hz", mode.Width, mode.Height, mode.GetRefreshRate())
				
				markers := []string{}
				if mode.Preferred {
					markers = append(markers, "preferred")
				}
				if head.CurrentMode != nil && mode == head.CurrentMode {
					markers = append(markers, "current")
				}
				
				if len(markers) > 0 {
					fmt.Printf(" (%s)", strings.Join(markers, ", "))
				}
				fmt.Println()
			}
		}

		// Print position
		fmt.Printf("  Position: %d,%d\n", head.Position.X, head.Position.Y)

		// Print transform
		fmt.Printf("  Transform: %s\n", head.Transform.String())

		// Print scale
		fmt.Printf("  Scale: %.6f\n", head.Scale)

		// Print adaptive sync (hardcoded to disabled for compatibility)
		fmt.Println("  Adaptive Sync: disabled")

		// Add blank line between outputs (except last one)
		if i < len(heads)-1 {
			fmt.Println()
		}
	}
}

func boolToYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// Note: AdaptiveSync helper function removed as the type is not available