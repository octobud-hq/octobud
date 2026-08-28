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
- **Svelte components** — ported rather than vendored. They genuinely have to change for
  a ~360px panel, so byte-identity would be a lie. See `src/lib/components/`.

## Longer term

This is the cheap version of a shared package. If a third surface ever needs these
modules, extract `shared/` (or adopt npm workspaces) instead of adding a second copy.
