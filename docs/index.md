---
layout: default
title: Home
nav_order: 1
---

# yum-bundle

A declarative package manager for yum/dnf, inspired by Homebrew's `brew bundle`.

## Overview

`yum-bundle` provides a simple, declarative, and shareable way to manage yum/dnf packages and repositories on RPM-based systems (RHEL, CentOS, Fedora, Rocky Linux, AlmaLinux). Define your system dependencies in a `Yumfile` and install them with a single command.

## Features

- **Declarative Package Management**: Define packages and repos in a simple text file
- **Idempotent Operations**: Safe to run multiple times — already-installed items are skipped
- **Full dnf/yum Ecosystem Support**: Packages, GPG keys, `.repo` files, baseurls, COPR repos, EPEL, DNF modules, and RPM URL installs
- **Version Pinning**: Install specific package versions via `Yumfile.lock`
- **Simple CLI**: `install`, `check`, `cleanup`, `sync`, `dump`, `lock`, `outdated`, `doctor`
- **GitHub Actions Integration**: Native action with caching for CI/CD

## Quick Start

```bash
# Install yum-bundle
curl -fsSL https://raw.githubusercontent.com/yum-bundle/yum-bundle/main/install.sh | sudo bash

# Create a Yumfile
cat > Yumfile <<EOF
yum vim
yum curl
yum git
EOF

# Install packages
sudo yum-bundle
```

## Use Cases

### Developer Onboarding
A new developer joins a project, clones the repo, and runs `sudo yum-bundle` to get all required system dependencies.

### Container Image Build
Replace long `RUN dnf install -y ...` lines with a `Yumfile` and a single `yum-bundle` command.

### System Sync
Use `yum-bundle dump > Yumfile` on your primary workstation and then `sudo yum-bundle` on a new server to reproduce the setup.

### CI/CD
Use the [GitHub Action](github-actions.html) for seamless integration with GitHub workflows, including built-in package caching and reproducible builds via lockfiles.

## Documentation

- [Installation](installation.html) — How to install yum-bundle
- [Usage](usage.html) — Command reference and examples
- [Yumfile Format](yumfile-format.html) — Complete syntax reference
- [GitHub Actions](github-actions.html) — Using yum-bundle in GitHub workflows
- [Contributing](contributing.html) — Contributing and development setup
