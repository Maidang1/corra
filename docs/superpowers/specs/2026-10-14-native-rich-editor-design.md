# Native rich text editor

Approved by user in chat: default Apple Notes-like WYSIWYG on macOS, optional
Markdown source mode, existing Go/MyGo shell and plain Markdown storage retained.

## Architecture
- New richtext/ package: platform-independent Markdown <-> editable paragraphs
  with UTF-16 attributed ranges, plus raw opaque blocks for constructs that cannot
  round-trip safely. Do not serialize opaque source from its visual label.
- New nativerich/ package: AppKit NSTextView in an NSScrollView attached to the
  existing NSWindow; Go cgo bridge, Objective-C implementation, non-darwin/no-cgo
  fallback. Native editing handles caret, selection, IME, clipboard and undo.
- views/ connects the native editor to App.Source/autosave. A separate rich mode
  defaults only when native bridge is available; source/preview stay available.
  Sync native buffer before source switch/save/export/open and hide native surface
  under MyGo dialogs. Native selection survives ordinary app redraw/autosave.
- Native toolbar actions change selected text/paragraph attributes; list return
  continues the list and an empty item exits it. Checklists are interactive.
- Unsupported Markdown (tables/math/diagrams/code or constructs not losslessly
  modeled) remains a protected preview/attachment block with original raw source,
  opening source mode for editing. Never silently discard syntax.

## Acceptance
Go round-trip tests including CJK/emoji UTF-16, rich span formatting, list metadata,
opaque blocks and blank paragraphs. Native AppKit harness tests NSTextView edits,
formatting, undo, selection, checklist and serialization without OS input injection.
Go view tests for native routing via fake interface, persistence/source toggles and
modal visibility. Build app and inspect screenshots; report any IME/manual limits.
