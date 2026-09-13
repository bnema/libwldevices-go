# Vendored Wayland protocol XML

These XML files are vendored verbatim so bindings can be regenerated on a
machine without `wayland-protocols` installed.

| File | Origin | Version |
| --- | --- | --- |
| `keyboard-shortcuts-inhibit-unstable-v1.xml` | `/usr/share/wayland-protocols/unstable/keyboard-shortcuts-inhibit/keyboard-shortcuts-inhibit-unstable-v1.xml` (wayland-protocols) | 1.49 |

Regenerate the Go bindings with:

```sh
go run ./scanner/cmd/wayland-scanner -p protocols -o internal/protocols/keyboard_shortcuts_inhibit.go protocol/keyboard-shortcuts-inhibit-unstable-v1.xml
```
