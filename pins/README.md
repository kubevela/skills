# Pins

The third-party server table moved to `kubevela-sre/references/servers.yaml` so that the
SRE skill is self-contained when installed on its own, as the Agent Skills specification
recommends. That file is the source of truth; `vela mcp setup` will embed a copy at release
time. This directory remains only so older links resolve.
