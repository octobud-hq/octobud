# Octobud API stub

A stand-in for the Octobud Go backend, so the panel can be developed and screenshotted
without a Go toolchain, a built binary, or a GitHub account.

```bash
npm run stub          # http://localhost:8808
PORT=8080 npm run stub # if you want to mimic `make backend-dev`
```

State lives in memory and mutates: archiving something in the panel really does remove it
from the inbox view on the next request. Restart the process to reset.

## What it implements

| Route | Notes |
| --- | --- |
| `GET /healthz` | What the panel probes to discover the backend port |
| `GET /api/views` | Five system views + four seeded custom views, with live unread counts |
| `GET /api/tags` | Five tags |
| `GET /api/notifications?query=&page=&pageSize=` | Paginated, newest first |
| `POST /api/notifications/{githubId}/{action}` | `mark-read`, `mark-unread`, `archive`, `unarchive`, `mute`, `unmute`, `star`, `unstar`, `unfilter`, `snooze`, `unsnooze` |
| `POST /api/notifications/bulk/{assign-tag,remove-tag}` | Body `{ githubIds, tagId }` |

Responses use the real envelopes (`{views}`, `{tags}`, `{notifications,total,page,pageSize}`,
`{notification}`, `{count}`) and notification objects are shaped like
`backend/internal/models.Notification`'s JSON, so the vendored `fromBackendNotification`
mapper runs unmodified.

## What it does not implement

**`query.mjs` is not the query engine.** The real one
(`backend/internal/query/`) is a lexer → parser → AST → SQL builder with parentheses,
explicit `AND`/`OR`/`NOT`, and free-text search across several columns. The stub supports
only what the seeded views need:

- space-separated terms, ANDed
- comma-separated values within a field, ORed
- a leading `-` for negation
- fields `is:`, `in:`, `type:`, `repo:`, `reason:`, `author:`, `tags:`, and the
  `starred:true` / `archived:true` boolean form
- bare words matched against title, repo full name and author

No parentheses, no infix `AND`/`OR`/`NOT` keywords. It reproduces the implicit-default
rules from `query.go` (empty query → inbox, a query containing `in:` → no defaults added,
otherwise → hide muted), because those change which items a view returns.

If you need real query semantics, run the real backend instead of extending this.

Also absent, because the panel never calls them: notification detail, timeline, review
comments, PR commits, view/tag CRUD, sync, OAuth, and everything under `/api/user`.

## CORS

The stub echoes whatever `Origin` it receives. The real backend allows only
`http://localhost:*` and `http://127.0.0.1:*` and relies on the extension's
`host_permissions` to make cross-origin requests exempt from CORS. Echoing here means a
host-permission mistake shows up as a clear failure against the real backend rather than
being masked in development — but it also means **the stub cannot prove the real backend
will accept the panel's requests**. That has to be checked against the Go binary.

## Fixtures

`fixtures.mjs` seeds 16 notifications chosen to exercise every branch `NotificationRow`
renders: each subject type, open/closed/merged/draft states, read and unread, starred,
muted, archived, snoozed, filtered, items with more than the three tags the row shows
inline, and a title long enough to wrap in a narrow panel.
