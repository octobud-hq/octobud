# Installation & Configuration

This guide covers installing and configuring Octobud on macOS. For Linux and Windows, build from source (see [Quick Start](../README.md#quick-start)) - core functionality works, but menu bar integration and auto-start are not available.

## macOS Installation

### Option 1: Installer (Recommended)

1. Download the latest `.pkg` installer from the [Releases page](https://github.com/octobud-hq/octobud/releases)
2. Open the `.pkg` file and follow the installation wizard
3. Octobud will be installed to `/Applications/Octobud.app`
4. Launch Octobud from Applications or Spotlight

The installer automatically:
- Installs `Octobud.app` to `/Applications`
- Adds Octobud to Login Items so it launches automatically on login

### Option 2: Build from Source

```bash
git clone https://github.com/octobud-hq/octobud.git && cd octobud
make build
./bin/octobud
```

## First Launch

When you first launch Octobud:

1. **Menu Bar Icon** - Octobud appears in your menu bar (top right on macOS)
2. **Browser Opens** - Your default browser opens to `http://localhost:8808`
3. **Connect GitHub** - You'll be prompted to connect your GitHub account. OAuth is the preferred method (recommended), or you can use a Personal Access Token as an alternative. Both require `notifications` and `repo` scopes.
4. **Configure Initial Sync** - After connecting GitHub, you'll be prompted to configure how far back to sync your existing notifications. We recommend starting with 30 days - you can always sync more later from Settings → Data.

## Menu Bar

On macOS, Octobud runs as a menu bar app. Click the menu bar icon to access:

- **Status** - Shows current status (running, syncing, last sync time)
- **Inbox** - Opens the Inbox view in your browser
- **Starred** - Opens the Starred notifications view
- **Snoozed** - Opens the Snoozed notifications view
- **Mute desktop notifications** - Submenu to temporarily mute notifications:
  - 30 minutes
  - 1 hour
  - Rest of day
  - Unmute
- **Settings** - Opens Octobud settings in your browser
- **Quit** - Exit the application

## Configuration

### Command Line Options

When running from the command line:

```bash
./bin/octobud --help
  -port int        Port to listen on (default 8808)
  -data-dir string Data directory (default: platform-specific)
  -no-open         Don't open browser on startup
  -version         Show version and exit
```

### Data Directory

Octobud stores all data locally on macOS:

| Platform | Location |
|----------|----------|
| macOS | `~/Library/Application Support/Octobud` |

The data directory contains:
- `octobud.db` - SQLite database with all your notifications and settings
- `.token_key` - Auto-generated encryption key for GitHub token storage (non-macOS only)

### GitHub Authentication

Octobud supports two methods for authenticating with GitHub:

1. **OAuth (Recommended)** - Connect your GitHub account directly through GitHub's OAuth flow. This is the preferred method as it's more secure and easier to set up.
2. **Personal Access Token (Alternative)** - Use a GitHub Personal Access Token if you prefer not to use OAuth.

Both methods require `notifications` and `repo` scopes.

#### During Initial Setup

When you first launch Octobud, you'll be prompted to connect your GitHub account. You can choose either OAuth or enter a Personal Access Token.

#### Updating Authentication

You can change your GitHub authentication method at any time:

1. Go to Settings → Account
2. For OAuth: Click "Connect with GitHub" to start the OAuth flow
3. For Personal Access Token: Paste your token (needs `notifications` and `repo` scopes) and click Save

#### Token Storage

**On macOS:** Your GitHub token is securely stored in the macOS Keychain using the service name `io.octobud` and account name `github_<username>`. This provides better security by leveraging the system keychain's built-in encryption and access controls.

**On Linux & Windows:** The token is encrypted and stored in the SQLite database using the `.token_key` file in the data directory. While the token is encrypted at rest, the encryption key is stored alongside the database, so an attacker with access to your machine could decrypt and access your credentials. Keychain support for these platforms is planned for a future release.

### Accessing Octobud

After installation, you can access Octobud through the Octobud menu bar item or in your browser at:

- `http://localhost:8808` - Default access URL

The app runs locally on your machine - it's not accessible from other devices on your network by default.

Octobud is configured to start automatically on login. You can manage this in **System Settings > General > Login Items**.

## Operations

### Backups

The SQLite database is a single file. To backup:

```bash
# On macOS
cp ~/Library/Application\ Support/Octobud/octobud.db ~/backup/octobud-$(date +%Y%m%d).db
```

For automated backups, you can use Time Machine (macOS) or set up a cron job.

### Logs

The application logs to stdout/stderr. When running as a regular app, check Console.app:

```bash
# View logs in Console.app
open -a Console
# Filter for "octobud"
```

When running as a service, use:

```bash
log show --predicate 'process == "octobud"' --last 1h
```

## Troubleshooting

### App Won't Launch

- Check that the app is in `/Applications/Octobud.app`
- Try launching from Terminal: `open /Applications/Octobud.app`
- Check Console.app for error messages

### Menu Bar Icon Not Appearing

- Check that CGO is enabled (required for menu bar support)
- Restart the app

### Browser Doesn't Open

- Check that port 8808 is not in use: `lsof -i :8808`
- Manually open `http://localhost:8808` in your browser
- Use `--no-open` flag if you don't want automatic browser opening

### Sync Not Working

- Verify GitHub token is configured in Settings → Account
- Check the token has `notifications` and `repo` scopes
- Check logs for error messages (see Logs section above)

### Database Issues

The SQLite database is stored in the data directory. If you need to reset:

```bash
# Stop Octobud first (quit from menu bar or kill process)
rm ~/Library/Application\ Support/Octobud/octobud.db
# Restart Octobud - it will create a fresh database
```

### Port Already in Use

If port 8808 is already in use, use a different port:

```bash
./bin/octobud --port 8809
```

Then access at `http://localhost:8809`.

## Related

- [Start Here](start-here.md) - Initial setup and core workflows
- [Contributing](CONTRIBUTING.md) - Development setup
