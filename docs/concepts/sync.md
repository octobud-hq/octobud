# How Syncing Works

Octobud automatically syncs your notifications from GitHub in the background, keeping your inbox up to date.

## Overview

The sync process runs continuously, periodically checking GitHub for new or updated notifications and bringing them into Octobud:

1. **Periodic Checks** - Every 30 seconds (configurable), Octobud checks for new notifications
2. **Fetch New Notifications** - Retrieves any notifications that have been created or updated since the last sync
3. **Process Notifications** - Saves repository information, fetches pull request/issue details, and stores the notifications
4. **Apply Rules** - Automatically applies your rules to new notifications (archive, mute, tag, etc.)
5. **Update Sync State** - Records the latest notification timestamp for efficient incremental syncs

## What Gets Synced

When Octobud syncs a notification, it:

- **Saves Repository Data** - Stores information about the repository (name, organization, etc.)
- **Fetches Subject Details** - Gets pull request or issue information from GitHub (author, state, number, etc.)
- **Stores Notification** - Saves the notification with all its metadata and links to the repository and subject
- **Applies Rules** - Runs new notifications through your rules to apply automatic actions

## Incremental Sync

After initial setup, Octobud uses incremental syncing to be efficient:

- **Subsequent Syncs** - Only fetches notifications that are new or have been updated since the last sync
- **State Tracking** - Octobud tracks the timestamp of the most recent notification to know where to start the next sync

This means syncs are fast and don't repeatedly process the same notifications.

## Processing Flow

### For Each Notification

1. **Repository** - The repository is saved or updated in Octobud's database
2. **Subject Data** - Pull request or issue details are fetched from GitHub
3. **Notification** - The notification is saved with links to the repository and subject
4. **Rules** - If this is a new notification (not an update), your rules are checked and applied

### Parallel Processing

- Multiple notifications can be processed at the same time
- Rules are applied automatically as notifications arrive
- The system handles errors gracefully - if one notification fails, others continue processing

## Sync Frequency

By default, Octobud syncs every 20 seconds. The sync interval balances:

- **Freshness** - Notifications appear quickly
- **Efficiency** - Doesn't overload GitHub's API
- **Performance** - Doesn't slow down the interface

## What to Expect

### First Time Setup

- You'll configure your sync settings (time period, optional limits) before notifications start syncing
- Initial sync duration depends on your chosen time period and notification volume
- A banner shows progress until your first notifications appear

### Ongoing Syncing

- New notifications appear within 30 seconds (or your configured interval)
- Notifications are processed in the background
- Rules are applied automatically as notifications arrive
- You don't need to wait for syncs to complete - you can work with what's already synced

### Errors

If a sync encounters an issue:

- The error is logged but doesn't stop other notifications from syncing
- Optional data (like PR metadata) may be missing if there's a fetch error
- The sync will retry on the next cycle
- Your existing notifications remain available

## Rule Application

Rules are applied automatically when new notifications arrive:

- Rules are checked in the order you've defined them
- Only new notifications trigger rules (updates to existing notifications don't re-trigger rules)
- If a rule fails to apply, it doesn't block other rules or the notification from being saved
- Rules can archive, mute, filter, star, mark as read, or tag notifications

