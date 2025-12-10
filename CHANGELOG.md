# Changelog

All notable changes to Octobud will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Added "title:" support in query syntax to filter against the subject title

### Changed
### Deprecated
### Removed
### Fixed
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

