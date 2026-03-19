---
layout: default
title: GitHub Actions
nav_order: 5
---

# GitHub Actions

Use the official [yum-bundle-action](https://github.com/yum-bundle/yum-bundle-action) to install packages from a Yumfile in your GitHub CI/CD workflows.

## Quick Start

```yaml
- uses: yum-bundle/yum-bundle-action@v1
```

This reads `./Yumfile` and installs all packages.

## Inputs

| Input | Description | Default |
|-------|-------------|---------|
| `file` | Path to Yumfile | `Yumfile` |
| `version` | yum-bundle version to use | `latest` |
| `cache` | Cache installed packages | `true` |

## Examples

### Basic Usage

```yaml
name: CI

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest  # or a self-hosted RHEL/Fedora runner
    steps:
      - uses: actions/checkout@v4

      - uses: yum-bundle/yum-bundle-action@v1

      - name: Build
        run: make build
```

### Custom Yumfile Path

```yaml
- uses: yum-bundle/yum-bundle-action@v1
  with:
    file: ci/Yumfile
```

### Pin to Specific Version

```yaml
- uses: yum-bundle/yum-bundle-action@v1
  with:
    version: "0.1.0"
```

### Disable Caching

```yaml
- uses: yum-bundle/yum-bundle-action@v1
  with:
    cache: "false"
```

## Self-Hosted Runners

yum-bundle works best with self-hosted runners running RPM-based Linux (RHEL, CentOS Stream, Fedora, Rocky Linux, AlmaLinux). GitHub's hosted `ubuntu-latest` runner can run yum-bundle tests but cannot actually install RPM packages.

```yaml
jobs:
  build:
    runs-on: [self-hosted, linux, rhel]
    steps:
      - uses: actions/checkout@v4
      - uses: yum-bundle/yum-bundle-action@v1
```
