# Security Policy

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue, please report it responsibly.

### How to Report

**Please do NOT open a public GitHub issue for security vulnerabilities.**

Instead, please use GitHub's Security Advisory feature:

1. Go to the [Security tab](https://github.com/octobud-hq/octobud/security) on the repository
2. Click **"Report a vulnerability"**
3. Fill out the security advisory form with details about the vulnerability

Alternatively, you can navigate directly to: https://github.com/octobud-hq/octobud/security/advisories/new

Include in your report:

1. **Description** - What is the vulnerability?
2. **Steps to Reproduce** - How can we reproduce it?
3. **Impact** - What is the potential impact?
4. **Suggested Fix** - If you have one (optional)

### What to Expect

- **Acknowledgment** - We'll acknowledge receipt within 48 hours
- **Updates** - We'll keep you informed of our progress
- **Resolution** - We aim to resolve critical issues within 7 days
- **Credit** - We'll credit you in the release notes (unless you prefer anonymity)

### Disclosure Policy

- We follow a coordinated disclosure process
- Please give us reasonable time to address the issue before public disclosure
- We typically aim for disclosure within 90 days or when a fix is released

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |
| < 1.0   | :x:                |

## Desktop Architecture & Token Storage

Octobud is a desktop application that runs locally on your machine. The architecture and security model differ by platform:

### macOS
- **Keychain Support**: ✅ **Available** - GitHub tokens are stored securely in the macOS Keychain using system-level encryption and access controls
- **Token Storage**: Tokens are stored in the macOS Keychain, which provides:
  - System-level encryption managed by macOS
  - Access controls that require user authentication
  - Integration with macOS security features (FileVault, etc.)
- **Security**: Tokens are protected by the macOS Keychain security model

### Linux & Windows
- **Keychain Support**: ❌ **Not Available** - Keychain support is not currently implemented on these platforms
- **Token Storage**: GitHub tokens are encrypted using AES-256-GCM and stored in the SQLite database
- **Encryption Key**: The encryption key is stored in a file (`.token_key`) in the data directory with restricted permissions (0600)
- **Security Limitation**: Because both the encrypted token and the encryption key are stored on the same machine, anyone with file system access to your machine can decrypt and access your GitHub credentials. This is a known limitation for Linux and Windows platforms.

## Security Best Practices

When using Octobud:

1. **Keep Updated** - Run the latest version
2. **Secure Your Token** - Protect your GitHub credentials
   - **On macOS:** Tokens benefit from Keychain security - ensure your Mac is secured with FileVault and a strong password
   - **On Linux & Windows:** Be aware that tokens can be decrypted by anyone with access to your machine. Use strong file system permissions, disk encryption, and limit physical/remote access to your machine
3. **Network Security** - Octobud runs on localhost only by default. If exposing over a network, use a reverse proxy with authentication
4. **Access Control** - Limit who can access your machine and the Octobud data directory

## Known Security Considerations

- **GitHub Token Scope** - Octobud requires `notifications` and `repo` scopes, which grants read access to your repositories
- **Self-Hosted** - You are responsible for securing your own deployment
- **Local-First** - Octobud is designed as a desktop application that trusts localhost. All data is stored locally
- **Single-User Application** - Octobud is designed for single-user use. For network-accessible deployments, consider additional security layers (VPN, reverse proxy authentication, firewall rules)

## Threat Model

Octobud is designed as a local-first desktop application with the following threat assumptions:

1. **Trusted Local Environment**: The application assumes the local machine is trusted. An attacker with physical or remote access to the machine could access stored data.
2. **Localhost Communication**: The HTTP server runs on localhost only and does not implement authentication. This is acceptable for single-user desktop use but inappropriate for network-accessible deployments.
3. **GitHub API Communication**: All GitHub API calls use HTTPS with certificate validation.
4. **Token Storage**: 
   - **macOS**: Uses system Keychain for secure storage. Tokens are protected by macOS Keychain security, which requires user authentication to access.
   - **Linux/Windows**: Tokens are encrypted (AES-256-GCM) and stored in SQLite, but the encryption key is stored in a file (`.token_key`) in the data directory. An attacker with file system access can decrypt tokens. **Keychain support is not available on these platforms.**

## Security Hardening Recommendations

For enhanced security in production or sensitive environments:

1. **Platform Selection**: Prefer macOS for production use due to Keychain support and superior token storage security
2. **Linux/Windows Considerations**: 
   - Use full disk encryption (LUKS, BitLocker, etc.)
   - Restrict file system permissions on the data directory (chmod 700)
   - Ensure the `.token_key` file has restricted permissions (0600)
   - Limit physical and remote access to the machine
   - Consider using a separate user account for running Octobud
3. **Regular Updates**: Keep Octobud and its dependencies updated
4. **Token Rotation**: Periodically rotate your GitHub Personal Access Token
5. **Minimal Scopes**: Use the minimum required GitHub token scopes (`notifications` and `repo`)
6. **Firewall Rules**: Restrict network access to Octobud if running on a shared machine
7. **File Permissions**: Ensure database and key files have restricted permissions (0600)

## Security Contact

For security-related questions or to report vulnerabilities:

- **Preferred**: Use GitHub Security Advisories (see "How to Report" above)
- **Alternative**: Contact the maintainer through GitHub (for sensitive matters, use private communication)

Thank you for helping keep Octobud secure!

