# Vendored modules

The files listed below are **byte-identical copies** of `frontend/src/lib/<same path>`.
They are the API contract with the Go backend plus the pure formatting helpers the
notification row needs — the code most likely to drift and silently break the panel.

`test/vendored.test.ts` asserts the copies still match their originals, so drift fails
`npm test` instead of rotting quietly.

## Copied verbatim — do not hand-edit

| Path (identical under `frontend/src/lib/` and `extension/src/lib/`) |
| --- |
| `api/types.ts` |
| `api/views.ts` |
| `api/notifications.ts` |
| `api/tags.ts` |
| `utils/time.ts` |
| `utils/notificationHelpers.ts` |
| `utils/notificationIcons.ts` |
| `utils/viewIcons.ts` |
| `utils/githubUrls.ts` |
| `utils/snoozeFormat.ts` |
| `utils/archiveIcons.ts` |
| `utils/muteIcons.ts` |
| `components/shared/SnoozeDropdown.svelte` |
| `components/shared/CompactPagination.svelte` |

Those two components needed no adaptation at all — they depend only on `api/types` and
Svelte, and are already sized for narrow layouts.

A byte-identity failure on a component is a real signal, not noise: if the desktop row or
dropdown changes, the panel should be re-synced or the divergence justified, because
matching the app's design language is the point of the feature.

They live at the same relative paths on purpose. With `$lib` aliased to
`extension/src/lib`, every internal import (`./time`, `$lib/utils/viewIcons`) resolves
without edits.

**To update:** re-copy from `frontend/`, then run `npm test`. If a change is needed for
the extension specifically, make it in `frontend/` and re-copy, or promote the file out
of this list with a note explaining why.

## Deliberately not vendored

- **`api/fetch.ts`** — same exported surface, different base-URL resolution. The desktop
  app is served from the backend's own origin and bakes `VITE_API_BASE_URL` in at build
  time; the panel is an extension origin talking to localhost on a port only known at
  runtime. Excluded from the byte-identity test.
- **`state/types.ts`** — imports SvelteKit-generated `routes/views/[slug]/$types`, so it
  cannot be copied. Its `BUILT_IN_VIEWS` constants were not lifted either: the backend
  already returns system views from `GET /api/views` flagged `systemView: true`, so the
  panel partitions the API response instead of keeping a second local copy of the list.
- **`components/timeline/ViewDropdown.svelte`** — **adopted**, not vendored. Nothing in the
  desktop app imports it; it is dead code there, and the panel is now its only consumer.
  Holding it byte-identical to an unused file would have meant keeping its two stale bits:
  `starred` had no icon (an empty, misaligned span for a view the backend does return) and
  `done` mapped an icon for a built-in slug that no longer exists. Both are fixed here.
- **`components/shared/TagDropdown.svelte`** — **adopted**, not vendored. Its positioning
  is only correct when the viewport is comfortably wider than the dropdown. The tag button
  sits inside the row's action bar, so right-anchoring the 280px-min panel put its left
  edge near −39px in a 360px sidebar, cutting off the search field and the tag names. The
  width is now clamped to the viewport and positioning resolves to a single clamped `left`
  rather than discarding the clamp on the right-anchor branch. Both changes are correct on
  desktop too and are worth upstreaming.

  `SnoozeDropdown.svelte` shares the same positioning logic and the same latent bug, but
  it is only 220px wide and its button is the last in the bar, so it lands on-screen at
  panel widths and is left vendored. If a narrower sidebar ever breaks it, adopt it the
  same way.
- **`components/notification_view/NotificationRow.svelte`** — ported, not copied. It is
  the one component that genuinely has to change for a ~360px panel. The four
  differences are listed in a comment at the top of the file: smaller action buttons, a
  plain status gutter instead of the multiselect entry point, click-opens-GitHub instead
  of an inline detail pane, and no keyboard focus index.
- **`components/shared/ApiErrorState.svelte`** — not reused. It is a `max-w-2xl`, `p-8`,
  `text-2xl` card telling you to check the API server, worker and database; none of that
  fits a side panel or describes the panel's actual failure mode. `components/panel/
  PanelMessage.svelte` covers the same ground at panel scale.
- **`@primer/octicons`** — not vendored, but aliased. `utils/notificationIcons.ts` imports
  the whole 4 MB registry for twelve 16px paths, which cost about a megabyte of bundled
  JS. Vite redirects the import to the generated `octicons-subset.ts`
  (`npm run octicons`), leaving the vendored file untouched.

## Longer term

This is the cheap version of a shared package. If a third surface ever needs these
modules, extract `shared/` (or adopt npm workspaces) instead of adding a second copy.
