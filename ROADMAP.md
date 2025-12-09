# Roadmap

This document outlines planned features for Octobud. Items here are under consideration but not yet scheduled.

## Planned Features

### Discussion Comments

Add support for discussion replies in the detail view (like we have for Issues/PRs).

### Better timeline handling

Currently many timeline events are filtered from the Issue/PR timeline. Many of them are just noisy, but they can trigger notifications, so it can be confusing to see a notification returned to the inbox without an obviously new timeline entry.

We could potentially improve this by contextually showing the latest timeline event somewhere to indicate what likely triggered the notification bump.

### Time-Based Filtering

Add support for filtering notifications by time-related criteria, e.g.:

- `updated:>2024-01-01` - Updated after a date
- `created:<7d` - Created within the last 7 days
- `snoozed_until:today` - Snooze expires today

This will enable more powerful queries and views for managing notification age and recency.

### Undoing actions

Investigate support for undoing actions as a quality of life improvement. e.g.:

- Archive a notification mistakenly
- Hit a keyboard shortcut or a button in the Toast message to undo the action

### GitHub Enterprise Support

Investigate support for GitHub Enterprise Server and GitHub Enterprise Cloud deployments, allowing users to:

- Connect to self-hosted GitHub Enterprise instances
- Use Enterprise Cloud with custom domains
- Manage notifications across multiple GitHub environments


---

Have a feature request? [Open an issue](https://github.com/octobud-hq/octobud/issues/new?template=feature_request.md) or start a [discussion](https://github.com/octobud-hq/octobud/discussions).

