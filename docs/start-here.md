# Start Here

This guide covers your first 5 minutes with Octobud: configuring your initial sync and learning the core workflows.

## Initial Setup

When you first launch Octobud, you'll be guided through two steps:

1. **Connect GitHub** - Connect your GitHub account. OAuth is the preferred method (recommended), or you can use a Personal Access Token as an alternative. Both require `notifications` and `repo` scopes.
2. **Configure Initial Sync** - Set how far back Octobud should sync your existing GitHub notifications.

## Initial Sync

After connecting GitHub, you'll be prompted to configure how far back Octobud should sync your existing GitHub notifications.

### Recommended Approach

**Start small.** We recommend 30 days or less, optionally with a hard limit on notification count. You can always sync more later from Settings.

| Option | Best For |
|--------|----------|
| **30 days** | Most users. Quick start with recent, relevant notifications. |
| **60-90 days** | Want a bit more history without going overboard. |
| **Custom** | Specific needs (e.g., 2 weeks or 6 months). |
| **All time** | Power users who want everything. May take a while. |

**Note**: GitHub rate limits may slow down syncing if importing a large amount.

### Advanced Options

- **Maximum Notifications** - Cap how many to sync. Useful with "All time" or long periods.
- **Unread Only** - Only sync notifications marked unread on GitHub. Reduces volume but may miss notifications you've seen but not addressed.

### Syncing More Later

From **Settings → Data → Sync additional history**, you can sync older notifications beyond your initial import. This syncs notifications *before* your oldest synced notification, so you won't re-sync what you already have.

## Core Workflow

Once your notifications are synced, here's the typical triage workflow:

1. Review your **Inbox** for unread notifications
2. **Star** (`s`) important items for follow-up
3. **Archive** (`e`) items you've handled
4. **Snooze** (`z`) items to revisit later
5. **Tag** (`t`) items to categorize (e.g., "followup", "review")

### Available Actions

| Action | Shortcut | Description |
|--------|----------|-------------|
| **Star** | `s` | Mark as important - appears in "Starred" view |
| **Mark Read/Unread** | `r` | Toggle read status |
| **Archive** | `e` | Move out of inbox |
| **Snooze** | `z` | Hide until a specific date/time |
| **Mute** | `m` | Permanently hide - only visible in "Everything" view |
| **Tag** | `t` | Add custom tags for organization |

### Notification States

- **Inbox** - Active notifications (not archived, muted, snoozed, or filtered)
- **Archived** - Removed from inbox but still searchable
- **Snoozed** - Temporarily hidden, automatically returns to inbox
- **Muted** - Permanently hidden (only visible in "Everything" view)
- **Filtered** - Skipped inbox due to a rule, but still accessible in custom views

### Bulk Actions

Click the **multiselect button** in the toolbar (or press `v`) to enter multiselect mode:

1. Click checkboxes to select notifications
2. Use **Select All** to select the current page or all pages
3. Click action buttons in the toolbar to apply to all selected

Keyboard users can use `x` to toggle selection, `a` to cycle through select-all options, and the usual action shortcuts (`e`, `s`, `z`, `t`).

## Quick Reference

### Essential Shortcuts

| Shortcut | Action |
|----------|--------|
| `j` / `k` | Navigate list |
| `h` | Show all shortcuts |
| `Cmd + k` | Command palette |
| `v` | Toggle multiselect mode |

### Basic Query Syntax

Filter notifications using queries in the search bar:

```
is:unread type:PullRequest           # Unread PRs
repo:owner/name reason:mention       # Mentions in a specific repo
author:dependabot                    # All Dependabot notifications
```

## Next Steps

- **[Query Syntax Guide](guides/query-syntax.md)** - Master the query language
- **[Views and Rules](guides/views-and-rules.md)** - Create saved views and automation rules
- **[Keyboard Shortcuts](guides/keyboard-shortcuts.md)** - Full shortcut reference

