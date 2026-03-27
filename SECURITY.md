# Security Policy

## Supported Versions

Clawrise is pre-1.0 and still moving quickly.

At this stage, security fixes are only guaranteed for the latest code on the default branch and the most recent tagged release, if one exists.

| Version | Supported |
| --- | --- |
| `main` / default branch | Yes |
| older branches or snapshots | No |

## Reporting a Vulnerability

Please do not disclose vulnerabilities publicly before maintainers have had a reasonable chance to investigate and respond.

Preferred reporting path:

1. Use GitHub private vulnerability reporting for this repository if it is enabled.
2. If private reporting is not available, open a public issue titled `[security] private coordination requested` without exploit details.
3. In that issue, only include enough information for maintainers to establish a private follow-up channel.

When you report a vulnerability, include:

- affected command, package, or operation
- impacted versions or commit hashes when known
- severity and likely impact
- clear reproduction steps
- whether credentials, tenant data, or tokens may be exposed
- suggested mitigations, if you have them

## Response Expectations

Maintainers will try to:

- acknowledge the report within 7 days
- validate the issue and determine impact
- coordinate a fix and release plan
- credit the reporter when disclosure is appropriate

## Handling Sensitive Data

- Never post real credentials in issues or pull requests.
- Sanitize request payloads, logs, and tenant identifiers before sharing them.
- If you are unsure whether something is sensitive, treat it as sensitive.
