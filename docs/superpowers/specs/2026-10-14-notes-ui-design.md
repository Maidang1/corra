# Corra Notes UI and Interaction Alignment

Approved in chat on 2026-10-14. Reference: user's second screenshot of Apple Notes.

## Scope
Keep Go/MyGo and Markdown storage/editor. Match the desktop Notes shell: unified
native titlebar with functioning traffic lights, inset rounded sidebar, soft
shadows, capsule toolbar groups, system typography, narrower note list and
compact document spacing. Preserve system window controls (no fake traffic lights).
At 1280 DIP use about 250 DIP for the sidebar and 225 DIP for the list; adapt the
search width at small windows. Light and dark must both remain readable.

## Behavior
All displayed actions must work or be disabled with a descriptive label.
Sidebar collapses. Local folders support create/rename/delete-empty, notes support
create/select/search/move/delete/restore. Search covers full source, not just the
truncated preview. Trash loads actual deleted notes, and is read-only until restore.
Format menu inserts Markdown into the active selection; checklist/table/attachment
and export use the same editor source. Provide edit/preview switching and autosave
for both editor and preview mutations. Switching folders/notes, deleting, export,
external opens, and closing must not discard pending edits or recreate deleted files.
Empty folders show an empty editor rather than an unrelated note.

## Boundaries
app/: filesystem operations, library indexing and note selection.
views/: presentation, command routing, editor synchronization and save lifecycle.
main.go/menu.go: native dialogs, attachments/export, window lifecycle and shortcuts.
No iCloud sync/collaboration, Apple account impersonation, or claims of a rich-text
editor: the full editor remains Markdown source, with a rendered reading view.

## Verification
Write failing regression tests before implementation. Use MyGo's Tester for real
hit testing and menu selection, assert changed state and on-disk content. Test
empty/trash/filtered states and pending saves. Run go test ./..., go test -race on
changed packages, go build ./.... Render light/dark at standard and minimum sizes.
Launch an isolated real app window with temporary library data; inspect screenshot
and exercise controls via native accessibility or mouse events where available.
