# Octobud browser extension

Puts one saved Octobud view at a time in the browser's side panel, with a dropdown at
the top to switch between your views. Chrome and Firefox, both MV3.

It is a client for the Octobud desktop app, not a standalone GitHub client: it talks to
the local backend on `localhost`, which is what holds the notification mirror, the saved
views and the GitHub token. Octobud has to be running.

## Quick start

```bash
npm install
npm run stub     # fake backend on :8808, so you don't need Go or a GitHub account
npm run build    # -> dist/chrome and dist/firefox
```

**Chrome** — `chrome://extensions` → enable Developer mode → *Load unpacked* → pick
`dist/chrome`. The toolbar button opens the panel.

**Firefox** — `about:debugging#/runtime/this-firefox` → *Load Temporary Add-on* → pick
`dist/firefox/manifest.json`. Open the sidebar from the toolbar button, or
View → Sidebar → Octobud.

## Scripts

| Script | What it does |
| --- | --- |
| `npm run dev` | Vite dev server for the panel document (no extension APIs — `storage` falls back to memory) |
| `npm run build` | Builds the panel, then assembles `dist/chrome` and `dist/firefox` |
| `npm run stub` | In-memory stand-in for the Octobud API (`dev/stub-server/`) |
| `npm test` | Vitest — vendoring drift, the stub's query evaluator, the octicon subset |
| `npm run check` | `svelte-check` |
| `npm run lint` / `format` / `format:check` | ESLint / Prettier |
| `npm run lint:firefox` | `web-ext lint` against `dist/firefox` |
| `npm run icons` | Regenerates `public/icons/*` from `frontend/static/favicon.png` (needs ImageMagick) |
| `npm run octicons` | Regenerates `src/lib/octicons-subset.ts` |

## How it connects

The backend has no authentication — it trusts localhost — so there is no sign-in. Two
things do need care:

- **Port.** The packaged Octobud binary listens on **8808**; `make backend-dev` uses
  **8080**. On first run the panel probes `/healthz` on each in that order and remembers
  the one that answers, under `octobud:backendUrl` in extension storage.
- **CORS.** The backend allows `http://localhost:*` and `http://127.0.0.1:*` only, and an
  extension origin (`chrome-extension://…`, `moz-extension://…`) does not match. The panel
  relies on `host_permissions` instead, which makes an extension page's cross-origin
  requests exempt from CORS. That is why `host_permissions` is not optional — without it
  every action POST fails preflight.

## Layout

```
manifest/          base.json + per-browser overrides, merged by scripts/build-manifest.mjs
public/            copied verbatim: icons, theme-boot.js, background.js (Chrome only)
dev/stub-server/   fake Octobud API + fixtures (see its README for what it does not do)
src/
  Panel.svelte     the panel shell: header, count row, list, error/empty states
  lib/
    api/           the backend contract, mostly vendored from frontend/
    components/    ported and adopted UI, see VENDORED.md
    panel/         panelController — the action layer, with optimistic updates
    ext/           browser-namespace shim and backend URL discovery
```

`src/lib/VENDORED.md` is the important one to read before changing anything under
`src/lib/api/` or `src/lib/utils/`: those files are byte-identical copies from
`frontend/src/lib`, and `npm test` fails if they drift. It also records which components
were copied, which were adapted, and why.

## Notes

- **Theme** follows the app's `octobud:theme` key and defaults to dark. Extension pages
  get their own origin, so the panel cannot read the app's `localStorage` — you may need
  to set it once in each.
- **No bulk mode, no inline detail.** A row's actions are all there (mark read/unread,
  archive, mute, star, snooze, tag, unfilter); clicking the row opens the item on GitHub.
  Creating and editing views happens in the app — the dropdown's *Add* opens it.
- **`web-ext lint` reports three `UNSAFE_VAR_ASSIGNMENT` warnings** about `innerHTML`.
  They come from Svelte's compiled static fragments and from the row's octicon, whose SVG
  paths are committed constants in `octicons-subset.ts`. No user or API data reaches
  `innerHTML`. There are no errors.
