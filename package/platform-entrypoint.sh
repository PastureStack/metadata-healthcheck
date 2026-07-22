#!/usr/bin/env bash
set -euo pipefail

# Compatible system stacks inject per-instance credentials with these legacy
# environment names. Keep the public configuration neutral while translating
# the injected values at the container boundary. Explicit PLATFORM_* values
# always take precedence.
if [[ -z "${PLATFORM_URL:-}" && -n "${CATTLE_URL:-}" ]]; then
  export PLATFORM_URL="$CATTLE_URL"
fi
if [[ -z "${PLATFORM_ACCESS_KEY:-}" && -n "${CATTLE_ACCESS_KEY:-}" ]]; then
  export PLATFORM_ACCESS_KEY="$CATTLE_ACCESS_KEY"
fi
if [[ -z "${PLATFORM_SECRET_KEY:-}" && -n "${CATTLE_SECRET_KEY:-}" ]]; then
  export PLATFORM_SECRET_KEY="$CATTLE_SECRET_KEY"
fi

/usr/bin/update-platform-ca
exec "$@"
