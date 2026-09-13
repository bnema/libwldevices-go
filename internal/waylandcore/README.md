# waylandcore

Generated client bindings for the core `wayland` protocol (`wl_seat`,
`wl_pointer`, `wl_keyboard`, `wl_shm`, `wl_surface`, and so on).

The consumer fixture (`test/consumer`) uses these bindings to receive input
events: the transport's own `wl.Pointer`/`wl.Keyboard` wrappers carry no event
dispatch, while the generated bindings decode each event into typed handlers.

`core.go` is generated output and must not be edited by hand. Regenerate it
with:

```sh
go run ./scanner/cmd/wayland-scanner \
  -p waylandcore \
  -o internal/waylandcore/core.go \
  /usr/share/wayland/wayland.xml
```
