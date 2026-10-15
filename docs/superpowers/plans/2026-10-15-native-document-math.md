# Native document math integration

Parent goal: implement every capability in `docs/typora-parity.md`, macOS only,
without WebView. This delivery does not complete that goal.

The embedded MathJax engine already provides document batches. Native editing
must render the full ordered equation context, even while a source span is
expanded or a local table/paragraph fragment is being rebuilt. Rendering must
preserve native text, selection, undo snapshots and marked IME text.

- [x] Collect math in document order, including table cells and expanded source.
- [x] Share macros, labels and forward/back references across native equations.
- [x] Provide none / AMS / all numbering choices and a forced refresh command.
- [x] Retain prefix/suffix context for input rules, insertion, clipboard, table
  rebuilding and edited source folding.
- [x] Refresh attachment attributes without rebuilding the text buffer or
  mutating images held by undo snapshots. Cache by equations and display options.
- [x] Add inline math source expansion/folding and Cmd+Return completion.
- [x] Preserve live editable source when a formula fails; support correction.
- [x] Parse raw inline TeX before Markdown escape/emphasis/entity handling;
  share delimiter rules between native and reading models and preserve GFM
  table pipe escaping during formula edits and table structure changes.
- [x] Review source-loss, selection, focus, context-order and attribute issues.
- [ ] Unify reading preview, PDF and HTML export with document batches/options.
  HTML now embeds offline SVG for inline/display equations with shared context;
  reading preview and PDF share document batches for display equations. Inline
  equation images in reading preview and PDF remain pending.
- [ ] Add Typora's math preferences: physics option, inline parsing compatibility,
  and optional legacy line breaking. Audit the supported macro inventory.
- [ ] Validate live Chinese IME candidate selection and large-document latency.
- [x] Fix the existing table code-span serializer case that removes a backslash
  before `|` after table edits. Independent review checked repeated round trips,
  embedded backticks and adjacent math cells.

Verification uses actual NSTextView instances, ordered PNG comparisons against
the batch renderer, and serializer round trips. The native screenshot can be
regenerated with:

```sh
CORRA_MATH_SCREENSHOT=/tmp/corra-native-math.png go test ./nativerich -run TestNativeDocumentMathNumbering -count=1
```

Reference: [Typora math documentation](https://support.typora.io/Math/), inspected
2026-10-15. It specifies shared macros, chemistry/physics, labels/references,
three numbering modes, forced refresh, and additional parsing/line-break options.

## Current verification

- `go test ./... -count=1 -timeout=180s`: 403 passing cases in 13 packages.
- `go build ./...`: successful on macOS arm64 with cgo.
- `go test -race ./app ./render ./richtext -count=1 -timeout=180s`:
  270 passing cases in 3 packages.
- Native numbered-equation screenshot generated and visually inspected at
  `/tmp/corra-native-math.png`; forward reference and two independent equation
  numbers are visible, with display math centered in the writing column.
- Manual Chinese IME interaction and large-document typing latency remain
  unverified; the full parity goal remains active.
- HTML regression tests preserve single-line display math and following prose,
  opening-line TeX, literal image alt text and mid-paragraph display delimiters.
  Image alt text does not contribute hidden macros to the document context.
- PDF tests verify both equations are embedded images with shared macros and
  references. They do not certify PDF font embedding or visual layout fidelity.
- Independent final parser/serializer review found no new regression; its
  additional fixtures covered TeX pipes with zero through three backslashes,
  `\left\|...\right\|`, text macros, escaped dollars and adjacent cells.
