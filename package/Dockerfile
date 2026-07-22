FROM ubuntu:26.04@sha256:3131b4cc82a783df6c9df078f86e01819a13594b865c2cad47bd1bca2b7063bb

ARG UBUNTU_MIRROR=http://archive.ubuntu.com/ubuntu
ARG IMAGE_VERSION=dev
ARG IMAGE_CREATED=2026-07-22T09:31:42Z
ARG IMAGE_REVISION=unknown
ENV DEBIAN_FRONTEND=noninteractive

RUN set -eux; \
    find /etc/apt -type f \( -name '*.list' -o -name '*.sources' \) -exec sed -i \
        -e "s|http://security.ubuntu.com/ubuntu|${UBUNTU_MIRROR}|g" \
        -e "s|http://archive.ubuntu.com/ubuntu|${UBUNTU_MIRROR}|g" {} +; \
    printf 'Acquire::Retries "5";\nAcquire::http::Timeout "30";\nAcquire::https::Timeout "30";\nAcquire::http::Pipeline-Depth "0";\n' > /etc/apt/apt.conf.d/80pasturestack-retries; \
    apt-get update; \
    apt-get install -y --no-install-recommends \
        bash ca-certificates curl haproxy iputils-ping libcap2-bin netcat-openbsd tini; \
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
