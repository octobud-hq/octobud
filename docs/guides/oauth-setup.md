# OAuth Setup Guide

This guide walks you through connecting Octobud to your GitHub account using OAuth, the recommended authentication method.

## Overview

OAuth is a convenient way to authenticate with GitHub in Octobud. It uses GitHub's Device Flow, which allows you to authorize Octobud without sharing your password. OAuth provides automatic token management and easier access control.

**Note**: If your organization disables OAuth apps or you have multiple GitHub orgs, consider using a [Personal Access Token](./personal-access-token-setup.md) instead, as it may be the only viable option in those cases.

## Required Permissions

Octobud requests the following OAuth scopes:

- `notifications` - Read access to notifications
- `repo` - Read access to repositories (includes discussions)

These permissions allow Octobud to:
- Sync your GitHub notifications
- Access repository details for PRs, issues, and discussions
- Display full context for your notifications

**Note**: The `repo` scope includes access to discussions, so no separate `read:discussions` scope is needed for OAuth.

## Step 1: Start OAuth Flow in Octobud

1. Open Octobud and go to **Settings → Account**
2. Under "GitHub Authentication", select **"OAuth"**
3. Click **"Connect with GitHub"**
4. A new browser tab will automatically open to GitHub's device authorization page

## Step 2: Get Your User Code

1. **Switch back to the Octobud tab** - The user code will be displayed in Octobud
2. Copy or note down the user code shown in Octobud

## Step 3: Authorize on GitHub

1. **Switch to the GitHub tab** that was automatically opened (or go to [github.com/login/device](https://github.com/login/device))
2. Enter the user code from Octobud
3. Click **"Continue"**
4. Review the permissions Octobud is requesting
5. Click **"Authorize Octobud"**

Octobud will automatically detect when you've completed authorization and connect your account.

## Step 4: Organization Approval (Required for Organization Access)

**Important**: If you're a member of GitHub organizations, organization administrators must approve Octobud before you can access notifications and repositories from those organizations.

### Why Organization Approval is Required

GitHub organizations can require administrator approval for third-party OAuth applications. This is a security feature that gives organizations control over which applications can access their data.

### How to Request Organization Approval

After you authorize Octobud:

1. **Check for approval prompts**: When you authorize Octobud, GitHub may show you a list of organizations that require approval
2. **Request access**: Click **"Request"** or **"Grant access"** for each organization you want Octobud to access
3. **Wait for administrator approval**: Organization administrators will receive a notification and must approve the request
4. **Check approval status**: You can check the status at [GitHub Settings → Applications → Authorized OAuth Apps](https://github.com/settings/applications)

### What Happens Without Organization Approval

If an organization hasn't approved Octobud:

- ✅ **Personal repositories** - You'll still see notifications from your personal repositories
- ❌ **Organization repositories** - You won't see notifications from organization repositories until approval is granted
- ❌ **Organization notifications** - Notifications from organization repositories won't sync

### For Organization Administrators

If you're an organization administrator and a team member requests access to Octobud:

1. Go to your organization's **Settings → Third-party access**
2. Find the pending request for **Octobud**
3. Review the requested permissions
4. Click **"Approve"** to grant access

You can also manage approved applications at any time from the organization settings.

## Troubleshooting

### Authorization Not Completing

If Octobud doesn't detect your authorization:

- **Check the user code**: Make sure you copied the user code from the Octobud tab and entered it correctly on GitHub
- **Check expiration**: User codes expire after a few minutes. If it expired, start the flow again
- **Check both tabs**: Make sure you've completed authorization in the GitHub tab and that Octobud is still open in the other tab
- **Refresh Octobud**: Try refreshing the Octobud interface or restarting the application
- **Check network**: Ensure Octobud can reach GitHub's API

### Organization Notifications Not Appearing

If you've authorized Octobud but don't see organization notifications:

1. **Check approval status**: Go to [GitHub Settings → Applications → Authorized OAuth Apps](https://github.com/settings/applications) and verify Octobud is approved for your organizations
2. **Request approval**: If not approved, request access from your organization administrators
3. **Wait for approval**: Organization administrators must explicitly approve the application
4. **Re-authorize if needed**: Sometimes you may need to disconnect and reconnect after organization approval

### "Access Denied" Error

If you see an "Access Denied" error:

- You may have clicked "Cancel" during the authorization process
- Start the OAuth flow again and make sure to click "Authorize" instead of "Cancel"
- If the issue persists, try disconnecting and reconnecting your account

### Token Expired or Revoked

If your OAuth token stops working:

1. Go to **Settings → Account** in Octobud
2. Disconnect your GitHub account
3. Reconnect using the OAuth flow again
4. Re-request organization approvals if needed

## OAuth vs Personal Access Token

### Advantages of OAuth

- ✅ **More secure** - No need to manually create and manage tokens
- ✅ **Easier management** - Revoke access directly from GitHub settings
- ✅ **Better organization support** - Organization administrators can centrally manage access
- ✅ **Automatic token refresh** - GitHub handles token lifecycle management
- ✅ **Audit trail** - Organizations can see which applications have access

### When to Use Personal Access Token

Consider using a Personal Access Token if:
- **Your organization disables OAuth apps** - Some organizations have policies that block OAuth applications entirely
- **You have multiple GitHub orgs** - PATs can be more reliable when working across multiple organizations with different OAuth policies
- You need fine-grained control over token permissions
- You prefer managing tokens manually

See the [Personal Access Token Setup Guide](./personal-access-token-setup.md) for PAT instructions.

## Security Best Practices

- **Use OAuth when possible** - OAuth is the recommended and more secure method
- **Review permissions** - Only authorize applications you trust
- **Revoke unused access** - If you stop using Octobud, revoke access from [GitHub Settings → Applications → Authorized OAuth Apps](https://github.com/settings/applications)
- **Monitor organization access** - Organization administrators should regularly review approved applications
- **Report suspicious activity** - If you notice unauthorized access, revoke immediately and report to GitHub

## Related

- [Personal Access Token Setup Guide](./personal-access-token-setup.md) - Alternative authentication method
- [Installation Guide](../installation.md) - General installation and configuration
- [Start Here](../start-here.md) - Initial setup and core workflows

