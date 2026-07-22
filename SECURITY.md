# Security Policy

The latest PastureStack release tag receives best-effort security maintenance. Historical upstream tags and superseded PastureStack image digests do not receive fixes.

Please report suspected vulnerabilities privately to the PastureStack organization maintainers. Do not include credentials, private registry addresses, production metadata, certificates, or customer data in a public issue.

Development and integration tests must use disposable environments and scoped test credentials. Public disclosure should wait until maintainers confirm a coordinated remediation plan. Deployments should pin the reviewed GHCR digest and review the accompanying SBOM rather than relying on a mutable local tag.
