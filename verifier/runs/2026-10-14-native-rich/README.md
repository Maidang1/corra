# Native rich editor verification

Date: 2026-10-14. Local macOS arm64, cgo enabled.

## Automated evidence

- `go test ./... -count=1`: 213 reported test/subtest cases passed.
- `go test -race ./app ./views ./richtext ./nativerich -count=1`: 188 passed.
- `CGO_ENABLED=0 go test ./... -count=1`: 205 passed (platform fallback).
- `go vet ./...`, `go build ./...`, `git diff --check`: passed.
- `CGO_ENABLED=1 go tool mygo build -skip-dmg -o dist`: builds the native app.
- `codesign --verify --deep --strict dist/darwin-arm64/Corra.app`: passed.
- Packaged binary build metadata confirms `CGO_ENABLED=1`; contains the
  `CRTextView` Objective-C class, not merely the fallback bridge.

The eight native harness tests use actual NSTextView/NSWindow instances, main
thread AppKit, attributed storage, selected ranges, clipboard and undo manager.
They cover rich attributes, heading/body changes, CJK/emoji, marked text/commit,
list continuation and exit, checkbox mouseDown, protected block boundaries,
clipboard paragraph breaks, unsupported RTF rejection, selection preservation,
read-only state, layer-backed/flipped frames, and native focus restoration.
Invalid UTF-8 source bytes survive the native JSON boundary and adjacent edits;
codec regressions also cover exact link destinations after label changes.
The harness backs up/restores the system clipboard without logging its content.

## Images

`dark-editor.png` and `dark-editor-light.png` are real NSScrollView/NSTextView
offscreen bitmap captures at 1x text zoom and Retina image resolution. They show
system typography, styled text, a round interactive checklist and protected
source content. They are not full application/window screenshots and do not
prove pointer routing across the MyGo/native boundary by themselves.

Generate again with:

```sh
CORRA_NATIVE_SCREENSHOT="$PWD/verifier/runs/2026-10-14-native-rich/dark-editor.png" \
  go test ./nativerich -run '^TestNativeEditor$' -count=1
```

## Runtime and limits

The actual MyGo app launched with the native bridge in an isolated temporary
library, with no startup crash. The development app was then gracefully closed
and restarted via `make dev`; its binary metadata confirms cgo is enabled.
Window enumeration confirms a visible 1280x820 Corra Dev window.

Accessibility permission is not granted (`AXIsProcessTrusted=false`). No actual
IME candidate-window selection, OS-level mouse/keyboard injection, or full
window screenshot acceptance is claimed. Manually check Chinese candidate
selection, source switching, save/reopen, resize, and toolbar selection commands
in the running app. Complex tables/images/math remain protected source blocks;
the separate reading preview retains their renderer.
