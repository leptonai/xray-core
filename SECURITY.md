# Security Policy: leptonai/xray-core

## Reporting a Vulnerability

Do not open a public GitHub issue or pull request for a suspected vulnerability. The issue templates under `.github/ISSUE_TEMPLATE/` are for functional bug reports and are public.

Report vulnerabilities privately through one of these channels:

1. **NVIDIA Vulnerability Disclosure Program (preferred):** [NVIDIA Product Security](https://www.nvidia.com/en-us/security/), using the [vulnerability report form](https://www.nvidia.com/en-us/security/report-vulnerability/).
2. **Email:** [psirt@nvidia.com](mailto:psirt@nvidia.com). Use the [NVIDIA public PGP key](https://www.nvidia.com/en-us/security/pgp-key) when the report contains sensitive evidence.
3. **GitHub private vulnerability reporting:** This repository's **Security** tab, then **Report a vulnerability**. If that option is not shown, use channel 1 or 2.

Include the repository name, the affected commit or image tag, the protocol and transport in use, the vulnerability type, reproduction steps, proof of concept if available, and the impact. Remove user IDs, private keys, and real server addresses from configuration examples. Use the private channel to share anything that is itself sensitive.

NVIDIA PSIRT acknowledges receipt, validates the report, and coordinates remediation and disclosure. For this repository, remediation is a source change on `main` and a rebuilt image. If the issue is in code inherited unchanged from upstream Xray-core, say so in the report.

## Security Architecture and Context

This repository is a modified copy of [XTLS/Xray-core](https://github.com/XTLS/Xray-core), a network proxy platform. It keeps the upstream Go module path `github.com/xtls/xray-core` and is based on the upstream 1.8.11 line (`core/core.go`). It is a service: a proxy daemon, started through the `xray` command in `main/`.

The maintained copy is the `main` branch. There are no tags and no releases in this repository. The upstream layout is unchanged:

- `proxy/`: protocol implementations, including VLESS, VMess, Trojan, Shadowsocks, SOCKS, HTTP, WireGuard, `dokodemo`, and `freedom`.
- `transport/internet/`: transports and security layers, including TCP, WebSocket, gRPC, QUIC, TLS, and REALITY.
- `app/`: the dispatcher, the router, inbound and outbound managers, DNS, statistics, logging, and the gRPC control API (`app/commander`, `app/proxyman/command`, `app/router/command`, `app/stats/command`).
- `infra/conf/`: the JSON configuration format.

The changes made in this repository are confined to about twenty files:

- A weighted round-robin balancer strategy, `wrr` (`app/router/strategy_wrr.go`, `infra/conf/router_strategy.go`, `app/router/config.proto`), and changes to rule reloading in `app/router/router.go`.
- Keep-alive tickers in `proxy/proxy.go`, `proxy/vless/encoding/encoding.go`, and `proxy/dokodemo/dokodemo.go` that refresh a connection's activity timer every 10 seconds while a copy is in progress, plus extra per-connection log lines there and in `common/signal/timer.go`.
- Log file rotation in `common/log/logger.go` through `lumberjack`: 500 MB per file, seven backups, seven days, compressed.
- Dependency updates in `go.mod`.

`.github/workflows/docker.yml` builds `.github/docker/Dockerfile` on every push to `main` and pushes the image to the GitHub Container Registry with the `latest` tag. `release.yml` builds binaries, `test.yml` runs `go test`, and `.github/dependabot.yml` asks for daily Go module and Actions updates.

Its security responsibility is to authenticate proxy clients, protect the traffic it carries with the configured transport security, and forward traffic only as the configuration allows.

The boundaries are:

- The inbound listeners defined in the configuration, which face untrusted networks.
- The outbound connections made on behalf of authenticated clients.
- The gRPC control API, when the configuration enables it on an inbound.
- The configuration file, which holds client IDs and private keys, and the geodata files loaded at start.
- The log files.

**Repository Exposure Classification:** Public. Basis: `github.com/leptonai/xray-core` is a public GitHub repository; this document is written to public-safe detail.

**Service Exposure Classification:** External / Regulated (medium confidence). Basis: public source for a network proxy, with a workflow that publishes a container image; the source does not show where builds of this repository are deployed.

### Threat Model

1. **Upstream fixes are not in this tree:** The code is based on the upstream 1.8.11 line from May 2024, and since then only dependency updates and the changes listed above have been applied. Security fixes released upstream after that point are absent unless someone ports them. Dependabot pull requests for network and cryptography modules are open at the time of writing.
2. **The control API has no authentication:** The services in `app/commander`, `app/proxyman/command`, `app/router/command`, and `app/stats/command` accept any caller that can reach the API inbound. Such a caller can add or remove inbounds, outbounds, and users, and can change routing. The API is safe only when the configuration binds it to loopback or to a private interface.
3. **The image ships a sample configuration with a fixed client ID:** `.github/docker/files/config.json` is copied into the image as `/etc/xray/config.json`. It defines a VMess inbound on port 9000 with a client ID that is published in this repository. A container started without replacing that file is a proxy that anyone who has read the file can use.
4. **Connections are kept alive during a copy:** The tickers added in `proxy/proxy.go`, `proxy/vless/encoding/encoding.go`, and `proxy/dokodemo/dokodemo.go` refresh the activity timer whether or not data is moving. The idle timeout then no longer closes a connection whose peer has stalled in that phase, so stalled peers hold sockets and goroutines for longer than the policy level intends.
5. **Logs record who connected to what:** Access and error logs contain client addresses and destinations, and the added log lines increase the volume per connection. `common/log/logger.go` keeps up to seven compressed backups for seven days. Anyone who can read the log directory learns users' traffic metadata.
6. **Authenticated clients can reach whatever the host can reach:** The `freedom` outbound connects to any destination a client asks for. Unless routing rules block them, that includes private addresses and metadata services on the server's own network.
7. **The image build uses inputs that are not pinned:** `.github/docker/Dockerfile` refers to `golang:alpine` and `alpine:latest` by tag and downloads `geoip.dat` and `geosite.dat` from a third-party release at build time without a checksum. The workflows refer to third-party actions by tag. Every push to `main` republishes `latest`.

### Critical Security Assumptions

- The upstream protocol and transport implementations are trusted as of the version this tree is based on. This repository does not audit them.
- The operator supplies a configuration with unique client IDs and keys, and never runs the image with the bundled sample configuration.
- The gRPC control API is bound to loopback or to an interface that only trusted management hosts can reach.
- Configuration files, private keys, and log files are readable only by the account that runs the proxy.
- A client that holds a valid ID is trusted to send traffic anywhere the routing rules allow. The rules block destinations that clients must not reach.
- The geodata files and the base images are trusted at build time.
- The host's file-descriptor and memory limits bound the cost of connections that stay open.

## Supported Versions

This repository has no tags and no releases. `main` is the only maintained copy, and images are built from it. Security reports should name the commit, or the image tag and digest. Fixes for code inherited from upstream arrive when this tree is updated from upstream.
