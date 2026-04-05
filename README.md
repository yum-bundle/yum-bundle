<p align="center">
  <img src="docs/icon-512.png" width="96" height="96" alt="yum-bundle">
</p>

<h1 align="center">yum-bundle</h1>

<p align="center">
  <a href="https://github.com/yum-bundle/yum-bundle/actions/workflows/ci.yml"><img src="https://github.com/yum-bundle/yum-bundle/actions/workflows/ci.yml/badge.svg?branch=main" alt="CI"></a>
  <a href="https://opensource.org/licenses/Apache-2.0"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License"></a>
</p>

A declarative, Brewfile-like wrapper for `yum`/`dnf`, inspired by `brew bundle` — not a full config management system.

**[📚 Full Documentation](https://yum-bundle.org)** | [Installation](#installation) | [Usage](#usage)

## Overview

`yum-bundle` provides a simple, declarative, and shareable way to manage yum/dnf packages and repositories on RPM-based systems (RHEL, CentOS, Fedora, Rocky Linux, AlmaLinux). Define your system dependencies in a `Yumfile` and install them with a single command.

## Features

- 📦 **Declarative Package Management**: Define packages in a simple `Yumfile`
- 🔄 **Idempotent Operations**: Safe to run multiple times — already-installed items are skipped
- 🔀 **Sync**: Make the system match the Yumfile in one command (install + cleanup)
- 🔑 **Full dnf/yum Ecosystem**: GPG keys, `.repo` files, baseurls, COPR repos, EPEL, DNF modules, RPM URL installs
- 📝 **Version Pinning**: Pin versions via `Yumfile.lock`
- 🚀 **Simple CLI**: `install`, `check`, `cleanup`, `sync`, `dump`, `lock`, `outdated`, `doctor`

## Why yum-bundle?

**Why not just bash scripts?** Idempotency is hard to get right; repository and key management is error-prone; and scripts become unmaintainable as they grow. yum-bundle gives you a single declarative file and predictable behavior every time.

| vs | yum-bundle advantage |
|----|----------------------|
| `rpm -qa` in scripts | Human-readable Yumfile, handles repos/keys/COPR/EPEL/modules, partial adoption |
| Ansible / Chef | Zero learning curve, no YAML or DSL — just packages and directives |
| Nix | Works with your existing dnf/rpm ecosystem; no paradigm shift |

## Installation

### Quick Install (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/yum-bundle/yum-bundle/main/install.sh | sudo bash
```

### Install via RPM Repository

```bash
cat <<EOF | sudo tee /etc/yum.repos.d/yum-bundle.repo
[yum-bundle]
name=yum-bundle
baseurl=https://yum-bundle.org/repo/
enabled=1
gpgcheck=0
EOF

sudo dnf install -y yum-bundle
```

### From Source

```bash
git clone https://github.com/yum-bundle/yum-bundle.git
cd yum-bundle
make build
sudo make install
```

## Usage

### Quick Start

```bash
# Create a Yumfile
cat > Yumfile <<EOF
yum vim
yum curl
yum git
EOF

# Install packages
sudo yum-bundle
```

### All Commands

```bash
sudo yum-bundle install            # Install from Yumfile (default command)
sudo yum-bundle install --locked   # Install from Yumfile.lock (reproducible)
sudo yum-bundle install --dry-run  # Preview changes

yum-bundle check                   # Check if all entries are satisfied
yum-bundle check --json            # Machine-readable output

sudo yum-bundle sync               # Install + cleanup in one step
sudo yum-bundle cleanup --force    # Remove packages no longer in Yumfile

yum-bundle dump                    # Generate Yumfile from current system
yum-bundle lock                    # Write Yumfile.lock with current versions
yum-bundle outdated                # List packages with available upgrades
yum-bundle doctor                  # Validate Yumfile and check environment
```

### Global Flags

```
--file, -f    Path to Yumfile (default: ./Yumfile)
--no-update   Skip dnf makecache before installing
--version     Show version and commit hash
```

## Yumfile Format

```
# Comments start with #

# Install packages
yum <package-name>
yum <package-name> = <version>

# Import a GPG key (https only)
key <url>

# Add a .repo file from URL (https only)
repo <url>

# Add a repository by baseurl (https only)
baseurl <url>

# Enable a COPR repository (user/project format)
copr <user>/<project>

# Enable EPEL (skipped on Fedora)
epel

# Enable a DNF module stream
module <name>:<stream>

# Install an RPM from a URL (https, .rpm suffix required)
rpm <url>
```

### Complete Example

```
# Enable EPEL for extra packages
epel

# COPR repository
copr atim/lazygit

# Custom repository
key https://packages.example.com/gpg.key
baseurl https://packages.example.com/el9/

# DNF module
module nodejs:18

# Bootstrap RPM
rpm https://dl.fedoraproject.org/pub/epel/epel-release-latest-9.noarch.rpm

# Packages
yum vim
yum curl
yum git
yum htop
yum nodejs
yum lazygit
```

## GitHub Actions

```yaml
- uses: yum-bundle/yum-bundle-action@v1
```

See [GitHub Actions documentation](https://yum-bundle.org/github-actions.html) for all options.

## Examples

See the [`examples/`](examples/) directory for common use cases:

1. [Basic packages](examples/1-basic/Yumfile)
2. [EPEL and COPR](examples/2-with-epel-copr/Yumfile)
3. [DNF module streams](examples/3-with-module/Yumfile)
4. [Docker CE setup](examples/4-docker-setup/Yumfile)
5. [With lockfile](examples/5-with-lockfile/Yumfile)

## License

Apache 2.0. See [LICENSE](LICENSE).
