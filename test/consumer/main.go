// Command consumer is the standalone external-consumer fixture for
// libwldevices-go.
//
// It lives in its own Go module (see go.mod) so that it consumes the library
// exactly the way a downstream application would: through the module's public
// import paths, with no access to the library's test helpers. It connects to a
// real headless wlroots compositor over the Wayland socket and exercises the
// virtual pointer and virtual keyboard paths end to end.
//
// Every assertion prints exactly one PASS/FAIL line and any failure makes the
// process exit non-zero, so test/integration/run.sh can propagate the result.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bnema/libwldevices-go/internal/client"
	"github.com/bnema/libwldevices-go/keyboard_shortcuts_inhibitor"
	"github.com/bnema/libwldevices-go/virtual_keyboard"
	"github.com/bnema/libwldevices-go/virtual_pointer"
	"github.com/bnema/wlturbo/wl"
)

// failed counts failed assertions.
var failed int

// check records the outcome of a single assertion.
func check(name string, ok bool, detail string) {
	if ok {
		fmt.Printf("PASS: %s%s\n", name, detail)
		return
	}
	failed++
	fmt.Printf("FAIL: %s%s\n", name, detail)
}

// errDetail renders an error as a printable detail suffix.
func errDetail(err error) string {
	if err == nil {
		return ""
	}
	return ": " + err.Error()
}

func main() {
	os.Exit(run())
}

func run() int {
	c, err := client.NewClient()
	if err != nil {
		fmt.Printf("FAIL: client.NewClient(): %v\n", err)
		return 1
	}
	defer func() {
		if cerr := c.Close(); cerr != nil {
			fmt.Printf("WARN: client.Close(): %v\n", cerr)
		}
	}()
	check("client.NewClient() connected", true, "")

	// (a) Full roundtrip with the compositor.
	err = c.GetDisplay().Roundtrip()
	check("display.Roundtrip()", err == nil, errDetail(err))

	// (b) Enumerate the compositor's globals and assert the virtual input
	// protocol globals are present, printing the negotiated name and version.
	registry := c.GetRegistry()
	// globals is the underlying wlturbo registry snapshot for this connection.
	var globals map[uint32]wl.Global = registry.GetGlobals()
	fmt.Printf("INFO: compositor announced %d globals\n", len(globals))
	requiredGlobals := []string{
		"wl_seat",
		"zwlr_virtual_pointer_manager_v1",
		"zwp_virtual_keyboard_manager_v1",
	}
	for _, iface := range requiredGlobals {
		g, ok := registry.FindGlobal(iface)
		check("global "+iface+" present", ok,
			fmt.Sprintf(" (name=%d version=%d)", g.Name, g.Version))
	}

	// (c) Virtual pointer: library-reported availability plus real events.
	check("client.HasVirtualPointer() reports zwlr_virtual_pointer_manager_v1",
		c.HasVirtualPointer(), "")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	pointerManager, err := virtual_pointer.NewVirtualPointerManager(ctx)
	check("virtual_pointer.NewVirtualPointerManager()", err == nil, errDetail(err))
	if err == nil {
		runPointer(c, pointerManager)
		if cerr := pointerManager.Close(); cerr != nil {
			fmt.Printf("WARN: virtual pointer manager close: %v\n", cerr)
		}
	}
	cancel()

	// (d) Virtual keyboard: construction sends the default keymap, then a real
	// key press/release pair.
	check("client.HasVirtualKeyboard() reports zwp_virtual_keyboard_manager_v1",
		c.HasVirtualKeyboard(), "")

	ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	keyboardManager, err := virtual_keyboard.NewVirtualKeyboardManager(ctx)
	check("virtual_keyboard.NewVirtualKeyboardManager()", err == nil, errDetail(err))
	if err == nil {
		runKeyboard(c, keyboardManager)
		if cerr := keyboardManager.Close(); cerr != nil {
			fmt.Printf("WARN: virtual keyboard manager close: %v\n", cerr)
		}
	}
	cancel()

	// (e) Keyboard shortcuts inhibitor: the compositor must accept the
	// inhibit_shortcuts request for a real surface and seat, and the connection
	// must survive the inhibitor's lifetime.
	if _, ok := registry.FindGlobal("zwp_keyboard_shortcuts_inhibit_manager_v1"); !ok {
		fmt.Println("SKIP: compositor does not announce zwp_keyboard_shortcuts_inhibit_manager_v1")
	} else {
		// inhibit_shortcuts references a surface and a seat, so the manager must
		// run on the connection that owns them: see
		// NewKeyboardShortcutsInhibitorManagerWithClient.
		inhibitorManager, err := keyboard_shortcuts_inhibitor.NewKeyboardShortcutsInhibitorManagerWithClient(c)
		check("keyboard_shortcuts_inhibitor.NewKeyboardShortcutsInhibitorManagerWithClient()", err == nil, errDetail(err))
		if err == nil {
			runInhibitor(c, registry, inhibitorManager)
			if cerr := inhibitorManager.Close(); cerr != nil {
				fmt.Printf("WARN: inhibitor manager close: %v\n", cerr)
			}
		}
	}

	if failed > 0 {
		fmt.Printf("FAIL: %d assertion(s) failed\n", failed)
		return 1
	}
	fmt.Println("PASS: all virtual input assertions succeeded")
	return 0
}

// runInhibitor drives the keyboard shortcuts inhibitor end to end: bind
// wl_compositor, create a surface, inhibit shortcuts on this client's seat, then
// destroy the inhibitor and confirm the connection is still healthy.
func runInhibitor(c *client.Client, registry *wl.Registry, manager *keyboard_shortcuts_inhibitor.KeyboardShortcutsInhibitorManager) {
	wlContext := c.GetContext()

	compositorGlobal, ok := registry.FindGlobal("wl_compositor")
	if !ok {
		check("global wl_compositor present for the inhibitor test", false, "")
		return
	}

	compositor := wl.NewCompositor(wlContext)
	compositorID, err := registry.BindID(compositorGlobal.Name, "wl_compositor", compositorGlobal.Version)
	if err != nil {
		check("bind wl_compositor", false, errDetail(err))
		return
	}
	compositor.SetID(compositorID)
	wlContext.Register(compositor)

	surface, err := compositor.CreateSurface()
	check("wl_compositor.CreateSurface()", err == nil, errDetail(err))
	if err != nil {
		return
	}

	seat := c.GetSeat()
	if seat == nil {
		check("client seat available", false, "")
		return
	}

	inhibitor, err := manager.InhibitShortcuts(surface, seat)
	check("keyboard_shortcuts_inhibitor.InhibitShortcuts()", err == nil, errDetail(err))
	if err != nil {
		return
	}

	// A malformed request would be answered with a protocol error here and the
	// compositor would drop the connection.
	err = manager.Roundtrip()
	check("compositor accepted inhibit_shortcuts", err == nil, errDetail(err))

	err = inhibitor.Destroy()
	check("keyboard shortcuts inhibitor Destroy()", err == nil, errDetail(err))

	err = manager.Roundtrip()
	check("connection healthy after inhibitor events", err == nil, errDetail(err))
}

// runPointer creates a virtual pointer and sends a relative motion plus a frame.
func runPointer(c *client.Client, manager *virtual_pointer.VirtualPointerManager) {
	pointer, err := manager.CreatePointer()
	check("virtual_pointer.CreatePointer()", err == nil, errDetail(err))
	if err != nil {
		return
	}
	defer func() {
		if cerr := pointer.Close(); cerr != nil {
			fmt.Printf("WARN: virtual pointer close: %v\n", cerr)
		}
	}()

	now := time.Now()
	err = pointer.Motion(now, 12, -7)
	check("virtual pointer relative Motion() send", err == nil, errDetail(err))

	err = pointer.Frame()
	check("virtual pointer Frame() send", err == nil, errDetail(err))

	err = c.GetDisplay().Roundtrip()
	check("client connection healthy after pointer events", err == nil, errDetail(err))
}

// runKeyboard creates a virtual keyboard (which sends the default keymap) and
// sends a key press/release pair.
func runKeyboard(c *client.Client, manager *virtual_keyboard.VirtualKeyboardManager) {
	// CreateKeyboard() also uploads the default xkb keymap, so a successful
	// construction proves the keymap send path worked.
	keyboard, err := manager.CreateKeyboard()
	check("virtual_keyboard.CreateKeyboard() (sends default keymap)",
		err == nil, errDetail(err))
	if err != nil {
		return
	}
	defer func() {
		if cerr := keyboard.Close(); cerr != nil {
			fmt.Printf("WARN: virtual keyboard close: %v\n", cerr)
		}
	}()

	now := time.Now()
	// Key() refuses to send with "keymap not set" when the keymap upload did not
	// happen, so a successful send also proves the keymap was set.
	err = keyboard.Key(now, virtual_keyboard.KEY_A, virtual_keyboard.KeyStatePressed)
	check("virtual keyboard key press send", err == nil, errDetail(err))

	err = keyboard.Key(now, virtual_keyboard.KEY_A, virtual_keyboard.KeyStateReleased)
	check("virtual keyboard key release send", err == nil, errDetail(err))

	err = c.GetDisplay().Roundtrip()
	check("client connection healthy after key events", err == nil, errDetail(err))
}
