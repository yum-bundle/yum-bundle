---
layout: default
title: Usage
nav_order: 3
---

# Usage

## Basic Commands

### Install (Default Command)

Install packages from a `Yumfile`:

```bash
# Use default Yumfile (./Yumfile)
sudo yum-bundle

# Or explicitly
sudo yum-bundle install

# Use a different Yumfile
sudo yum-bundle --file /path/to/Yumfile
```

The `install` command:
1. Imports any specified GPG keys
2. Adds any specified repositories (`repo`, `baseurl`, `copr`)
3. Enables EPEL if `epel` directive is present
4. Enables DNF modules if `module` directives are present
5. Runs `dnf makecache` (unless `--no-update` is specified)
6. Installs all `yum` packages
7. Installs any `rpm` URL packages

### Check

Check if Yumfile entries are satisfied (no root required):

```bash
apt-bundle check
apt-bundle check --file /path/to/Yumfile
apt-bundle check --json    # machine-readable output
```

### Sync

Install missing items and remove packages no longer in the Yumfile:

```bash
sudo yum-bundle sync
sudo yum-bundle sync --autoremove    # also run dnf autoremove
sudo yum-bundle sync --dry-run       # preview changes
```

### Cleanup

Remove packages that yum-bundle installed but are no longer in the Yumfile:

```bash
sudo yum-bundle cleanup              # dry-run (show what would be removed)
sudo yum-bundle cleanup --force      # actually remove packages
sudo yum-bundle cleanup --zap        # remove ALL packages not in Yumfile (dangerous)
```

### Dump

Generate a Yumfile from the system's current state:

```bash
yum-bundle dump          # print to stdout
yum-bundle dump > Yumfile
```

### Lock

Pin installed versions of Yumfile packages to `Yumfile.lock`:

```bash
yum-bundle lock
```

Install from lock file:

```bash
sudo yum-bundle install --locked
```

### Outdated

List Yumfile packages that have available upgrades:

```bash
yum-bundle outdated
```

### Doctor

Validate Yumfile syntax and check the environment:

```bash
yum-bundle doctor
yum-bundle doctor --yumfile-only    # only validate Yumfile
```

## Global Flags

- `--file, -f`: Path to Yumfile (default: `./Yumfile`)
- `--no-update`: Skip `dnf makecache` before installing
- `--version`: Show version and commit hash
- `--help, -h`: Show help

## Examples

### Basic Package Installation

```yumfile
yum vim
yum curl
yum git
```

```bash
sudo yum-bundle
```

### With EPEL and COPR

```yumfile
epel
copr atim/lazygit

yum lazygit
yum htop
```

### With Custom Repository and GPG Key

```yumfile
key https://packages.example.com/gpg.key
baseurl https://packages.example.com/el9/

yum my-package
```

### With DNF Module

```yumfile
module nodejs:18

yum nodejs
yum npm
```

### With RPM URL

```yumfile
rpm https://dl.fedoraproject.org/pub/epel/epel-release-latest-9.noarch.rpm

yum htop
```

### Dry Run

Preview what would be installed without making changes:

```bash
sudo yum-bundle install --dry-run
```
