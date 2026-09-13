# xdgshell

Generated client bindings for the stable `xdg-shell` protocol. The generator
lives in this repository (`scanner/cmd/wayland-scanner`).

`xdg_shell.go` is generated output and must not be edited by hand. Regenerate it
with:

```sh
go run ./scanner/cmd/wayland-scanner \
  -p xdgshell \
  -o internal/xdgshell/xdg_shell.go \
  /usr/share/wayland-protocols/stable/xdg-shell/xdg-shell.xml
```

Only the package name and output path are repository choices; the XML path is
the distribution's copy of the stable protocol (`wayland-protocols` package).
