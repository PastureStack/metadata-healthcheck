# syntax=docker/dockerfile:1.7
FROM ubuntu:26.04@sha256:2260313b31c8c011cd2eebe728008efac1b3982be73eb71348ea2648d2c0e09b

ARG UBUNTU_SNAPSHOT=20260825T000000Z

ADD --checksum=sha256:5b6326f1932cfd7be57130561dc78192f09ed9f58bfbe6606ed0c50510c3b6cd \
    https://archive.ubuntu.com/ubuntu/pool/main/c/ca-certificates/ca-certificates_20260816ubuntu0.26.04.1_all.deb \
    /tmp/ca-certificates.deb
ARG IMAGE_VERSION=dev
ARG IMAGE_CREATED=2026-07-22T09:31:42Z
ARG IMAGE_REVISION=unknown
ENV DEBIAN_FRONTEND=noninteractive

RUN set -eux; \
    mkdir -p /tmp/ca-bootstrap /etc/ssl/certs; \
    dpkg-deb --extract /tmp/ca-certificates.deb /tmp/ca-bootstrap; \
    find /tmp/ca-bootstrap/usr/share/ca-certificates -type f -name '*.crt' | LC_ALL=C sort | while IFS= read -r certificate; do sed -e '$a\' "${certificate}"; done > /etc/ssl/certs/ca-certificates.crt; \
    rm -rf /tmp/ca-bootstrap; \
    rm -f /etc/apt/sources.list /etc/apt/sources.list.d/*.list /etc/apt/sources.list.d/*.sources; \
    printf 'Types: deb\nURIs: https://snapshot.ubuntu.com/ubuntu/%s\nSuites: resolute resolute-updates resolute-backports resolute-security\nComponents: main universe restricted multiverse\nSigned-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg\nSnapshot: no\n' "${UBUNTU_SNAPSHOT}" > /etc/apt/sources.list.d/pasturestack-snapshot.sources; \
    printf 'Acquire::Retries "5";\nAcquire::http::Timeout "30";\nAcquire::https::Timeout "30";\nAcquire::http::Pipeline-Depth "0";\nAcquire::https::CaInfo "/etc/ssl/certs/ca-certificates.crt";\nAcquire::https::Verify-Peer "true";\nAcquire::https::Verify-Host "true";\nAcquire::AllowInsecureRepositories "false";\nAPT::Get::AllowUnauthenticated "false";\n' > /etc/apt/apt.conf.d/80pasturestack-snapshot; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
        bash /tmp/ca-certificates.deb curl haproxy iputils-ping libcap2-bin netcat-openbsd tini; \
    test "$(dpkg-query -W -f='${Version}' ca-certificates)" = '20260816ubuntu0.26.04.1'; \
    rm -f /tmp/ca-certificates.deb; \
    rm -rf /var/lib/apt/lists/* /var/cache/apt/* /var/log/apt/*; \
    rm -f /var/log/alternatives.log /var/log/dpkg.log; \
    rm -f /usr/bin/pebble

# ldconfig records filesystem traversal details in this optional cache. The
# runtime loader uses /etc/ld.so.cache; removing aux-cache makes clean builds
# content-identical without changing runtime resolution.
RUN rm -f /var/cache/ldconfig/aux-cache

RUN mkdir -p /etc/haproxy /licenses
COPY metadata-healthcheck platform-entrypoint.sh update-platform-ca haproxy_reload.sh /usr/bin/
COPY haproxy_template.cfg /etc/haproxy/haproxy_template.cfg
COPY LICENSE ORIGIN.md THIRD_PARTY_NOTICES.md /licenses/

RUN chmod +x /usr/bin/metadata-healthcheck /usr/bin/platform-entrypoint.sh /usr/bin/update-platform-ca /usr/bin/haproxy_reload.sh

LABEL org.opencontainers.image.title="metadata-healthcheck" \
      org.opencontainers.image.description="Metadata-driven node healthcheck agent" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.source="https://github.com/PastureStack/metadata-healthcheck" \
      org.opencontainers.image.created="${IMAGE_CREATED}" \
      org.opencontainers.image.revision="${IMAGE_REVISION}" \
      org.opencontainers.image.version="${IMAGE_VERSION}"

ENTRYPOINT ["/usr/bin/platform-entrypoint.sh", "/usr/bin/tini", "--", "/usr/bin/metadata-healthcheck"]
CMD []
