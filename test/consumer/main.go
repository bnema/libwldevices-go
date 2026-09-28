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
	"github.com/bnema/wlturbo/protocol/core"
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
		"zwp_virtual_keyboard_manager_v1",
	}
	for _, iface := range requiredGlobals {
		g, ok := registry.FindGlobal(iface)
		check("global "+iface+" present", ok,
			fmt.Sprintf(" (name=%d version=%d)", g.Name, g.Version))
	}

	// (b2) Purpose-built fixture: a mapped xdg_toplevel with a known title and
	// app id and an shm buffer, which records the input the compositor delivers
	// to it. Injection only means something if a real client observes it.
	fx, ferr := startFixture(c)
	check("fixture xdg_toplevel mapped with a committed shm buffer", ferr == nil, errDetail(ferr))

	// (c) Virtual pointer: library-reported availability plus real events
	// observed by the fixture.
	if c.HasVirtualPointer() {
		check("client.HasVirtualPointer() reports zwlr_virtual_pointer_manager_v1", true, "")
	} else {
		fmt.Println("SKIP: compositor does not announce zwlr_virtual_pointer_manager_v1")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	var pointerManager *virtual_pointer.VirtualPointerManager
	if c.HasVirtualPointer() {
		pointerManager, err = virtual_pointer.NewVirtualPointerManager(ctx)
		check("virtual_pointer.NewVirtualPointerManager()", err == nil, errDetail(err))
	}
	var pointer *virtual_pointer.VirtualPointer
	if pointerManager != nil {
		pointer, err = pointerManager.CreatePointer()
		check("virtual_pointer.CreatePointer()", err == nil, errDetail(err))
	}
	if pointer != nil && fx != nil {
		runPointer(c, pointer, fx)
	}

	// (d) Virtual keyboard: construction sends the default keymap, then a real
	// key press/release pair that the fixture must observe.
	check("client.HasVirtualKeyboard() reports zwp_virtual_keyboard_manager_v1",
		c.HasVirtualKeyboard(), "")

	kctx, kcancel := context.WithTimeout(context.Background(), 30*time.Second)
	keyboardManager, err := virtual_keyboard.NewVirtualKeyboardManager(kctx)
	check("virtual_keyboard.NewVirtualKeyboardManager()", err == nil, errDetail(err))
	var keyboard *virtual_keyboard.VirtualKeyboard
	if err == nil {
		keyboard, err = keyboardManager.CreateKeyboard()
		check("virtual_keyboard.CreateKeyboard() (sends default keymap)",
			err == nil, errDetail(err))
	}
	if keyboard != nil && fx != nil {
		runKeyboard(c, keyboard, pointer, fx)
	}

	// (d2) The end-to-end assertion: the fixture's own observations prove the
	// injected pointer and keyboard events reached a client.
	if fx != nil {
		assertFixtureInput(c, fx, c.HasVirtualPointer())
	}

	if keyboardManager != nil {
		if cerr := keyboardManager.Close(); cerr != nil {
			fmt.Printf("WARN: virtual keyboard manager close: %v\n", cerr)
		}
	}
	kcancel()
	if pointerManager != nil {
		if cerr := pointerManager.Close(); cerr != nil {
			fmt.Printf("WARN: virtual pointer manager close: %v\n", cerr)
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

	_, ok := registry.FindGlobal("wl_compositor")
	if !ok {
		check("global wl_compositor present for the inhibitor test", false, "")
		return
	}

	compositor := core.NewCompositor(wlContext)
	_, err := registry.BindNegotiated("wl_compositor", 6, compositor)
	if err != nil {
		check("bind wl_compositor", false, errDetail(err))
		return
	}

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

// runPointer injects a pointer path and asserts the fixture observes it. The
// request-acceptance assertions are kept unchanged.
func runPointer(c *client.Client, pointer *virtual_pointer.VirtualPointer, fx *fixture) {
	now := time.Now()
	err := pointer.Motion(now, 12, -7)
	check("virtual pointer relative Motion() send", err == nil, errDetail(err))

	err = pointer.Frame()
	check("virtual pointer Frame() send", err == nil, errDetail(err))

	err = c.GetDisplay().Roundtrip()
	check("client connection healthy after pointer events", err == nil, errDetail(err))

	// The virtual pointer device is what grants the seat its pointer capability,
	// so the fixture can only create wl_pointer once that capability appears.
	if !fx.waitPointerCapability(c) {
		check("fixture observed wl_pointer capability", false, errDetail(fx.fixtureErr()))
		return
	}
	check("fixture observed wl_pointer capability", true, "")

	if err := fx.bindPointer(); err != nil {
		check("fixture wl_seat.get_pointer()", false, errDetail(err))
		return
	}
	check("fixture wl_seat.get_pointer()", true, "")

	// Warp to the centre of the output (motion_absolute coordinates are
	// normalised), nudge across the window, then click. Frame() batches each
	// pointer state.
	if err := pointer.MotionAbsolute(time.Now(), 1, 1, 2, 2); err != nil {
		check("virtual pointer absolute Motion() to window centre send", false, errDetail(err))
	} else {
		check("virtual pointer absolute Motion() to window centre send", true, "")
	}
	if err := pointer.Motion(time.Now(), 40, 30); err != nil {
		check("virtual pointer Motion() across the window send", false, errDetail(err))
	}
	if err := pointer.Frame(); err != nil {
		check("virtual pointer Frame() after motion send", false, errDetail(err))
	}
	if err := pointer.Button(time.Now(), virtual_pointer.BTN_LEFT, virtual_pointer.ButtonStatePressed); err != nil {
		check("virtual pointer button press send", false, errDetail(err))
	}
	if err := pointer.Frame(); err != nil {
		check("virtual pointer Frame() after press send", false, errDetail(err))
	}
	if err := pointer.Button(time.Now(), virtual_pointer.BTN_LEFT, virtual_pointer.ButtonStateReleased); err != nil {
		check("virtual pointer button release send", false, errDetail(err))
	}
	if err := pointer.Frame(); err != nil {
		check("virtual pointer Frame() after release send", false, errDetail(err))
	}

	ok := fx.pumpUntil(c, "fixture pointer observations", func() bool {
		o := fx.snapshot()
		return o.pointerMotion >= 1 && o.pointerButtonP >= 1 && o.pointerButtonR >= 1
	})
	o := fx.snapshot()
	check("fixture observed pointer motion, button press and release", ok,
		fmt.Sprintf(" (motion=%d button_press=%d button_release=%d axis=%d%s)",
			o.pointerMotion, o.pointerButtonP, o.pointerButtonR, o.pointerAxis, errDetail(fx.fixtureErr())))
}

// runKeyboard creates a virtual keyboard (which sends the default keymap) and
// sends a key press/release pair the fixture must observe.
func runKeyboard(c *client.Client, keyboard *virtual_keyboard.VirtualKeyboard, pointer *virtual_pointer.VirtualPointer, fx *fixture) {
	now := time.Now()
	// Key() refuses to send with "keymap not set" when the keymap upload did not
	// happen, so a successful send also proves the keymap was set.
	err := keyboard.Key(now, virtual_keyboard.KEY_A, virtual_keyboard.KeyStatePressed)
	check("virtual keyboard key press send", err == nil, errDetail(err))

	err = keyboard.Key(now, virtual_keyboard.KEY_A, virtual_keyboard.KeyStateReleased)
	check("virtual keyboard key release send", err == nil, errDetail(err))

	err = c.GetDisplay().Roundtrip()
	check("client connection healthy after key events", err == nil, errDetail(err))

	// The virtual keyboard device grants the seat its keyboard capability, so
	// the fixture creates wl_keyboard only after that capability appears.
	if !fx.waitKeyboardCapability(c) {
		check("fixture observed wl_keyboard capability", false, errDetail(fx.fixtureErr()))
		return
	}
	check("fixture observed wl_keyboard capability", true, "")

	if err := fx.bindKeyboard(); err != nil {
		check("fixture wl_seat.get_keyboard()", false, errDetail(err))
		return
	}
	check("fixture wl_seat.get_keyboard()", true, "")

	// Nudge the pointer so the compositor re-evaluates focus now that a
	// keyboard exists; a keyboard enter follows the focused surface.
	if pointer != nil {
		_ = pointer.Motion(time.Now(), 1, 1)
		_ = pointer.Frame()
	}
	if !fx.pumpUntil(c, "fixture keyboard focus", func() bool { return fx.snapshot().keyboardEnter >= 1 }) {
		check("fixture observed wl_keyboard enter (focused)", false, errDetail(fx.fixtureErr()))
		return
	}
	check("fixture observed wl_keyboard enter (focused)", true, "")

	// Re-inject now that the fixture holds keyboard focus, then require the
	// fixture itself to report the events.
	now = time.Now()
	err = keyboard.Key(now, virtual_keyboard.KEY_A, virtual_keyboard.KeyStatePressed)
	check("virtual keyboard focused key press send", err == nil, errDetail(err))
	err = keyboard.Key(now, virtual_keyboard.KEY_A, virtual_keyboard.KeyStateReleased)
	check("virtual keyboard focused key release send", err == nil, errDetail(err))

	ok := fx.pumpUntil(c, "fixture keyboard key events", func() bool {
		o := fx.snapshot()
		return o.keyboardKeyP >= 1 && o.keyboardKeyR >= 1
	})
	o := fx.snapshot()
	check("fixture observed keyboard key press and release", ok,
		fmt.Sprintf(" (key_press=%d key_release=%d modifiers=%d%s)",
			o.keyboardKeyP, o.keyboardKeyR, o.keyboardMods, errDetail(fx.fixtureErr())))

	_ = keyboard.Close()
}

// assertFixtureInput is the end-to-end gate: it fails, listing expected and
// observed events, when the injected input never reached the fixture client.
func assertFixtureInput(c *client.Client, fx *fixture, pointerAvailable bool) {
	// Give the compositor a final chance to flush anything still queued.
	_ = c.GetDisplay().Roundtrip()
	o := fx.snapshot()
	ok := (!pointerAvailable || (o.pointerMotion >= 1 && o.pointerButtonP >= 1 && o.pointerButtonR >= 1)) &&
		o.keyboardKeyP >= 1 && o.keyboardKeyR >= 1
	detail := fmt.Sprintf(
		" (observed pointer enter=%d leave=%d motion=%d button_press=%d button_release=%d axis=%d; keyboard enter=%d leave=%d key_press=%d key_release=%d modifiers=%d)",
		o.pointerEnter, o.pointerLeave, o.pointerMotion, o.pointerButtonP, o.pointerButtonR, o.pointerAxis,
		o.keyboardEnter, o.keyboardLeave, o.keyboardKeyP, o.keyboardKeyR, o.keyboardMods)
	if !ok {
		detail += "; expected key_press>=1, key_release>=1 and pointer events when advertised"
	}
	check("fixture observed injected pointer and keyboard input", ok, detail)
}
