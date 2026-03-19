---
layout: default
title: Yumfile Format
nav_order: 4
---

# Yumfile Format

A `Yumfile` is a line-oriented text file. Blank lines and lines starting with `#` are ignored. Each non-comment line is a directive followed by an argument (except `epel` which is bare).

## Directives

### `yum` — Install a Package

```
yum <package-name>
yum <package-name>=<version>
yum <package-name> = <version>
```

Installs a package via `dnf install -y` (or `yum install -y`). The package is tracked in yum-bundle's state file for `cleanup` and `sync`.

**Examples:**
```
yum vim
yum curl
yum nodejs = 18.0.0
yum python3-pip
```

### `key` — Import a GPG Key

```
key <https-url>
```

Downloads a GPG key and imports it via `rpm --import`. Only `https://` URLs are accepted.

**Examples:**
```
key https://packages.microsoft.com/keys/microsoft.asc
key https://packages.example.com/gpg.key
```

### `repo` — Add a Repository File

```
repo <https-url>
```

Downloads a `.repo` file from a URL and places it in `/etc/yum.repos.d/`. Only `https://` URLs are accepted.

**Examples:**
```
repo https://packages.microsoft.com/config/rhel/9/prod.repo
```

### `baseurl` — Add a Repository by Base URL

```
baseurl <https-url>
```

Creates a minimal `.repo` file in `/etc/yum.repos.d/` with the given base URL. Useful when you have a custom repository without a pre-built `.repo` file.

**Examples:**
```
baseurl https://packages.example.com/el9/x86_64/
```

### `copr` — Enable a COPR Repository

```
copr <user>/<project>
```

Enables a [Fedora COPR](https://copr.fedorainfracloud.org/) community repository using `dnf copr enable`. Requires `dnf-plugins-core` to be installed. Operation is idempotent — if the `.repo` file already exists, the command is skipped.

**Examples:**
```
copr atim/lazygit
copr carlwgeorge/ripgrep
```

### `epel` — Enable EPEL

```
epel
```

Enables [EPEL](https://docs.fedoraproject.org/en-US/epel/) (Extra Packages for Enterprise Linux) via `dnf install -y epel-release`. Automatically skipped on Fedora (where EPEL is not needed). Idempotent — skipped if `epel.repo` already exists.

**Example:**
```
epel
```

### `module` — Enable a DNF Module

```
module <name>:<stream>
```

Enables a [DNF module](https://docs.fedoraproject.org/en-US/modularity/) stream using `dnf module enable`. Requires DNF (RHEL 8+/Fedora). The `name:stream` format is required. Idempotent — skipped if the module stream is already enabled.

**Examples:**
```
module nodejs:18
module php:8.1
module postgresql:15
```

### `rpm` — Install an RPM from a URL

```
rpm <https-url>
```

Installs an RPM package directly from an HTTPS URL using `dnf install -y`. Only `https://` URLs are accepted and the URL must end with `.rpm`. Useful for bootstrapping repository-setup packages (e.g., `epel-release`, `rpmfusion-*-release`).

**Examples:**
```
rpm https://dl.fedoraproject.org/pub/epel/epel-release-latest-9.noarch.rpm
rpm https://mirrors.rpmfusion.org/free/el/rpmfusion-free-release-9.noarch.rpm
```

## Comments

```
# This is a comment
yum vim  # inline comments are not supported — entire line must start with #
```

## Complete Example

```
# System tools
yum vim
yum curl
yum git
yum htop

# Enable EPEL for extra packages
epel

# COPR repositories
copr atim/lazygit

# Custom repository with GPG key
key https://packages.example.com/gpg.key
baseurl https://packages.example.com/el9/

# DNF module for Node.js
module nodejs:18

# Packages from custom repo
yum nodejs
yum my-custom-package

# Bootstrap RPM
rpm https://packages.example.com/bootstrap-latest.noarch.rpm
```

## Directive Ordering

Directives are processed in order. The recommended order is:

1. `key` — Import GPG keys first
2. `repo` / `baseurl` / `copr` — Add repositories
3. `epel` — Enable EPEL
4. `module` — Enable DNF modules
5. `rpm` — Install bootstrap RPMs
6. `yum` — Install packages

## Security

All URLs (keys, repo files, baseurls, RPM installs) must use `https://`. Plain `http://` and `file://` URLs are rejected for security reasons.
