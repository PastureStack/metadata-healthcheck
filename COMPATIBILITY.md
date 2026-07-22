# Compatibility

The migration preserves these externally observable behaviors:

- metadata API version `2015-12-19` by default;
- metadata resources `/services`, `/containers`, `/self/host`, and `/version`;
- healthcheck fields for port, interval, thresholds, request line, and response timeout;
- HAProxy-based TCP or HTTP probes;
- `serviceEvents` creation with healthcheck identity, state, and external timestamp;
- the transition rule that reports `INIT` before the first complete `UP` state.
- a bounded startup wait for the metadata service, which permits normal system-stack ordering without hiding a persistent outage;
- a container-network-only HAProxy readiness listener on TCP 42 and a root-owned statistics socket used only inside the container; the Catalog does not publish TCP 42 on a host port.

Public configuration names use `PLATFORM_*`. The default metadata URL uses the protocol's link-local address `169.254.169.250` because historical deployments do not provide a neutral DNS alias during rolling upgrades. Compatible system stacks still inject per-instance credentials as `CATTLE_URL`, `CATTLE_ACCESS_KEY`, and `CATTLE_SECRET_KEY`; the container entrypoint maps those values only when the corresponding neutral variable is absent. These names are an input adapter at the compatibility boundary, not new public configuration, and explicit `PLATFORM_*` values always take precedence.

Compatibility must be verified in an isolated Linux VM before a release. A successful Windows build does not prove HAProxy, Unix socket, CA replacement on restart, or container-network behavior.
