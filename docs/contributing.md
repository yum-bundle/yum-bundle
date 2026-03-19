---
layout: default
title: Contributing
nav_order: 6
---

# Contributing

## Development Setup

### Prerequisites

- Go 1.23 or later
- `make`
- An RPM-based Linux system (for running `yum-bundle` itself; tests can run anywhere)

### Clone and Build

```bash
git clone https://github.com/yum-bundle/yum-bundle.git
cd yum-bundle
make build
```

### Run Tests

```bash
make test
```

Tests use dependency-injected mocks and do not require root or an RPM system.

### Linting

```bash
make lint
```

### Install Pre-commit Hook

```bash
make install-hooks
```

This installs a git pre-commit hook that runs `golangci-lint --fix`, `go build`, and `golangci-lint` on staged Go files.

## Project Structure

```
yum-bundle/
├── cmd/yum-bundle/       # Binary entry point
├── internal/
│   ├── commands/         # CLI commands (cobra)
│   ├── yum/              # YumManager: packages, repos, keys, copr, epel, modules
│   ├── yumfile/          # Yumfile parser
│   └── testutil/         # Mock executor for tests
├── docs/                 # Jekyll documentation site
├── Makefile
├── VERSION               # Major.minor version (e.g. "0.1")
└── .nfpm.yaml            # RPM packaging config
```

## Release Process

Releases are automated via GitHub Actions:

1. Bump `VERSION` file (e.g., `0.1` → `0.2`)
2. Commit and push to `main`
3. The release workflow auto-calculates the next patch version, builds RPM packages for all architectures, and publishes a GitHub release

## Submitting Changes

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Ensure `make test lint build` passes
5. Open a pull request

## Reporting Issues

Please report bugs and feature requests at [GitHub Issues](https://github.com/yum-bundle/yum-bundle/issues).
