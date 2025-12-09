# Releasing Octobud

This guide covers how to create releases of Octobud, including code signing for macOS.

## Local vs CI Builds

There are two ways to build Octobud for macOS:

1. **Local Build** (`macos/build-local.sh`) - For testing and development
2. **CI Build** (GitHub Actions) - For official releases

### Local Build

To build locally for testing:

```bash
# Build unsigned (for testing)
./macos/build-local.sh --skip-sign

# Build signed (requires certificates in Keychain)
VERSION=1.0.0 ./macos/build-local.sh

# Build signed and notarized (requires API credentials)
VERSION=1.0.0 ./macos/build-local.sh --notarize
```

The local build script:
- Builds the frontend
- Compiles the Go binary
- Creates a `.app` bundle
- Optionally signs and notarizes
- Creates a `.pkg` installer

Output is placed in the `dist/` directory.

### CI Build (Automated)

Releases are created automatically via GitHub Actions when you push a version tag:

```bash
# Create and push a tag
git tag -s v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

The automated workflow (`.github/workflows/release.yml`) will:
1. Build the frontend
2. Build Go binaries for macOS (arm64 and amd64)
3. Create macOS `.app` bundles
4. Sign and notarize (if signing secrets are configured)
5. Create `.pkg` installers for both architectures
6. Generate SHA-256 checksums
7. Create a GitHub Release with all artifacts

The workflow automatically:
- Extracts version from the Git tag
- Builds for both macOS architectures
- Signs with Developer ID certificates (if configured)
- Notarizes with Apple (if API credentials configured)
- Marks pre-releases appropriately (e.g., `v1.0.0-beta.1`)

> **Note:** macOS has full support (menu bar, auto-start). Linux and Windows have limited support (core functionality works, but menu bar and auto-start are not available).

### Manual Releases

You can also trigger a release manually from the Actions tab:
1. Go to Actions → Release
2. Click "Run workflow"
3. Enter the version (e.g., `v1.0.0`)
4. Click "Run workflow"

This is useful for re-running a failed release without creating a new tag.

## Version Format

Use semantic versioning: `vMAJOR.MINOR.PATCH`

- `v1.0.0` - Stable release
- `v1.0.0-beta.1` - Pre-release (marked as prerelease on GitHub)
- `v1.0.0-rc.1` - Release candidate

## macOS Code Signing

To create signed and notarized macOS releases, you need to configure Apple Developer credentials as GitHub secrets.

### Required Secrets

Configure these secrets in your GitHub repository: **Settings → Secrets and variables → Actions → New repository secret**

| Secret | Description | How to Obtain |
|--------|-------------|---------------|
| `APPLE_APP_CERT_P12` | Base64-encoded Developer ID Application certificate (.p12 file) | Export from Keychain Access, convert to base64 |
| `APPLE_INSTALLER_CERT_P12` | Base64-encoded Developer ID Installer certificate (.p12 file) | Export from Keychain Access, convert to base64 |
| `APPLE_CERTIFICATE_PASSWORD` | Password used when exporting the .p12 files | The password you set when exporting |
| `APPLE_SIGNING_IDENTITY` | Your organization or personal name as shown in certificate | Found in certificate details (e.g., "Your Company LLC") |
| `APPLE_TEAM_ID` | 10-character Team ID | Found in [Apple Developer → Membership](https://developer.apple.com/account/#!/membership) |
| `APPLE_API_KEY_ID` | App Store Connect API Key ID | Created in [App Store Connect → Users and Access → Keys](https://appstoreconnect.apple.com/access/api) |
| `APPLE_API_KEY_ISSUER` | App Store Connect Issuer ID | Found in App Store Connect API key details |
| `APPLE_API_KEY_P8` | Base64-encoded .p8 private key | Download from App Store Connect (only available once), convert to base64 |

### Prerequisites

- **Apple Developer Account** - [developer.apple.com](https://developer.apple.com)
- **Two certificates** from Apple Developer portal:
  - Developer ID Application (signs the .app bundle)
  - Developer ID Installer (signs the .pkg installer)
- **App Store Connect API key** (for notarization) with "Developer" role access

### Converting Files to Base64

Certificates and API keys need to be base64-encoded before adding as secrets:

```bash
# For .p12 certificates
base64 -i certificate.p12 | pbcopy

# For .p8 API key
base64 -i AuthKey_XXXXXXXXXX.p8 | pbcopy
```

### Unsigned Releases

If signing secrets aren't configured, the workflow will still produce unsigned `.pkg` files. Users will see an "unidentified developer" warning but can bypass it by right-clicking → Open.

## Testing Releases

Before publishing a release, test the build:

1. **Build locally** with `--skip-sign` to verify the build process works
2. **Test the .app bundle**:
   ```bash
   open dist/Octobud.app
   ```
   Verify:
   - App launches and shows menu bar icon
   - Browser opens to `http://localhost:8808`
   - App functions correctly

3. **Test the .pkg installer**:
   ```bash
   open dist/Octobud-*.pkg
   ```
   Verify:
   - Installer runs successfully
   - App is installed to `/Applications`
   - App launches after installation

4. **For signed builds**, verify signatures:
   ```bash
   codesign --verify --deep --strict dist/Octobud.app
   pkgutil --check-signature dist/Octobud-*.pkg
   ```

## Troubleshooting

### Local Build Issues

**Build script fails with "command not found"**
- Ensure you're running on macOS
- Check that required tools are installed: `npm`, `go`, `pkgbuild`, `codesign`

**Frontend build fails**
- Run `cd frontend && npm install` first
- Check Node.js version (requires 18+)

**Code signing fails**
- Verify certificates are in Keychain: `security find-identity -v -p codesigning`
- Ensure certificates are not expired
- Check that you have the correct Developer ID certificates (not Mac Developer)

**Notarization fails**
- Verify API credentials are set correctly
- Check that the package was signed before notarization
- Review notarization logs (see below)

### CI Build Issues

**Workflow fails on signing step**
- Verify all required secrets are set in GitHub repository settings
- Check that secrets are base64-encoded correctly
- Ensure `APPLE_SIGNING_IDENTITY` matches exactly what's in your certificate

**Notarization fails in CI**
- Check that all three API key secrets are set: `APPLE_API_KEY_ID`, `APPLE_API_KEY_ISSUER`, `APPLE_API_KEY_P8`
- Verify the API key has notarization permissions
- Review workflow logs for specific error messages

### Notarization Failed

Check the notarization log:
```bash
xcrun notarytool log <submission-id> \
  --key ~/private_keys/AuthKey_XXXX.p8 \
  --key-id XXXX \
  --issuer XXXX
```

Common issues:
- **Hardened runtime not enabled**: The binary must be signed with `--options runtime` (already handled in build scripts)
- **Missing timestamp**: Use `--timestamp` when signing (already handled in build scripts)
- **Invalid entitlements**: Some entitlements require specific capabilities
- **Binary not signed correctly**: Ensure binary is signed before app bundle, and app bundle is signed before creating .pkg

### Certificate Expired

Developer ID certificates are valid for 5 years. To renew:
1. Create a new certificate in Apple Developer portal
2. Download and install in Keychain
3. Export as .p12 and update GitHub secrets
4. For local builds, the script will automatically find the new certificate

### Differences Between Local and CI Builds

- **Local builds** use certificates from your Keychain
- **CI builds** import certificates from GitHub secrets
- **Local builds** can skip signing/notarization for quick testing
- **CI builds** automatically handle both architectures (arm64 and amd64)
- **CI builds** create releases with all platform artifacts

## Release Artifacts

Each release includes:

| Platform | File | Description |
|----------|------|-------------|
| macOS (Apple Silicon) | `Octobud-X.Y.Z-macos-arm64.pkg` | Signed installer for M1/M2/M3 Macs |
| macOS (Intel) | `Octobud-X.Y.Z-macos-amd64.pkg` | Signed installer for Intel Macs |
| All | `checksums.txt` | SHA-256 checksums for all files |

> **Note:** macOS has full support (menu bar, auto-start). Linux and Windows have limited support (core functionality works, but menu bar and auto-start are not available).

## Release Checklist

Before creating a release:

- [ ] Update version in `VERSION` file (e.g., `1.0.0`)
- [ ] Update `CHANGELOG.md` with release notes
- [ ] Verify all tests pass (`make test`)
- [ ] Review breaking changes and document them
- [ ] Update documentation if needed
- [ ] Create signed Git tag: `git tag -s vX.Y.Z -m "Release vX.Y.Z"`
- [ ] Push tag: `git push origin vX.Y.Z`

After release:

- [ ] Verify artifacts are downloadable from GitHub Releases
- [ ] Test installation from release artifacts
- [ ] Verify signatures and checksums
- [ ] Update any documentation links to latest version
- [ ] Announce release (if applicable)

## macOS Installer Details

The `.pkg` installer:
1. Installs `Octobud.app` to `/Applications`

Users can access Octobud at `http://localhost:8808`.

