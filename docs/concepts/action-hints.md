# Action Hints

Action hints help Octobud predict whether a notification will disappear from your current view when you perform an action on it. This enables the interface to update instantly without waiting for a refresh.

## What Are Action Hints?

When you view a list of notifications, Octobud calculates which actions would cause each notification to disappear from that view. For example:

- In your **Inbox** view, archiving or muting a notification will remove it (since the inbox excludes archived and muted items)
- In an **Archived** view, unarchiving a notification will remove it (since it's no longer archived)

## How It Works

Octobud tests each action against your current query to see if the notification would still match. If the action would cause the notification to no longer match your current filter, it's marked as a "dismissive" action.

## Supported Actions

The following actions are tested to see if they would dismiss a notification from the current view:

- **Archive/Unarchive** - Moving items in or out of the archive
- **Mute/Unmute** - Hiding or showing muted notifications
- **Snooze/Unsnooze** - Temporarily hiding notifications
- **Filter/Unfilter** - Skipping or allowing items back into the inbox

## Actions That Never Dismiss

Some actions never cause notifications to disappear from a view:

- **Read/Unread** - Marking as read or unread doesn't remove notifications (similar to Gmail)
- **Star/Unstar** - Starring doesn't affect visibility in views

These actions are always visible and won't cause a notification to disappear unless you refresh the view.

## Why This Matters

This system ensures that:

- The interface updates immediately when you take actions
- You can see right away which actions will remove a notification from your current view
- The behavior is consistent with your query filters

