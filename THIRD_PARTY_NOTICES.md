# Third-Party Notices

The Go source currently uses only the standard library. The container image installs Ubuntu, HAProxy, CA certificates, curl, iputils, libcap, netcat, and tini from the configured Ubuntu package repositories. Their licenses and notices remain governed by those distributions and must be included by any final image or software-bill-of-materials process.

Historical vendored dependencies were removed from the current tree after their required behavior was replaced with small local implementations. Their original licenses, notices, and authors remain available in the preserved upstream Git history and must be reviewed again if any code is restored.
