# Keyboard Shortcuts

Octobud is designed for keyboard-first navigation. Press `h` at any time to see available shortcuts.

## Navigation

| Shortcut | Action |
|----------|--------|
| `j` / `k` | Navigate notification list |
| `Space` | Toggle notification detail |
| `Shift + p` | Toggle split view |
| `gg` | Jump to first notification |
| `Shift + g` | Jump to last notification |
| `[` / `]` | Navigate pages |
| `Shift + j` / `Shift + k` | Navigate views |
| `/` | Focus query input |
| `Cmd + b` | Toggle sidebar |

**Conditional Behaviors:**
- **`[` / `]` (Navigate pages)**: Disabled when detail view is open in list mode (not split view)
- **`Cmd + b`**: In multiselect mode, opens bulk command palette instead of toggling sidebar

## Command Palette

| Shortcut | Action |
|----------|--------|
| `Shift + Cmd + k` | Open command palette (empty) |
| `Cmd + k` | Open command palette (prepopulated with view switcher) |
| `Shift + v` | Open bulk actions command palette |

## Actions

These shortcuts work on the currently focused notification and toggle the action state.

| Shortcut | Action |
|----------|--------|
| `s` | Toggle star/unstar notification |
| `r` | Toggle read/unread notification |
| `e` | Archive/unarchive notification |
| `z` | Snooze/unsnooze notification (opens dropdown if not snoozed, unsnoozes if already snoozed) |
| `m` | Toggle mute/unmute notification |
| `t` | Open tag dropdown (or close if already open) |
| `i` | Allow back into inbox (if skipped by rule) |
| `o` | Open in GitHub |
| `h` | Toggle keyboard shortcuts |
| `Escape` | Close dropdowns, clear selection, or exit multiselect |
| `Space` | Toggle notification detail open/closed |

## Multiselect Mode

In multiselect mode, actions are explicit rather than toggles. Use the base key for the primary action and `Shift + key` for the reverse action.

| Shortcut | Action |
|----------|--------|
| `v` | Toggle multiselect mode |
| `x` | Toggle selection of focused item |
| `a` | Cycle select all (page → all → none) |
| `s` | Star selected |
| `Shift + s` | Unstar selected |
| `r` | Mark selected as read |
| `Shift + r` | Mark selected as unread |
| `e` | Archive selected |
| `Shift + e` | Unarchive selected |
| `m` | Mute selected |
| `z` | Open bulk snooze dropdown |
| `t` | Open bulk tag dropdown |
| `i` | Allow selected back into inbox (if skipped by rule) |

**Conditional Behaviors:**
- **`v`**: Cannot activate multiselect mode if detail view is open in list mode (not split view). Use `Shift + v` to open bulk actions palette instead.
- **All action shortcuts** in multiselect mode only work when items are selected (use `x` to select items first)

## Tips

1. **Vim-style Navigation** - Use `j`/`k` for up/down navigation just like in Vim
2. **Quick Jump** - Use `gg` to jump to the top and `Shift + g` to jump to the bottom
3. **Bulk Actions** - Press `v` to enter multiselect mode, select with `x`, then apply actions
4. **View Switching** - Use `Cmd + k` for quick view switching or `Shift + j/k` to cycle through views
5. **Stay in Keyboard** - Most workflows don't require the mouse at all
6. **Single vs. Bulk Actions** - Actions toggle states on single notifications, while multiselect mode uses explicit actions (base key for action, Shift+key for reverse)
7. **Escape Key** - Press Escape multiple times to exit multiselect mode: first clears selection, second exits multiselect

