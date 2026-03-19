---
layout: default
title: Installation
nav_order: 2
---

# Installation

## Quick Install (Recommended)

Install the latest release using the install script:

```bash
curl -fsSL https://raw.githubusercontent.com/yum-bundle/yum-bundle/main/install.sh | sudo bash
```

This script automatically detects your system architecture (x86_64, aarch64, armv7hl) and installs the appropriate `.rpm` package.

## Install via RPM Repository

For production environments or when you need version control, install from the official RPM repository:

```bash
# Add the repository
cat <<EOF | sudo tee /etc/yum.repos.d/yum-bundle.repo
[yum-bundle]
name=yum-bundle
baseurl=https://yum-bundle.org/repo/
enabled=1
gpgcheck=0
EOF

# Install yum-bundle
sudo dnf install -y yum-bundle
```

### Benefits

- ✅ Version control — Can pin to specific versions
- ✅ Multi-architecture — Supports x86_64, aarch64, armv7hl
- ✅ Production-ready — Standard RPM package management
- ✅ Reproducible — Same version every time

## From Source

### Prerequisites

- Go 1.23 or later
- An RPM-based Linux system (for running the tool)
- `make` (usually pre-installed)

### Build and Install

```bash
# Clone the repository
git clone https://github.com/yum-bundle/yum-bundle.git
cd yum-bundle

# Build the binary
make build

# Install to /usr/local/bin (requires sudo)
sudo make install
```

The binary will be installed to `/usr/local/bin/yum-bundle`.

### Custom Installation Directory

```bash
INSTALL_DIR=/opt/bin sudo make install
```

### Without sudo

```bash
INSTALL_DIR=$HOME/.local/bin USE_SUDO="" make install
```

## Uninstallation

```bash
sudo make uninstall
```

Or manually:

```bash
sudo rm /usr/local/bin/yum-bundle
```

## Verification

```bash
yum-bundle --version
yum-bundle --help
```

## Next Steps

- Learn how to use yum-bundle in the [Usage Guide](usage.html)
- Understand the [Yumfile Format](yumfile-format.html)
