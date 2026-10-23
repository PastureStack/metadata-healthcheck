# Metadata Healthcheck

Current source builds use Go 1.27.3 while the Go language minimum remains 1.26.0. Ubuntu 26.04 images install the checksum-pinned `ca-certificates` 20260816ubuntu0.26.04.1 package with only officially fixed, already-installed packages advanced by the reviewed security overlay. These source inputs take effect in newly built artifacts; existing published release coordinates below describe their original artifacts.

Metadata Healthcheck is a node-side agent that converts metadata healthcheck declarations into HAProxy probes and reports stable state changes to a compatible control API.

The current published Linux release is [v0.3.17](https://github.com/PastureStack/metadata-healthcheck/releases/tag/v0.3.17).

PastureStack is an independent community effort to preserve, audit, and modernize the Rancher 1.6 ecosystem. It is not affiliated with or endorsed by Rancher Labs or SUSE.

**Upstream:** [`rancher/healthcheck`](https://github.com/rancher/healthcheck). This GitHub fork retains the upstream Git history, authorship, dates, and license notices unchanged; PastureStack maintenance is consolidated into one commit after the preserved upstream boundary.

## Runtime contract

The default metadata endpoint is the control plane's link-local endpoint, `http://169.254.169.250/2015-12-19`. This avoids depending on a deployment-specific DNS alias during system-stack upgrades. Override it with `--metadata-url` only when the compatible metadata service is exposed elsewhere.

Control API reporting requires:

- `PLATFORM_URL`
- `PLATFORM_ACCESS_KEY`
- `PLATFORM_SECRET_KEY`

The agent reads `/services`, `/containers`, `/self/host`, and the long-polling `/version` endpoint. It creates `serviceEvents` with `healthcheckUuid`, `reportedHealth`, and `externalTimestamp` fields.

The release container runs the agent and its private HAProxy process. It does not require a Docker socket or additional Linux capabilities, but it currently runs as root inside the container so it can validate, install, and reload HAProxy configuration before HAProxy drops privileges.

For managed system stacks, the container includes a narrow compatibility adapter for the per-instance credential names emitted by the compatible control plane. Operators should continue to configure `PLATFORM_*`; explicit neutral values take precedence. TCP 42 is used only as an un-published container readiness endpoint for the system-stack health monitor.

Routine startup, configuration, monitoring, and state-change messages are
written to standard output. Metadata, HAProxy, and control API failures are
written to standard error so log viewers can distinguish operational
information from actionable errors.

```sh
docker run --rm \
  --network platform-system \
  -e PLATFORM_URL=https://control.example \
  -e PLATFORM_ACCESS_KEY=replace-me \
  -e PLATFORM_SECRET_KEY=replace-me \
  ghcr.io/pasturestack/metadata-healthcheck:v0.3.17 \
  --metadata-url http://169.254.169.250/2015-12-19
```

Use scoped runtime credentials and route the link-local metadata address only inside the managed host network. Do not place credentials in an image, Compose file committed to Git, or command history.

## Local checks

```sh
make test
make validate
make build
IMAGE=ghcr.io/pasturestack/metadata-healthcheck:v0.3.17 make integration
```

The integration target verifies the HAProxy process and Unix socket, metadata long polling, authenticated service-event reporting, and the container argument contract on an isolated Docker network.

## Security

Do not use production credentials for development. See [SECURITY.md](SECURITY.md) for reporting guidance and [COMPATIBILITY.md](COMPATIBILITY.md) for the preserved protocol boundary.

## License and provenance

This project is distributed under the Apache License 2.0. See [LICENSE](LICENSE), [ORIGIN.md](ORIGIN.md), and [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

Security micro-updates are constrained by `security/ubuntu-security-packages.lock` to the signed Ubuntu 26.04 snapshot of 2026-10-23; the general package snapshot remains unchanged. Builds record the resulting package versions in `/usr/share/pasturestack/ubuntu-security-packages.tsv`. Existing runtime and firewall contracts are unchanged.
