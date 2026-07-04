# Operator

<!-- TODO: uncomment badges after first publish
[![CI](https://github.com/qualithm/operator-go/actions/workflows/ci.yaml/badge.svg)](https://github.com/qualithm/operator-go/actions/workflows/ci.yaml)
[![codecov](https://codecov.io/gh/qualithm/operator-go/graph/badge.svg)](https://codecov.io/gh/qualithm/operator-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/qualithm/operator-go.svg)](https://pkg.go.dev/github.com/qualithm/operator-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/qualithm/operator-go)](https://goreportcard.com/report/github.com/qualithm/operator-go)
-->

Go client library and operator CLI for the qualithm platform management API. The shared `operator`
package authenticates with a member API token and backs both the `qualithm` CLI and the forthcoming
MCP server, so the two surfaces never diverge.

## Features

- **`operator` client package** — typed methods over the provisioning surface: authorities,
  enrollments, credentials, devices, and API tokens.
- **`qualithm` CLI** — verbs over every resource, plus an idempotent `apply` for device-as-code
  manifests.
- **Dual output** — human-readable tables by default, stable line-delimited JSON with `--json`.
- **Client-level dry-run** — `--dry-run` reports the planned mutation without sending it; GETs still
  execute so reads stay accurate.
- **Stable exit codes** — scriptable status mapping (auth, not-found, conflict, rate-limited…).

## Installation

```bash
go install github.com/qualithm/operator-go/cmd/qualithm@latest
```

Use the client library directly:

```bash
go get github.com/qualithm/operator-go
```

## Quick Start

The CLI authenticates with a member API token (prefix `qmt_`). Provide it via `--token` or the
`QUALITHM_API_TOKEN` environment variable; point at an environment with `--url` or
`QUALITHM_API_URL` (defaults to `https://api.qualithm.com`).

```bash
export QUALITHM_API_TOKEN=qmt_...

# list devices as a table
qualithm device list

# create a time-boxed enrollment code for a space
qualithm enrollment create --space spc_123 --label lab-floor-2

# mint a device credential, machine-readable
qualithm credential mint --device dev_123 --json

# preview a revoke without applying it
qualithm credential revoke --device dev_123 --credential cred_123 --dry-run
```

Global flags are accepted by every verb and must precede positional arguments:

| Flag        | Env                  | Description                                |
| ----------- | -------------------- | ------------------------------------------ |
| `--url`     | `QUALITHM_API_URL`   | management API base URL                    |
| `--token`   | `QUALITHM_API_TOKEN` | member API token (`qmt_…`)                 |
| `--json`    | —                    | emit JSON instead of a human table         |
| `--dry-run` | —                    | report the planned change without applying |

## Usage

### Resources and verbs

| Resource     | Verbs                                                          |
| ------------ | -------------------------------------------------------------- |
| `authority`  | `list` · `create` · `revoke <id>`                              |
| `enrollment` | `list` · `create` · `revoke <id>`                              |
| `credential` | `list` · `mint` · `cert` · `rotate` · `revoke`                 |
| `device`     | `list` · `get <id>` · `create` · `update <id>` · `delete <id>` |
| `token`      | `list` · `create` · `revoke <id>`                              |
| `apply`      | `<manifest.yaml>` — idempotent device-as-code reconcile        |
| `version`    | print the CLI version                                          |

### Examples

```bash
# devices
qualithm device list --space spc_123 --json
qualithm device get dev_123
qualithm device create --space spc_123 --name gateway-01

# credentials (one-time secrets are printed once, on create)
qualithm credential mint --device dev_123 --label rotate-2025 --expires-at 2025-12-31T00:00:00Z
qualithm credential cert --device dev_123 --csr-file device.csr --expires-days 90
qualithm credential rotate --device dev_123 --credential cred_123 --revoke

# api tokens
qualithm token create --name ci-runner --expires-days 90 --json

# certificate authorities (bring-your-own PEM)
qualithm authority create --name fleet-ca --kind byo --cert-file ca.pem
```

### device-as-code with `apply`

`apply` reconciles a fleet manifest, creating any missing devices (matched by name within a space)
and leaving existing ones untouched. Combine with `--dry-run` to preview.

```yaml
# fleet.yaml
spaces:
  - id: spc_123
    devices:
      - name: gateway-01
      - name: gateway-02
```

```bash
qualithm apply fleet.yaml --dry-run   # show would-create / exists decisions
qualithm apply fleet.yaml             # apply
```

### Exit codes

| Code | Meaning                |
| ---- | ---------------------- |
| 0    | ok (including dry-run) |
| 1    | error                  |
| 2    | usage                  |
| 3    | auth (401 / 403)       |
| 4    | not found (404)        |
| 5    | conflict (409)         |
| 6    | rate limited (429)     |
| 7    | api (other non-2xx)    |

### Using the client library

```go
package main

import (
	"context"
	"fmt"
	"log"

	operator "github.com/qualithm/operator-go"
)

func main() {
	client, err := operator.New("qmt_...")
	if err != nil {
		log.Fatal(err)
	}

	page, err := client.ListDevices(context.Background(), 0, 0)
	if err != nil {
		log.Fatal(err)
	}
	for _, d := range page.Items {
		fmt.Println(d.ID, d.Name)
	}
}
```

Enable a dry-run at construction so mutating calls return a `*operator.DryRunError` instead of
sending:

```go
client, _ := operator.New("qmt_...", operator.WithDryRun(true))
```

## API Reference

Full API documentation is hosted on
[pkg.go.dev](https://pkg.go.dev/github.com/qualithm/operator-go).

Serve docs locally:

```bash
make docs
```

## Examples

See the [`examples/`](examples/) directory for runnable examples:

| Example                                       | Description                          |
| --------------------------------------------- | ------------------------------------ |
| [`basic_usage`](examples/basic_usage/main.go) | Construct a client and list devices. |

```bash
go run ./examples/basic_usage
```

## Development

### Prerequisites

- [Go](https://go.dev/dl/) 1.26+

### Setup

```bash
make install-tools
```

This installs local development tooling, including `golangci-lint`, `goimports`, and `govulncheck`.

> **Note:** Tools are installed to `$GOPATH/bin` (typically `~/go/bin`). Make sure that directory is
> on your `$PATH`, otherwise the installed binaries won't be found.

### Building

```bash
make build
```

### Testing

```bash
make test              # unit tests with race detector
make test-coverage     # with coverage report
```

### Linting & Formatting

```bash
make lint
make fmt
make vet
```

### Security Tooling

```bash
make audit   # govulncheck
make gosec   # standalone gosec scan
make lint    # golangci-lint (includes gosec checks via .golangci.yaml)
```

Daily CI security audit runs both tools in `.github/workflows/audit.yaml`.

## Publishing

Tagged releases are automatically built and published to GHCR when CI passes on `main`. Go modules
are consumed directly from the Git tag (`vX.Y.Z`) via `go install`; no separate registry publish
step is required.

## Minimum Supported Go Version

Go 1.26+.

## License

Apache-2.0

## CI & Branch Protection

The `.github/workflows/ci.yaml` workflow and the `main` / `test` branch rulesets are generated by
[dx](https://github.com/qualithm/dx). To change CI for this repo, edit the relevant archetype in
`dx/ci-templates/` and run `dx ci sync`; do not edit `ci.yaml` directly. The umbrella job at the end
of the workflow supplies the single required status check (`CI Required`).
