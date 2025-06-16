# Examples

This directory contains working examples demonstrating the wayland-virtual-input-go library.

## Prerequisites

- Wayland compositor with virtual input protocol support (Sway, Hyprland, wlroots-based)
- Go 1.19 or later
- Active Wayland session (`XDG_SESSION_TYPE=wayland`)

## Running Examples

### 1. Simple Mouse Control (`mouse_simple.go`)

Demonstrates basic mouse operations: movement, clicking, and scrolling.

```bash
go run examples/mouse_simple.go
```

**What it does:**
- Creates virtual pointer device
- Moves mouse cursor in patterns
- Performs left and right clicks
- Scrolls vertically

### 2. Simple Keyboard Input (`keyboard_simple.go`)

Demonstrates basic keyboard operations: typing text, individual keys, and special keys.

```bash
go run examples/keyboard_simple.go
```

**What it does:**
- Creates virtual keyboard device
- Types strings with mixed case
- Presses individual keys
- Handles special keys (Enter, Space)

**Important:** Click in a text field or terminal before running to see the input!

### 3. Combined Demo (`combined_demo.go`)

Advanced example showing mouse and keyboard working together to automate a complete workflow.

```bash
go run examples/combined_demo.go
```

**What it does:**
- Opens context menu with right-click
- Navigates menu with mouse
- Opens text editor via keyboard commands
- Types a complete document
- Saves and exits using keyboard shortcuts

## Troubleshooting

### No mouse movement or keyboard input

1. **Check protocol support:**
   ```bash
   wayland-info | grep -E "(virtual_pointer|virtual_keyboard)"
   ```
   Should show `zwlr_virtual_pointer_manager_v1` and `zwp_virtual_keyboard_manager_v1`

2. **Verify Wayland session:**
   ```bash
   echo $XDG_SESSION_TYPE
   ```
   Should output `wayland`

3. **Check compositor:**
   - ✅ Sway, Hyprland, wlroots-based: Full support
   - ⚠️ GNOME, KDE: Limited support
   - ❓ Others: Check individual compositor documentation

4. **Enable debug output:**
   ```bash
   WAYLAND_DEBUG=1 go run examples/mouse_simple.go
   ```

### Permission issues

Some compositors may require special permissions for virtual input devices. Try:

1. Running from a terminal (not from a sandboxed environment)
2. Checking if your user has necessary permissions
3. Looking at compositor-specific documentation for virtual input

## Customization

All examples can be modified to suit your needs:

- **Timing**: Adjust `time.Sleep()` durations
- **Coordinates**: Change mouse movement distances
- **Text**: Modify typed content
- **Keys**: Use different key combinations

See the main library documentation for complete API reference.

## Example Output

When working correctly, you should see:

```
Simple Mouse Control Example
============================
Creating virtual pointer manager... OK
Creating virtual pointer... OK

Starting demonstrations in 2 seconds...
Watch your mouse cursor!

1. Moving mouse 100px right, 50px down
2. Drawing a small circle with mouse movement
3. Left click
4. Right click
5. Scroll down
6. Scroll up

Example completed! All mouse operations were sent to the compositor.
```

The mouse cursor should visibly move, and clicks/scrolls should register in the active application.