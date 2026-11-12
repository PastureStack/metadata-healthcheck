# syntax=docker/dockerfile:1.7
FROM ghcr.io/pasturestack/ubuntu-security-backports:v0.1.1@sha256:45d6b507b63a18459ae5cf76afd4d9731f8ef43b9c40e4c3dbb01899e1752b19 AS ubuntu_security_backports
FROM ubuntu:26.04@sha256:2260313b31c8c011cd2eebe728008efac1b3982be73eb71348ea2648d2c0e09b

COPY security/install-ubuntu-security-updates.sh /usr/local/lib/pasturestack/install-ubuntu-security-updates.sh
COPY security/ubuntu-security-packages.lock /usr/share/pasturestack/ubuntu-security-packages.lock
COPY security/ubuntu-runtime-security.lock /usr/share/pasturestack/ubuntu-runtime-security.lock

ADD --checksum=sha256:a7e3bf769aec16ba21ca39cba180d6d311dafcbae68346a05f137c5cc26c9ca6 \
    https://security.ubuntu.com/ubuntu/pool/main/c/ca-certificates/ca-certificates_20260816ubuntu0.26.04.2_all.deb \
    /tmp/ca-certificates.deb
ADD --checksum=sha256:0de729b8fe0e1c67c04e8e8d5bd6e0420dbd47f75b93f0b71c8e740edc2d5ffc \
    https://security.ubuntu.com/ubuntu/pool/main/a/apt/apt_3.2.1ubuntu0.1_amd64.deb \
    /tmp/apt.deb
ADD --checksum=sha256:94e5d41303265b66af0f8c70ade18bfb77260ad286edee6c8045faddef82b94c \
    https://security.ubuntu.com/ubuntu/pool/main/a/apt/libapt-pkg7.0_3.2.1ubuntu0.1_amd64.deb \
    /tmp/libapt-pkg7.0.deb
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
    printf 'Types: deb\nURIs: https://archive.ubuntu.com/ubuntu\nSuites: resolute\nComponents: main universe\nSigned-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg\nSnapshot: no\n' > /etc/apt/sources.list.d/pasturestack-snapshot.sources; \
    printf 'Acquire::Retries "5";\nAcquire::http::Timeout "30";\nAcquire::https::Timeout "30";\nAcquire::http::Pipeline-Depth "0";\nAcquire::https::CaInfo "/etc/ssl/certs/ca-certificates.crt";\nAcquire::https::Verify-Peer "true";\nAcquire::https::Verify-Host "true";\nAcquire::AllowInsecureRepositories "false";\nAPT::Get::AllowUnauthenticated "false";\n' > /etc/apt/apt.conf.d/80pasturestack-snapshot; \
    dpkg -i /tmp/apt.deb /tmp/libapt-pkg7.0.deb; \
    for package in apt libapt-pkg7.0; do \
        test "$(dpkg-query -W -f='${Version}' "$package")" = '3.2.1ubuntu0.1'; \
    done; \
    rm -f /tmp/apt.deb /tmp/libapt-pkg7.0.deb; \
    apt-get update --error-on=any; \
    apt-get install -y --no-install-recommends \
        bash /tmp/ca-certificates.deb curl haproxy iputils-ping libcap2-bin netcat-openbsd tini; \
    test "$(dpkg-query -W -f='${Version}' ca-certificates)" = '20260816ubuntu0.26.04.2'; \
    sh /usr/local/lib/pasturestack/install-ubuntu-security-updates.sh /usr/share/pasturestack/ubuntu-security-packages.lock /etc/ssl/certs/ca-certificates.crt; \
    sh /usr/local/lib/pasturestack/install-ubuntu-security-updates.sh /usr/share/pasturestack/ubuntu-runtime-security.lock /etc/ssl/certs/ca-certificates.crt 20261112T000000Z; \
    test "$(dpkg-query -W -f='${Version}' haproxy)" = '3.2.25-0ubuntu0.26.04.2'; \
    /usr/sbin/haproxy -v | grep -F 'HAProxy version 3.2.25-0ubuntu0.26.04.2'; \
    test "$(dpkg-query -W -f='${Version}' perl-base)" = '5.40.1-7ubuntu0.4'; \
    dpkg-query -W -f='${binary:Package}\t${Version}\n' | LC_ALL=C sort > /usr/share/pasturestack/ubuntu-security-packages.tsv; \
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

# Install only qualified packages already present after the final Ubuntu update.
# This read-only build mount leaves no installer, archive or extra runtime tool.
RUN --mount=from=ubuntu_security_backports,source=/out,target=/opt/pasturestack/ubuntu-security-backports,ro \
    set -eux; \
    evidence=/usr/share/pasturestack/security/ubuntu-backports; \
    mkdir -p "${evidence}"; \
    /opt/pasturestack/ubuntu-security-backports/install-existing-backports --install-in-docker-build > "${evidence}/install.json"; \
    /opt/pasturestack/ubuntu-security-backports/install-existing-backports --verify-installed > "${evidence}/verify.json"; \
    for document in manifest.json qualification.json openvex.json source-inputs.lock.json; do \
        cp "/opt/pasturestack/ubuntu-security-backports/${document}" "${evidence}/${document}"; \
    done; \
    printf '%s\n' 'ghcr.io/pasturestack/ubuntu-security-backports:v0.1.1@sha256:45d6b507b63a18459ae5cf76afd4d9731f8ef43b9c40e4c3dbb01899e1752b19' > "${evidence}/bundle-image.txt"; \
    (cd "${evidence}" && sha256sum manifest.json qualification.json openvex.json source-inputs.lock.json install.json verify.json bundle-image.txt > SHA256SUMS); \
    dpkg-query -W -f='${binary:Package}\t${Version}\n' | LC_ALL=C sort > /usr/share/pasturestack/ubuntu-security-packages.tsv; \
    rm -f /var/cache/ldconfig/aux-cache

ENTRYPOINT ["/usr/bin/platform-entrypoint.sh", "/usr/bin/tini", "--", "/usr/bin/metadata-healthcheck"]
CMD []
