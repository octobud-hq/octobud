# Views and Rules

Views and rules help you organize and automate your notification workflow.

## Views

Views are saved queries that create filtered lists of notifications. They appear in your sidebar for quick access.

### Built-in Views

Octobud comes with several default views:

- **Inbox** - All notifications which haven't been archived, muted, snoozed, or routed to skip the inbox by a rule
- **Starred** - Starred notifications
- **Snoozed** - Currently snoozed notifications
- **Archived** - Archived notifications
- **Everything** - Everything, including muted notifications

### Create a View from the Search Bar

The easiest way to create a view:

1. Type a query in the search bar (e.g., `in:inbox type:pullrequest reason:review_requested`)
2. Click the **Save** button in the search bar and then click **Save as new view**
3. Give it a name and **Save view**

That's it! The view is now in your sidebar.

**Example:** Create a view for PRs that need review:
```
in:inbox type:pullrequest reason:review_requested
```

### Creating a View (Detailed Steps)

For more control when creating a view:

1. Click the **Add** button in the sidebar, or click the **Save** button in the query input to start creating a new view based on the current query
2. Enter a name for your view
3. Write a query to filter notifications (see [Query Syntax](query-syntax.md))
   - **Default query:** The query field is pre-populated with `in:inbox,filtered`
   - **Note:** Custom views default to `in:inbox,filtered`, which includes both inbox and filtered notifications. This makes custom views good targets for skip inbox rules, as notifications that skip the inbox will still appear in your view.
4. Choose an icon
5. Click **Save**

### Example Views

**Needs Review**
```
type:pullrequest reason:review_requested
```

**My PRs**
```
type:pullrequest reason:author
```

**Urgent**
```
(reason:mention OR reason:review_requested) AND is:unread
```

**Work Repos**
```
org:my-company -is:archived
```

### Managing Views

- **Reorder** - Drag and drop views in the sidebar
- **Edit** - Right-click a view to edit
- **Delete** - Right-click and select delete

## Rules

Rules automatically apply actions to notifications that match a query. They help automate your workflow.

### How Rules Work

When a new notification arrives, Octobud checks it against all enabled rules in order. If a notification matches a rule's query, the rule's actions are applied.

### Creating a Simple Rule

Start with a basic rule to see how they work:

1. Go to **Settings** → **Rules** → **New Rule**
2. Choose **Query-based**
3. Enter a query (e.g., `author:dependabot`)
4. Select an action (e.g., **Archive**)
5. Click **Create rule**

**Example:** Auto-archive all Dependabot notifications:
- Query: `author:dependabot`
- Action: Archive

### Creating a Rule (Detailed Steps)

1. Go to **Settings** → **Rules**
2. Click **New Rule**
3. Configure the rule:

**Step 1: Choose Rule Type**
- **Query-based** - Use a custom query to match notifications
- **View-linked** - Link to a view's query (automatically updates when the view changes)

**Step 2: Define the Query or View**
- For query-based: Enter a filter query (same syntax as views)
- For view-linked: Select an existing custom view from the dropdown

**Step 3: Configure Rule Details**
- **Rule name** (required) - A descriptive name like "Dependabot PRs"
- **Description** (optional) - What this rule does

**Step 4: Select Actions**
Choose one or more actions to apply to matching notifications:
- Skip inbox
- Mark as read
- Star
- Archive
- Mute

**Step 5: Apply Tags (optional)**
Select tags to automatically apply to matching notifications

**Step 6: Additional Options**
- **Enable rule** - Toggle the rule on/off
- **Apply to existing notifications** - Retroactively apply to all existing notifications (only shown when creating)

4. Click **Create rule**

### Available Actions

| Action | Description |
|--------|-------------|
| **Skip Inbox** | Mark as filtered (won't appear in inbox, but can appear in custom views) |
| **Mark as Read** | Automatically mark as read |
| **Archive** | Move to archive |
| **Star** | Add star |
| **Mute** | Mute the notification (also archives it) |

### Query-based vs View-linked Rules

**Query-based rules** have their own independent query. Use these when you want a rule that doesn't correspond to any view.

**View-linked rules** inherit their query from an existing view. When you update the view's query, the rule automatically uses the new query. This is useful when you want a view and a rule to always stay in sync.

### Example Rules

**Auto-archive Dependabot**
```
Type: Query-based
Query: author:dependabot
Actions: Archive
```

**Skip CI Noise**
```
Type: Query-based
Query: type:CheckSuite
Actions: Skip Inbox, Mark as Read
```

**Tag Security Alerts**
```
Type: Query-based
Query: type:RepositoryVulnerabilityAlert
Actions: (none)
Tags: security
```

**Auto-star Review Requests**
```
Type: Query-based
Query: reason:review_requested
Actions: Star
```

### Rule Order

Rules are processed in order from top to bottom. You can reorder rules by dragging them.

**Tip:** Put more specific rules before general ones.

## Tags

Tags help you categorize and organize notifications.

### Creating Tags

1. Click the **Add** button in the sidebar's Tags section, or right-click an existing tag to edit it
2. Click **Add Tag**
3. Enter a name and choose a color
4. Click **Save**

### Using Tags

- **Manual tagging** - Select notifications and press `t` to add/remove tags
- **Rule-based tagging** - Create rules that automatically add tags
- **Filter by tag** - Use `tags:name` in queries

### Example Tags

- `urgent` - Needs immediate attention
- `waiting` - Waiting for someone else
- `review` - Needs your review
- `followup` - Follow up later

## Best Practices

1. **Start with Views** - Create views for your main workflows before adding rules
2. **Use View-linked Rules** - When you want a view and rule to stay in sync
3. **Use Rules Sparingly** - Too many rules can make behavior unpredictable
4. **Test Queries First** - Test your query in the search bar before using in a rule
5. **Review Filtered** - Periodically enter the `in:filtered` query in the search bar to ensure rules aren't unexpectedly hiding important notifications
6. **Combine Tags and Rules** - Use rules to auto-tag, then create views based on tags

