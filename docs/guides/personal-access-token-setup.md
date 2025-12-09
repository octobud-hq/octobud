# Personal Access Token Setup Guide

This guide walks you through creating and configuring a GitHub Personal Access Token (PAT) for use with Octobud.

## Overview

While OAuth is the recommended authentication method, you can use a Personal Access Token as an alternative. This guide covers the complete setup process, including handling organization SSO requirements.

## Required Permissions

Your Personal Access Token needs the following scopes:

- `repo` - Read access to repositories
- `notifications` - Read access to notifications
- `read:discussions` - Read access to discussions

## Step 1: Create a Classic Token

1. Go to [GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)](https://github.com/settings/tokens)
2. Click **"Generate new token"** → **"Generate new token (classic)"**
3. Give your token a descriptive name (e.g., "Octobud")
4. Set an expiration date (or leave as "No expiration" if you prefer)
5. Select the required scopes:
   - ✅ **repo** (Full control of private repositories)
   - ✅ **notifications** (Access notifications)
   - ✅ **read:discussions** (Read access to discussions)
6. Click **"Generate token"**
7. **Important**: Copy the token immediately - you won't be able to see it again!

## Step 2: Authorize for Organizations (SSO)

If you're a member of GitHub organizations that use Single Sign-On (SSO), you'll need to authorize your token for each organization:

1. After creating your token, you may see a notice that the token needs to be authorized for organizations, or a **Configure SSO** button in the list of tokens.
2. Click **"Enable SSO"** or **"Authorize"** next to each organization that requires it
3. Complete the SSO authorization flow for each organization
4. You can also manage this later at [GitHub Settings → Personal access tokens](https://github.com/settings/tokens)

### Why SSO Authorization Matters

Organizations with SSO enabled require explicit authorization for Personal Access Tokens. Without this authorization, Octobud won't be able to access notifications or repository data from those organizations.

## Step 3: Add Token to Octobud

1. Open Octobud and go to **Settings → Account**
2. Under "GitHub Authentication", select **"Personal Access Token"**
3. Paste your token into the token field
4. Click **"Save"**

Octobud will validate the token and connect your account.

## Known Limitations

### Organization Restrictions on Repository Access

Some GitHub organizations have policies that allow Personal Access Tokens to access notifications but restrict access to repository details (PRs, issues, etc.). In these cases:

- ✅ **Notifications will sync** - You'll see notifications from these organizations
- ❌ **PR/Issue details won't load** - When you click on a notification, the PR or issue details may not load because the token doesn't have sufficient repository access

This is a limitation imposed by the organization's security policies, not by Octobud. If you encounter this issue:

1. Check with your organization administrators about PAT policies
2. Consider using OAuth instead, which may have different permission requirements
3. Some organizations may allow repository access through OAuth but restrict it for PATs

### Workaround

If you need full functionality for organization repositories, OAuth is often the better choice as it may have different permission handling that bypasses these restrictions. However, the Organization administrator may still need to explicitly give Octobud permission to access the Org's data.

## Troubleshooting

### Token Not Working

- **Verify scopes**: Ensure all three required scopes are selected
- **Check expiration**: If you set an expiration date, verify the token hasn't expired
- **SSO authorization**: Make sure you've authorized the token for all required organizations
- **Organization policies**: Some organizations may block PAT access entirely

### PR/Issue Details Not Loading

If notifications appear but PR/issue details don't load:

1. This is likely due to organization restrictions on repository access (see "Known Limitations" above)
2. Try using OAuth instead of a PAT
3. Contact your organization administrators about PAT access policies

### Token Revoked or Expired

If your token stops working:

1. Check if it was revoked in [GitHub Settings → Personal access tokens](https://github.com/settings/tokens)
2. Create a new token following the steps above
3. Update the token in Octobud (Settings → Account)

## Security Best Practices

- **Use OAuth when possible** - OAuth is more secure and easier to manage
- **Set expiration dates** - Regularly rotate your tokens for better security
- **Limit scope** - Only grant the minimum required permissions
- **Store securely** - On macOS, tokens are stored in Keychain. On other platforms, they're encrypted but the key is stored locally
- **Revoke unused tokens** - Delete tokens you're no longer using

## Related

- [Installation Guide](../installation.md) - General installation and configuration
- [Start Here](../start-here.md) - Initial setup and core workflows

