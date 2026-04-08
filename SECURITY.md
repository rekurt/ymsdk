# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in ymsdk, please report it responsibly.

**Do NOT open a public GitHub issue for security vulnerabilities.**

Instead, please report vulnerabilities by opening a
[private security advisory](https://github.com/rekurt/ymsdk/security/advisories/new)
on GitHub.

You should receive a response within 48 hours. We will work with you to
understand the issue and address it promptly.

## Security Considerations

This SDK handles API tokens for the Yandex Messenger Bot API. Please ensure:

- Never commit API tokens to version control
- Use environment variables or secret management for token storage
- Keep the SDK updated to the latest version
- Review the `ymerrors` package for proper error handling of authentication failures
