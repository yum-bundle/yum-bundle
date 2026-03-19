# yum-bundle RPM Repository

This directory contains the yum-bundle RPM repository, served at `https://yum-bundle.org/repo/`.

## Usage

Add the repository to your system:

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

## Structure

```
repo/
├── packages/                     # RPM package files
│   ├── yum-bundle_X.Y.Z_linux_x86_64.rpm
│   ├── yum-bundle_X.Y.Z_linux_aarch64.rpm
│   └── yum-bundle_X.Y.Z_linux_armv7hl.rpm
└── repodata/                     # createrepo metadata (auto-generated)
    ├── repomd.xml
    └── ...
```

## Update Process

The repository is automatically updated by the `update-rpm-repo` GitHub Actions workflow when a new release is published. The workflow:

1. Downloads the new `.rpm` packages from the GitHub release
2. Places them in `docs/repo/packages/`
3. Runs `createrepo_c --update .` to regenerate metadata
4. Commits and pushes the changes

## Manual Update

To manually update the repository:

```bash
cd docs/repo
createrepo_c --update .
```
