# Security Policy

## Reporting a Vulnerability

Please report security vulnerabilities **privately** — do not open a public
GitHub issue.

- Open a private security advisory: GitHub **Security** tab →
  **Report a vulnerability**, or
- Email: gary@blankenship.me

Include:
- A description of the issue and its impact.
- Steps to reproduce (proof of concept, affected versions, flags/config used).
- Any suggested fix.

Expect an initial response within 72 hours. Please do not disclose the issue
publicly until a fix has been released.

## Scope

This policy covers the `crawler` CLI in this repository.

`crawler` is network-facing by design. Crawling hosts you do not own can impose
load and may be governed by `robots.txt`, terms of service, or law. The crawler
honours `robots.txt` and stays within the seed's domain and base path by
default; use `--max-pages`, keep scope narrow, and leave `--no-robots` off when
crawling third-party sites.
