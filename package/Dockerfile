# syntax=docker/dockerfile:1.7
FROM ghcr.io/pasturestack/ubuntu-security-backports:v0.1.1@sha256:45d6b507b63a18459ae5cf76afd4d9731f8ef43b9c40e4c3dbb01899e1752b19 AS ubuntu_security_backports
FROM ubuntu:26.04@sha256:2260313b31c8c011cd2eebe728008efac1b3982be73eb71348ea2648d2c0e09b AS qualified_backports

COPY security/install-ubuntu-security-updates.sh /usr/local/lib/pasturestack/install-ubuntu-security-updates.sh
COPY security/ubuntu-security-packages.lock /usr/share/pasturestack/ubuntu-security-packages.lock

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
    printf 'Types: deb\nURIs: https://archive.ubuntu.com/ubuntu\nSuites: resolute resolute-updates resolute-security\nComponents: main universe\nSigned-By: /usr/share/keyrings/ubuntu-archive-keyring.gpg\nSnapshot: no\n' > /etc/apt/sources.list.d/pasturestack-snapshot.sources; \
    printf 'Acquire::Retries "5";\nAcquire::http::Timeout "30";\nAcquire::https::Timeout "30";\nAcquire::http::Pipeline-Depth "0";\nAcquire::https::CaInfo "/etc/ssl/certs/ca-certificates.crt";\nAcquire::https::Verify-Peer "true";\nAcquire::https::Verify-Host "true";\nAcquire::AllowInsecureRepositories "false";\nAPT::Get::AllowUnauthenticated "false";\n' > /etc/apt/apt.conf.d/80pasturestack-snapshot; \
    dpkg -i /tmp/apt.deb /tmp/libapt-pkg7.0.deb; \
    for package in apt libapt-pkg7.0; do \
        test "$(dpkg-query -W -f='${Version}' "$package")" = '3.2.1ubuntu0.1'; \
    done; \
    rm -f /tmp/apt.deb /tmp/libapt-pkg7.0.deb; \
    apt-get update --error-on=any; \
    # Preserve the input version for the existing checksum-qualified Brotli backport.\
    apt-get install -y --no-install-recommends \
        bash /tmp/ca-certificates.deb curl haproxy iputils-ping libbrotli1=1.2.0-3build1 libcap2-bin netcat-openbsd tini; \
    test "$(dpkg-query -W -f='${Version}' ca-certificates)" = '20260816ubuntu0.26.04.2'; \
    sh /usr/local/lib/pasturestack/install-ubuntu-security-updates.sh /usr/share/pasturestack/ubuntu-security-packages.lock /etc/ssl/certs/ca-certificates.crt; \
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

FROM scratch AS official_ubuntu_security
ADD --checksum=sha256:be83aefc14597fba683031a520f303950a7338b95d74decbe625faa51c49933e https://security.ubuntu.com/ubuntu/pool/main/g/glibc/libc6_2.43-2ubuntu2.7_amd64.deb /tmp/official-ubuntu-security/libc6.deb
ADD --checksum=sha256:c0393ea4fa3c45cefe0351db369d3a4a459edc8045856f2e3612ca17e77b2797 https://security.ubuntu.com/ubuntu/pool/main/g/glibc/libc-bin_2.43-2ubuntu2.7_amd64.deb /tmp/official-ubuntu-security/libc-bin.deb
ADD --checksum=sha256:408ba056d39651c2489b59522a614276d7a96020c6a401e73cd0da7d771764b0 https://security.ubuntu.com/ubuntu/pool/main/g/glibc/libc-gconv-modules-extra_2.43-2ubuntu2.7_amd64.deb /tmp/official-ubuntu-security/libc-gconv-modules-extra.deb
# Ubuntu 26.04 resolute-security snapshot, USN-8956-1.
ADD --checksum=sha256:bedeef1cf3268f90338c411cdb0a45b224d0660fd2240f310f380316c8244606 https://snapshot.ubuntu.com/ubuntu/20261107T200000Z/pool/main/b/brotli/libbrotli1_1.2.0-3ubuntu0.1_amd64.deb /tmp/official-ubuntu-security/libbrotli1.deb

FROM qualified_backports AS runtime
RUN --mount=from=official_ubuntu_security,source=/tmp/official-ubuntu-security,target=/tmp/official-ubuntu-security,ro \
    set -eux; \
    set -- \
        'libc6 glibc 2.43-2ubuntu2.7' 'libc-bin glibc 2.43-2ubuntu2.7' \
        'libc-gconv-modules-extra glibc 2.43-2ubuntu2.7' \
        'libbrotli1 brotli 1.2.0-3ubuntu0.1'; \
    for spec in "$@"; do \
        package="${spec%% *}"; \
        source_and_version="${spec#* }"; \
        source="${source_and_version%% *}"; \
        version="${source_and_version#* }"; \
        archive="/tmp/official-ubuntu-security/$package.deb"; \
        test "$(dpkg-deb -f "$archive" Package)" = "$package"; \
        test "$(dpkg-deb -f "$archive" Source)" = "$source"; \
        test "$(dpkg-deb -f "$archive" Architecture)" = amd64; \
        test "$(dpkg-deb -f "$archive" Version)" = "$version"; \
        dpkg --compare-versions "$(dpkg-query -W -f='${Version}' "$package")" lt "$version"; \
    done; \
    dpkg --path-include='/usr/share/doc/libc6/*' --path-include='/usr/share/man/man1/getconf.1.gz' \
        --path-include='/usr/share/man/man1/tzselect.1.gz' --path-include='/usr/share/doc/libbrotli1/*' \
        --install /tmp/official-ubuntu-security/*.deb; \
    audit="$(dpkg --audit)"; \
    test -z "$audit"; \
    for spec in "$@"; do \
        package="${spec%% *}"; \
        version="${spec##* }"; \
        test "$(dpkg-query -W -f='${Version}' "$package")" = "$version"; \
        integrity="$(dpkg --verify "$package")"; \
        test -z "$integrity"; \
    done; \
    getent hosts localhost >/dev/null; \
    curl --version | grep -F 'brotli/'; \
    printf '%s\n' 'PastureStack' | grep -Pqx 'Pasture(?:Stack)'; \
    rm -f /var/cache/ldconfig/aux-cache /var/log/dpkg.log; \
    dpkg-query -W -f='${binary:Package}\t${Version}\n' | LC_ALL=C sort > /usr/share/pasturestack/ubuntu-security-packages.tsv

ENTRYPOINT ["/usr/bin/platform-entrypoint.sh", "/usr/bin/tini", "--", "/usr/bin/metadata-healthcheck"]
CMD []
