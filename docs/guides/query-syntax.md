# Query Syntax

Octobud uses a query language to filter notifications. You can use queries in views, rules, and the search bar.

**Note:** All search values use **contains matching** (not exact). For example, `repo:traefik` matches `traefik/traefik` or `traefik/other`.

## Basic Syntax

### Combining Filters (Space-Separated)

Put multiple filters together to narrow your search (logical AND):

```
repo:owner/name is:unread type:PullRequest
```

This finds: notifications from `owner/name` repository **AND** unread **AND** pull requests.

### Multiple Options (Comma-Separated)

Use commas within a field to match any of the options (logical OR):

```
repo:cli,other              # repo:cli OR repo:other
reason:mention,review       # reason:mention OR reason:review
tags:urgent,bug             # tags:urgent OR tags:bug
```

### Free Text Search

Any text without a field prefix searches across title, repository, author, type, state, and PR/Issue number:

```
dependabot          # Matches title, author, etc.
fix bug             # Multiple words (implicit AND)
```

### Negation

Prefix any filter with `-` to negate it:

```
-is:read           # Not read (unread)
-type:Issue        # Not an issue
-repo:owner/name   # Not from this repo
```

## Advanced Querying

### Explicit AND and OR

You can use explicit `AND` and `OR` operators:

```
is:unread AND type:PullRequest
type:PullRequest OR type:Issue
```

### Grouping with Parentheses

Use parentheses to group conditions and control precedence:

```
(type:PullRequest OR type:Issue) is:unread
(reason:mention OR reason:review_requested) AND is:unread
NOT (author:bot OR author:dependabot)
```

**Operator precedence:** Parentheses → NOT → AND → OR

### Complex Queries

Combine all operators for sophisticated filtering:

```
((repo:cli AND is:unread) OR (in:snoozed AND repo:docs)) AND NOT author:bot
```

## Reference

### Status Filters (`is:`)

| Filter | Description |
|--------|-------------|
| `is:read` | Read notifications |
| `is:unread` | Unread notifications |
| `is:starred` | Starred notifications |
| `is:snoozed` | Currently snoozed notifications |
| `is:archived` | Archived notifications |
| `is:muted` | Muted notifications |
| `is:filtered` | Filtered (skipped inbox) notifications |

### Location Filters (`in:`)

| Filter | Description |
|--------|-------------|
| `in:inbox` | In inbox (not archived, snoozed, muted, or filtered) |
| `in:archive` | In archive (not muted) |
| `in:snoozed` | Currently snoozed (not archived or muted) |
| `in:filtered` | Filtered/skipped inbox (notifications that were automatically filtered by rules and skipped the inbox) |
| `in:anywhere` | All notifications (no location filter) |

**Note:** Use `in:inbox,filtered` (equivalent to `in:inbox OR in:filtered`) to include notifications that have skipped the inbox due to rule automation.

### Type Filters (`type:`)

| Filter | Description |
|--------|-------------|
| `type:PullRequest` | Pull request notifications |
| `type:Issue` | Issue notifications |
| `type:Release` | Release notifications |
| `type:Discussion` | Discussion notifications |
| `type:Commit` | Commit notifications |
| `type:CheckSuite` | CI/CD check suite notifications |
| `type:RepositoryVulnerabilityAlert` | Security alert notifications |

### Reason Filters (`reason:`)

| Filter | Description |
|--------|-------------|
| `reason:review_requested` | Review was requested |
| `reason:mention` | You were mentioned |
| `reason:author` | You authored the item |
| `reason:comment` | Someone commented |
| `reason:assign` | You were assigned |
| `reason:team_mention` | Your team was mentioned |
| `reason:subscribed` | You're subscribed |
| `reason:state_change` | State changed |
| `reason:ci_activity` | CI activity |

### Repository and Author Filters

| Filter | Description |
|--------|-------------|
| `repo:owner/name` | Match repository (contains matching) |
| `org:owner` | All repos in an organization (contains matching) |
| `author:username` | Filter by author (contains matching) |

### State Filters

| Filter | Description |
|--------|-------------|
| `state:open` | Open issues/PRs |
| `state:closed` | Closed issues/PRs |
| `merged:true` | Merged pull requests |
| `merged:false` | Unmerged pull requests |
| `state_reason:completed` | Issues closed as completed |
| `state_reason:not_planned` | Issues closed as not planned |

### Tag Filters

| Filter | Description |
|--------|-------------|
| `tags:tag-name` | Has tag matching pattern (contains matching) |

### Boolean Filters

Use with `true`/`false`, `yes`/`no`, or `1`/`0`: `read:true`, `archived:true`, `muted:true`, `snoozed:true`, `filtered:true`

## Example Queries

### Unread PR reviews

```
is:unread type:PullRequest reason:review_requested
```

### My mentions across all repos

```
reason:mention in:anywhere
```

### Issues in a specific org

```
org:my-company type:Issue is:unread
```

### Everything except CI notifications

```
-reason:ci_activity
```

### Starred but not archived

```
is:starred -is:archived
```

### Open PRs that need review

```
type:PullRequest reason:review_requested state:open -is:archived
```

### Merged PRs

```
type:PullRequest merged:true
```

### Dependabot PRs to archive

```
author:dependabot type:PullRequest
```

### Exclude all bot authors

```
-author:[bot]
```

**Note:** GitHub bot usernames end with `[bot]` (e.g., `dependabot[bot]`, `renovate[bot]`). Using `[bot]` matches all bots without catching human users with "bot" in their name.

### Notifications with a specific tag

```
tags:urgent
```

### Filtered notifications (skipped inbox)

```
in:filtered
in:filtered type:PullRequest
in:anywhere is:filtered  # Alternative syntax
```

Useful for reviewing what rules have filtered and ensuring important notifications aren't being hidden.

### Complex queries

```
(reason:mention OR reason:review_requested) AND is:unread
NOT (author:bot OR author:dependabot)
((repo:cli AND is:unread) OR (in:snoozed AND repo:docs)) AND NOT author:bot
```

## Tips

1. **Start Simple** - Begin with one or two filters and add more as needed
2. **Use Views** - Save complex queries as views for quick access
3. **Try Negation** - Sometimes it's easier to exclude what you don't want
4. **Combine with Rules** - Use queries in rules to auto-organize notifications
5. **Review Filtered** - Periodically check `in:filtered` to ensure rules aren't hiding important notifications
6. **Contains Matching** - All search values use contains matching, not exact matching

