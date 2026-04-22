# Changelog

All notable changes to Octobud will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.2]

### Added

- **New query filters**: `label:`, `assignee:`, `reviewer:`, `team_reviewer:`, `draft:`, `merged:`, `state_reason:` for filtering notifications by PR/issue metadata
- **Assignee and reviewer display**: Detail view shows avatar stacks for assignees and reviewers with hover popovers listing logins and review statuses
- **Reviewer status tracking**: PR reviewer statuses (approved, changes requested, commented, dismissed, pending) are fetched from the GitHub Reviews API and displayed as colored status dots on reviewer avatars
- **GitHub label chips**: Labels from PRs/issues are displayed as colored chips in the detail view metadata row, matching their GitHub colors
- **Enrichment pipeline**: Background enrichment system with versioned processing for re-enriching existing notifications when new data sources are added
- **Query input auto-space**: Focusing the query input appends a trailing space so you can immediately start typing a new filter token and see the base suggestion list
- **Token expiry checks/warnings**: Show warning banner when a token is expring in the next 7 days, and an error banner if it is invalid or expired.

### Changed

- **BREAKING**: `Space` now scrolls the detail view (with `Shift+Space` to scroll up) when the detail is open, instead of closing it. Open the detail with `Enter` or `Space` (while closed); close it with `Escape`. Previously `Space` toggled the detail open/closed.
- Timeline "Changes requested" review badge now uses a diff icon instead of a warning emoji
- GitHub label chip text contrast improved in both light and dark mode using `color-mix`

### Fixed

- `merged:` and `state_reason:` query fields were handled in the SQL builder but rejected by the validator, so they never worked

## [0.3.0]

### Added

Major timeline updates: 

- **PR review comments**: Inline review comments are fetched and displayed alongside their corresponding review event in the timeline
- **Discussion comment replies**: Discussion timelines now show threaded replies beneath each comment
- **Cross-reference sources**: Cross-referenced events now display the linked issue or PR with title, number, and state
- **Event grouping**: Consecutive events of the same type (labels, assignments, review requests) are grouped into a single compact row
- **Live timeline refresh**: When viewing a notification's timeline, new activity from polling is automatically appended without requiring a page refresh
- **"New activity" indicator**: A divider line and floating button appear when unseen timeline items arrive, with smooth scroll-to and auto-dismiss on scroll
- **Timeline last-seen tracking**: Tracks the last-seen timeline timestamp per notification so the "New activity" indicator persists across sessions
- **Auto-mark-read on new activity**: Notifications marked unread by incoming activity are automatically marked read when the user views the new items
- **Auto-scroll to new timeline activity**: Optional setting to automatically scroll to unseen timeline activity when opening the detail view (disabled by default)

Rule improvements:

- **Duplicate rule**: Duplicate an existing rule from the expanded rule card to quickly create variations with pre-filled settings
- **Re-apply rule to existing notifications**: The "Apply to existing notifications" toggle is now available when editing rules, not just when creating them

### Changed

- Rule, tag, and view dialogs no longer close when clicking outside, preventing accidental loss of in-progress edits
- Fixed and improved backend timeline fetching
- Discussion timelines now report `hasMore` when there are more than 100 comments
- Service worker `NEW_NOTIFICATIONS` messages now include `githubIds` so the frontend can target refreshes to the currently-open notification
- Timeline deduplication uses composite keys (`type-timestamp` fallback) to handle events with null IDs (e.g., commits, merges)
- Multiselect is redesigned to be much easier to use:
  - No jarring layout movements - bulk action bar replaces the query input
  - Enter multiselect just by clicking the select area for a notification
  - Much bigger/easier click targets for checkboxes

## [0.2.1]

### Added
### Changed

- Less aggressive retrying when syncing, now respecting rate limit headers if present.

### Deprecated
### Removed
### Fixed

## [0.2.0]

### Added

- **Undo System**: Full undo support for notification actions
  - Undo individual actions: archive, star, mute, snooze, read/unread, tag/untag
  - Undo bulk actions performed on selected notifications (by ID)
  - Toast notifications with inline undo button and `Cmd+Z` / `Ctrl+Z` keyboard shortcut
  - Recent actions history dropdown (`Shift+H`) showing up to 20 undoable actions
  - Actions persist across page refreshes via local storage
  - Intelligent inverse cancellation: rapid toggling (e.g., star then unstar) cancels out in history
- **History Dropdown Keyboard Navigation**: Full vim-style navigation when history is open
  - `J` / `K` to navigate between items
  - `Enter` to undo the focused action
  - `Space` to open the notification in detail view
  - `Escape` or `H` to close
  - All other shortcuts blocked while history is open (modal focus trap)
- **Unread badge on favicon (optional)** - show unread count in a badge on favicon. Off by default but can be enabled in settings -> notifications.
- **Timeline improvements** - some filtered timeline items have been reintroduced with richer context (review request, mention, label)

### Changed

- Bulk actions on "select all" (query-based) now always show a confirmation dialog with a warning that the action cannot be undone
- Toast notifications moved to bottom-right corner of the screen
- Improved detail/icon/title handling for CI and release notifications

### Deprecated
### Removed
### Fixed

## [0.1.6]

### Added

- Add progressive page size shrinking when syncing notifications to address an issue where
some users randomly can't fetch more than 10-15 notifications without getting 502/504 errors.

### Changed
### Deprecated
### Removed
### Fixed

## [0.1.5]

### Added

### Changed

- Cleaned up CI to remove redundant lint step, and unify with local dev flow.

### Deprecated
### Removed
### Fixed

- Fix restart handling after new version install.

## [0.1.4]

### Added

- Added "title:" support in query syntax to filter against the subject title
- Add new version installation detection and refresh prompt + restart.
- Re-apply rules if subject is refreshed and wasn't present before.

### Changed

- Make PAT auth more prominent in onboarding/settings.

### Deprecated
### Removed
### Fixed

- Fix log filing.
- Fix unread count in tray.
- Fix bug that would periodically allow notifications to be synced without processing/rule application.

### Security

## [0.1.3] - 2025-12-09

- Small big fixes around subject urls
- Fix discussion timeline fetching (working now!)

## [0.1.2] - 2025-12-09

- Fix update check version resolution when doing manual checks
- Improve avatar fetching
- Add a first attempt at Discussion timeline fetching

## [0.1.1] - 2025-12-09

- Bug fixes and in-app doc improvements around OAuth/PAT
- Improved handling of certain action-related events
- In-app handling of situations where a PAT gives access to notifications but can't access repos (due to org policy or weird SSO situations)
- Remove VERSION file and checks on it
- Always show current version in the frontend

## [0.1.0] - 2025-12-08

### Added

- Initial release of Octobud
- Gmail-inspired notification inbox interface
- Split pane mode for viewing notifications and details side-by-side
- Full notification lifecycle management: Star, Snooze, Archive, Tag, and Mute
- Inline Issue and PR details with status, comments, and timeline
- Custom Views with rich query language for filtering notifications
- Keyboard-first navigation with comprehensive keyboard shortcuts
- Automation Rules to automatically archive, filter, or tag notifications based on criteria
- Custom Tags with colors for organizing notifications
- Real-time background sync to keep notifications up to date
- Desktop notifications for review requests and issue replies
- Local-first architecture with SQLite storage for fast performance
- macOS menu bar integration with unread count and quick actions
- macOS auto-start capability
- Secure GitHub token storage via macOS Keychain
- GitHub OAuth device flow authentication
- Personal Access Token authentication option
- Background job scheduler for async operations
- Privacy-first design - all data stored locally on your machine

### Platform Support

- **macOS**: Full support with menu bar integration, auto-start, and Keychain storage
- **Linux & Windows**: Core functionality available, with encrypted token storage (Keychain support planned)

