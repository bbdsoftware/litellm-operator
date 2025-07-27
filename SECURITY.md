# Security Policy

## Supported Versions

We release patches for security vulnerabilities. Which versions are eligible for receiving such patches depends on the CVSS v3.0 Rating:

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

The LiteLLM Operator team and community take security bugs seriously. We appreciate your efforts to responsibly disclose your findings, and will make every effort to acknowledge your contributions.

### How to Report a Security Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via one of the following methods:

1. **Email**: Send an email to [security@your-domain.com](mailto:security@your-domain.com)
2. **Private GitHub Security Advisory**: Use GitHub's [private vulnerability reporting feature](https://github.com/your-org/litellm-operator/security/advisories/new)

### What to Include

Please include the following information in your report:

- Type of issue (e.g. buffer overflow, SQL injection, cross-site scripting, etc.)
- Full paths of source file(s) related to the manifestation of the issue
- The location of the affected source code (tag/branch/commit or direct URL)
- Any special configuration required to reproduce the issue
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit the issue

### Response Timeline

- **Initial Response**: We will acknowledge receipt of your vulnerability report within 48 hours
- **Status Updates**: We will send you regular updates about our progress, at least every 7 days
- **Resolution**: We aim to resolve critical vulnerabilities within 30 days of the initial report

### Security Response Process

1. **Triage**: The security team will assess the severity and impact of the vulnerability
2. **Investigation**: We will investigate and develop a fix
3. **Disclosure**: We will coordinate the release of the fix and public disclosure
4. **Recognition**: We will acknowledge your contribution (unless you prefer to remain anonymous)

### Security Best Practices

When deploying LiteLLM Operator, we recommend:

- **Principle of Least Privilege**: Run the operator with minimal required permissions
- **Network Security**: Use network policies to restrict operator network access
- **Regular Updates**: Keep the operator and its dependencies up to date
- **Monitoring**: Monitor operator logs for suspicious activity
- **Secrets Management**: Use Kubernetes secrets or external secret management systems for sensitive data

### Vulnerability Disclosure Policy

- We will provide advance notification to users about security updates when possible
- Security advisories will be published on GitHub Security Advisories
- We follow a coordinated disclosure timeline of 90 days from initial report to public disclosure

## Questions?

If you have any questions about this security policy, please contact us at [security@your-domain.com](mailto:security@your-domain.com). 