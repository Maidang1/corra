# Notes UI alignment verification (2026-10-14)

## Automated acceptance

Commands (run from repository root):

```sh
CORRA_SCREENSHOTS="$PWD/verifier/runs/2026-10-14-notes-ui" go test ./... -count=1
go test -race ./app ./views -count=1
go vet ./...
go build ./...
go tool mygo build -skip-dmg -o dist
git diff --check
```

Screenshot fixtures are headless MyGo rasterizations, not native-window captures.
`light960.png`, `light1280.png`, `dark960.png`, `dark1280.png` cover minimum and
standard widths; collapsed sidebar geometry is tested separately. Native window
controls are deliberately absent in headless screenshots (the OS draws them).

Assertions cover toolbar/menu hit targets, formatting, typing and UI-thread
scheduled autosave, current-buffer copy/export callbacks, attachment callback,
folder dialogs, full-text search, empty selection, delete confirmation/cancel,
trash restore/read-only, safe failed-save navigation, source-preserving nested
checkboxes, source navigation from structured previews, and collapsed toolbar.
Backend tests cover file selection, watcher delivery/cancellation, atomic writes,
folder CRUD, safe paths/collisions, attachment rebasing and async image publication.

## Native evidence and limitation

Launched `/tmp/corra-verify-1014/Corra` with
`CORRA_CONFIG_DIR=/tmp/corra-verify-1014/config` (isolated fixture library).
CGWindowList reported a visible Corra 1280x820 window. No real user library was used.

System-level mouse injection and capture could not be validated:
`AXIsProcessTrusted()` returned false; `screencapture -l <window>` returned
`could not create image from window`. Therefore this run does not claim that
AppKit event delivery, traffic-light placement or native dialog interaction has
been verified. The signed app bundle is available at
`dist/darwin-arm64/Corra.app` for manual acceptance.

## Boundaries

- Markdown source editor + reading view, not a native rich-text Notes editor.
- Local files only; no iCloud/collaboration/Apple account integration.
- Folder deletion only for empty folders; restore targets the library root.
- Existing historical trash links may already lack their original base directory.
- Renaming a folder does not rebase links pointing outside its subtree.
- Atomic-save final-stat/rename and relocation final-stat/remove checks are
  best effort against another process changing a file at exactly that instant.
- Native system dialogs, real PDF dialog save and live OS input require manual
  acceptance; callback routing and PDF rendering are tested independently.
