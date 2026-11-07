# MyGo editor probe

Isolated from the GPUiX proof of concept. It pins `github.com/egoist/mygo` v0.8.3
and drives `ui/richtext`, MyGo's native attributed-text editor.

```sh
export PATH="$HOME/sdk/go/bin:$PATH"   # Go 1.27.3; the image Go is older
cd probes/mygo-editor
go test -count=1
```

`go run .` opens a window when a display and GTK 3 are available. This probe
does not use the webview, so WebKitGTK is not the editor.

Simulated input-method composition uses `ui.Tester.Compose`. That calls the
editor's composition callbacks. It is not a Japanese OS IME.
