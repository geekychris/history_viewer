# Complete CI/CD Setup Guide for History Viewer

This document provides a comprehensive guide to the GitHub Actions CI/CD workflow that was set up for the History Viewer project, including setup, usage, troubleshooting, and maintenance.

## Table of Contents

- [Overview](#overview)
- [What Was Created](#what-was-created)
- [Initial Setup (One-Time)](#initial-setup-one-time)
- [How It Works](#how-it-works)
- [Creating Releases](#creating-releases)
- [Installation Methods](#installation-methods)
- [Troubleshooting](#troubleshooting)
- [Maintenance](#maintenance)
- [Architecture Details](#architecture-details)

---

## Overview

The CI/CD workflow automatically builds and releases your History Viewer application for multiple platforms whenever you push a version tag to GitHub. It handles:

- **Multi-platform builds**: macOS (Intel + Apple Silicon) and Linux (x86_64)
- **GitHub releases**: Automatic release creation with downloadable binaries
- **Homebrew support**: Automatic formula updates for easy macOS installation
- **Security**: SHA256 checksums for all binaries

### What Gets Built

Each release includes three binaries:
1. `history_viewer-darwin-amd64.tar.gz` - macOS Intel (x86_64)
2. `history_viewer-darwin-arm64.tar.gz` - macOS Apple Silicon (M1/M2/M3)
3. `history_viewer-linux-amd64.tar.gz` - Linux x86_64

Plus a `SHA256SUMS.txt` file containing checksums for verification.

---

## What Was Created

### 1. Repository Files

**Main Repository (`geekychris/history_viewer`):**
```
.github/
├── workflows/
│   └── release.yml                 # Main workflow file
├── RELEASE_QUICKSTART.md           # Quick reference guide
├── SETUP_SUMMARY.md                # Setup overview
└── WORKFLOW_DIAGRAM.md             # Visual workflow diagrams

RELEASE.md                          # Detailed release documentation
CI_CD_COMPLETE_GUIDE.md            # This file
README.md                          # Updated with install instructions
```

**Homebrew Tap Repository (`geekychris/homebrew-history-viewer`):**
```
Formula/
└── history-viewer.rb              # Homebrew formula (auto-updated)
```

### 2. GitHub Configuration

- **Repository**: `geekychris/history_viewer` (main project)
- **Homebrew Tap**: `geekychris/homebrew-history-viewer` (Homebrew formulas)
- **Secret**: `HOMEBREW_TAP_TOKEN` (for automatic formula updates)

---

## Initial Setup (One-Time)

This setup has already been completed, but here's what was done for reference:

### Step 1: Create Homebrew Tap Repository

```bash
# Created public repository
gh repo create geekychris/homebrew-history-viewer --public

# Cloned and set up structure
git clone https://github.com/geekychris/homebrew-history-viewer.git
cd homebrew-history-viewer
mkdir -p Formula
```

### Step 2: Create Initial Homebrew Formula

Created `Formula/history-viewer.rb` with v1.0.0 release information.

### Step 3: GitHub Personal Access Token

Created a Personal Access Token with `repo` and `workflow` permissions and added it as a repository secret named `HOMEBREW_TAP_TOKEN`.

### Step 4: First Release

```bash
# Committed workflow files
git add .github/
git commit -m "Add CI/CD workflow"
git push

# Created first release tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

---

## How It Works

### Workflow Trigger

The workflow runs automatically when:
1. You push a tag matching `v*` pattern (e.g., `v1.0.0`, `v2.1.3`)
2. You manually trigger it from GitHub Actions UI

### Build Process

```
Tag Pushed (v1.0.0)
        ↓
┌───────────────────────────────┐
│   Parallel Build Jobs         │
│                               │
│  • macOS Intel (amd64)       │
│  • macOS ARM64               │
│  • Linux x86_64 (amd64)      │
└───────────────────────────────┘
        ↓
┌───────────────────────────────┐
│   Create GitHub Release       │
│                               │
│  • Upload all binaries        │
│  • Generate SHA256SUMS.txt    │
│  • Add install instructions   │
└───────────────────────────────┘
        ↓
┌───────────────────────────────┐
│   Update Homebrew Formula     │
│                               │
│  • Calculate checksums        │
│  • Update formula file        │
│  • Commit and push            │
└───────────────────────────────┘
```

### Build Details

**macOS Builds:**
- Run on GitHub's `macos-latest` runner
- Use Xcode command line tools (pre-installed)
- Cross-compile for both Intel and ARM64 architectures
- Build with `-tags web` flag for portability

**Linux Build:**
- Runs on GitHub's `ubuntu-latest` runner
- Installs dependencies: `gcc libgl1-mesa-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev`
- Native compilation for x86_64
- Build with `-tags web` flag for portability

**Build Time:**
- Typically completes in 2-5 minutes total
- All builds run in parallel

---

## Creating Releases

### Standard Release Process

```bash
# 1. Make your changes
git add .
git commit -m "Add new feature"
git push

# 2. Create and push a version tag
git tag -a v1.1.0 -m "Release v1.1.0 - Add new feature"
git push origin v1.1.0

# 3. Monitor the workflow
gh run watch
```

That's it! The workflow handles everything else automatically.

### Version Numbering

Follow [Semantic Versioning](https://semver.org/):

- **v1.0.0** → First stable release
- **v1.1.0** → New features (backwards compatible)
- **v1.0.1** → Bug fixes only
- **v2.0.0** → Breaking changes

### Manual Workflow Trigger

You can also trigger the workflow manually:

```bash
# Via GitHub CLI
gh workflow run release.yml -f version=v1.1.0

# Or via GitHub web interface:
# 1. Go to Actions tab
# 2. Select "Build and Release"
# 3. Click "Run workflow"
# 4. Enter version number
```

### Monitoring Releases

```bash
# Watch active workflow
gh run watch

# List recent runs
gh run list --workflow=release.yml --limit 5

# View specific run
gh run view <run-id>

# View release
gh release view v1.0.0

# List all releases
gh release list
```

---

## Installation Methods

### Method 1: Homebrew (macOS - Recommended)

```bash
# Tap the repository (one-time)
brew tap geekychris/history-viewer

# Install
brew install history-viewer

# Run
history_viewer
```

**Updating:**
```bash
brew update
brew upgrade history-viewer
```

**Uninstalling:**
```bash
brew uninstall history-viewer
brew untap geekychris/history-viewer
```

### Method 2: Pre-built Binaries

**macOS Apple Silicon (M1/M2/M3):**
```bash
curl -LO https://github.com/geekychris/history_viewer/releases/latest/download/history_viewer-darwin-arm64.tar.gz
tar -xzf history_viewer-darwin-arm64.tar.gz
sudo mv history_viewer-darwin-arm64 /usr/local/bin/history_viewer
chmod +x /usr/local/bin/history_viewer
```

**macOS Intel:**
```bash
curl -LO https://github.com/geekychris/history_viewer/releases/latest/download/history_viewer-darwin-amd64.tar.gz
tar -xzf history_viewer-darwin-amd64.tar.gz
sudo mv history_viewer-darwin-amd64 /usr/local/bin/history_viewer
chmod +x /usr/local/bin/history_viewer
```

**Linux x86_64:**
```bash
curl -LO https://github.com/geekychris/history_viewer/releases/latest/download/history_viewer-linux-amd64.tar.gz
tar -xzf history_viewer-linux-amd64.tar.gz
sudo mv history_viewer-linux-amd64 /usr/local/bin/history_viewer
chmod +x /usr/local/bin/history_viewer
```

**Verify installation:**
```bash
history_viewer -h
```

### Method 3: Build from Source

```bash
git clone https://github.com/geekychris/history_viewer.git
cd history_viewer
go build -o history_viewer
./history_viewer
```

---

## Troubleshooting

### Build Failures

**Problem: Workflow fails to start**
```bash
# Check if workflow file is valid
cat .github/workflows/release.yml

# Verify workflow exists on remote
git log --oneline .github/workflows/release.yml

# Check GitHub Actions is enabled
# Visit: https://github.com/geekychris/history_viewer/settings/actions
```

**Problem: Linux build fails with dependency errors**

The workflow installs the exact dependencies from BUILD.md:
```bash
gcc libgl1-mesa-dev libxcursor-dev libxrandr-dev libxinerama-dev libxi-dev libxxf86vm-dev
```

If the build fails, check the workflow logs and verify dependencies match BUILD.md.

**Problem: macOS build fails**

Check that the Xcode command line tools are being used correctly. The GitHub runner has them pre-installed.

### Release Failures

**Problem: "Create Release" step fails**

Common causes:
- Release with that tag already exists → Delete old release first
- Missing artifacts → Check that all build jobs completed
- Invalid tag name → Must start with 'v' (e.g., v1.0.0)

```bash
# Delete existing release and tag
gh release delete v1.0.0 --yes
git push --delete origin v1.0.0
git tag -d v1.0.0

# Create new release
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

**Problem: Checksums file upload fails**

Ensure the "Generate checksums" step created `SHA256SUMS.txt`:
```yaml
- name: Generate checksums
  run: |
    cd artifacts
    find . -name "*.tar.gz" -exec sha256sum {} \; | sed 's|\./[^/]*/||' > ../SHA256SUMS.txt
```

### Homebrew Update Failures

**Problem: "Checkout homebrew tap" fails with 404**

The homebrew-history-viewer repository doesn't exist or the token doesn't have access.

```bash
# Verify repository exists
gh repo view geekychris/homebrew-history-viewer

# Check token has correct permissions
# Go to: https://github.com/settings/tokens
# Verify token has: repo, workflow scopes
```

**Problem: Formula not updating automatically**

If the automatic update fails, update manually:

```bash
cd ~/homebrew-history-viewer

# Download latest release binaries
VERSION="v1.1.0"
curl -LO https://github.com/geekychris/history_viewer/releases/download/${VERSION}/history_viewer-darwin-amd64.tar.gz
curl -LO https://github.com/geekychris/history_viewer/releases/download/${VERSION}/history_viewer-darwin-arm64.tar.gz

# Calculate checksums
AMD64_SHA=$(shasum -a 256 history_viewer-darwin-amd64.tar.gz | awk '{print $1}')
ARM64_SHA=$(shasum -a 256 history_viewer-darwin-arm64.tar.gz | awk '{print $1}')

echo "AMD64: $AMD64_SHA"
echo "ARM64: $ARM64_SHA"

# Edit Formula/history-viewer.rb with new version and checksums
# Then commit and push
git add Formula/history-viewer.rb
git commit -m "Update to ${VERSION}"
git push
```

### Installation Issues

**Problem: Homebrew installation fails**

```bash
# Update Homebrew
brew update

# Try again
brew install geekychris/history-viewer/history-viewer

# Check for errors
brew info geekychris/history-viewer/history-viewer
```

**Problem: Binary won't run (macOS)**

macOS may block unsigned binaries:
```bash
# Remove quarantine attribute
xattr -d com.apple.quarantine /usr/local/bin/history_viewer

# Or go to System Preferences > Security & Privacy and allow it
```

**Problem: Binary won't run (Linux)**

```bash
# Ensure executable permissions
chmod +x /usr/local/bin/history_viewer

# Check if it's the right architecture
file /usr/local/bin/history_viewer

# Try running directly
/usr/local/bin/history_viewer -h
```

---

## Maintenance

### Updating the Workflow

To modify the build process, edit `.github/workflows/release.yml`:

```bash
# Edit the workflow
vim .github/workflows/release.yml

# Commit and push
git add .github/workflows/release.yml
git commit -m "Update workflow: <description>"
git push

# Test with a new tag
git tag -a v1.0.1-test -m "Test workflow changes"
git push origin v1.0.1-test
```

**Common modifications:**

1. **Change Go version:**
```yaml
- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version: '1.24'  # Update version here
```

2. **Add new platform:**
```yaml
strategy:
  matrix:
    arch: [amd64, arm64, arm]  # Add new architecture
```

3. **Modify build flags:**
```yaml
run: |
  go build -tags web -ldflags="-s -w" -o binary  # Add flags
```

### Rotating GitHub Token

If you need to rotate the `HOMEBREW_TAP_TOKEN`:

1. Create new token at: https://github.com/settings/tokens/new
2. Select scopes: `repo`, `workflow`
3. Update repository secret:
   - Go to: https://github.com/geekychris/history_viewer/settings/secrets/actions
   - Edit `HOMEBREW_TAP_TOKEN`
   - Paste new token

### Repository Maintenance

**Cleaning up old releases:**
```bash
# List releases
gh release list

# Delete old release
gh release delete v0.9.0 --yes

# Delete old tag
git push --delete origin v0.9.0
git tag -d v0.9.0
```

**Archiving old builds:**

Consider keeping only the last 5-10 releases to save storage:
- GitHub Actions artifacts expire after 90 days automatically
- Release assets remain indefinitely unless manually deleted

---

## Architecture Details

### Workflow File Structure

```yaml
name: Build and Release

on:
  push:
    tags: ['v*']
  workflow_dispatch:

permissions:
  contents: write

jobs:
  build-macos:      # Builds for macOS (Intel + ARM)
  build-linux-amd64: # Builds for Linux x86_64
  create-release:    # Creates GitHub release
  update-homebrew:   # Updates Homebrew formula
```

### Build Tags

The workflow uses `-tags web` which:
- Builds the web UI version (browser-based)
- Excludes native GUI dependencies
- Makes binaries more portable
- Reduces build complexity

To build with native UI support instead, remove `-tags web` from build commands.

### Binary Naming Convention

```
history_viewer-{os}-{arch}.tar.gz

Examples:
  history_viewer-darwin-amd64.tar.gz   (macOS Intel)
  history_viewer-darwin-arm64.tar.gz   (macOS Apple Silicon)
  history_viewer-linux-amd64.tar.gz    (Linux x86_64)
```

### Directory Structure

```
Your GitHub Repositories:

geekychris/history_viewer/
├── .github/
│   ├── workflows/
│   │   └── release.yml              # Workflow definition
│   ├── RELEASE_QUICKSTART.md        # Quick reference
│   ├── SETUP_SUMMARY.md             # Setup overview
│   └── WORKFLOW_DIAGRAM.md          # Visual diagrams
├── go.mod
├── main.go
├── README.md                        # User documentation
├── RELEASE.md                       # Release process guide
└── CI_CD_COMPLETE_GUIDE.md         # This file

geekychris/homebrew-history-viewer/
└── Formula/
    └── history-viewer.rb            # Auto-updated by workflow
```

### Security Considerations

✅ **Good Practices:**
- Repository secrets used for sensitive tokens
- Token has minimum required permissions (repo, workflow)
- SHA256 checksums provided for all binaries
- Using pinned versions of GitHub Actions (@v4, @v5)

⚠️ **Important Notes:**
- Never commit tokens or secrets to repository
- Rotate `HOMEBREW_TAP_TOKEN` periodically
- Review workflow logs for any sensitive data exposure

---

## Quick Reference

### Common Commands

```bash
# Create a release
git tag -a v1.1.0 -m "Release v1.1.0"
git push origin v1.1.0

# Watch workflow
gh run watch

# View release
gh release view v1.1.0

# List releases
gh release list

# Delete release
gh release delete v1.1.0 --yes
git push --delete origin v1.1.0
git tag -d v1.1.0

# Test Homebrew formula
brew info geekychris/history-viewer/history-viewer

# Update Homebrew
brew update
brew upgrade history-viewer
```

### Important URLs

- **Main Repository**: https://github.com/geekychris/history_viewer
- **Homebrew Tap**: https://github.com/geekychris/homebrew-history-viewer
- **Releases**: https://github.com/geekychris/history_viewer/releases
- **Actions**: https://github.com/geekychris/history_viewer/actions
- **Settings**: https://github.com/geekychris/history_viewer/settings

### Release Checklist

Before creating a release:

- [ ] All changes committed and pushed to main
- [ ] Version number decided (semantic versioning)
- [ ] README updated if needed
- [ ] Tests passing locally
- [ ] Tag created and pushed
- [ ] Workflow completes successfully
- [ ] Test at least one installation method
- [ ] Verify Homebrew formula updated
- [ ] Update release notes on GitHub (optional)

---

## Support and Resources

### Documentation Files

- **CI_CD_COMPLETE_GUIDE.md** (this file) - Complete reference
- **RELEASE_QUICKSTART.md** - Quick commands and common tasks
- **RELEASE.md** - Detailed release process
- **SETUP_SUMMARY.md** - Setup overview
- **WORKFLOW_DIAGRAM.md** - Visual workflow diagrams
- **BUILD.md** - Build instructions and dependencies

### External Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [Go Cross Compilation](https://go.dev/doc/install/source#environment)
- [Semantic Versioning](https://semver.org/)

### Getting Help

If you encounter issues:

1. Check the workflow logs in GitHub Actions tab
2. Review this documentation
3. Check specific guides (RELEASE.md, BUILD.md)
4. Verify GitHub Actions is enabled for the repository
5. Check repository secrets are configured correctly

---

## Appendix: Workflow YAML

For reference, here's the complete workflow file:

**Location**: `.github/workflows/release.yml`

The workflow consists of four main jobs:
1. `build-macos` - Builds for macOS (Intel and ARM64)
2. `build-linux-amd64` - Builds for Linux x86_64
3. `create-release` - Creates GitHub release with all artifacts
4. `update-homebrew` - Updates the Homebrew formula automatically

See the actual file for the complete YAML configuration.

---

## Version History

- **v1.0.0** (2026-01-27) - Initial release
  - macOS Intel and Apple Silicon support
  - Linux x86_64 support
  - Homebrew installation support
  - Automatic release creation
  - SHA256 checksum generation

---

## License

Same as the main History Viewer project (MIT License).

---

**Last Updated**: 2026-01-27  
**Repository**: https://github.com/geekychris/history_viewer
