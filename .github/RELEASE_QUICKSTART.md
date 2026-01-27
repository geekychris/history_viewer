# Release Quick Start Guide

## One-Time Setup

### 1. Create Homebrew Tap Repository

```bash
# On GitHub, create a new repository: homebrew-history-viewer
# Then:
gh repo create geekychris/homebrew-history-viewer --public
git clone https://github.com/geekychris/homebrew-history-viewer.git
cd homebrew-history-viewer
mkdir -p Formula
```

### 2. Create Initial Formula

```bash
cat > Formula/history-viewer.rb << 'EOF'
class HistoryViewer < Formula
  desc "Powerful Go-based tool for analyzing zsh command history with AI-powered insights"
  homepage "https://github.com/geekychris/history_viewer"
  version "0.0.1"
  
  url "https://github.com/geekychris/history_viewer/releases/download/v0.0.1/history_viewer-darwin-arm64.tar.gz"
  sha256 "placeholder"

  def install
    bin.install "history_viewer-darwin-arm64" => "history_viewer"
  end

  test do
    system "#{bin}/history_viewer", "-h"
  end
end
EOF

git add Formula/history-viewer.rb
git commit -m "Initial formula"
git push -u origin main
```

### 3. Create GitHub Personal Access Token

```bash
# 1. Go to: https://github.com/settings/tokens/new
# 2. Name: "Homebrew Tap Update"
# 3. Select scopes: repo, workflow
# 4. Generate token and copy it
```

### 4. Add Token to Repository Secrets

```bash
# Go to: https://github.com/geekychris/history_viewer/settings/secrets/actions/new
# Name: HOMEBREW_TAP_TOKEN
# Value: [paste your token]
```

## Creating a Release

### Simple Method (Recommended)

```bash
# 1. Ensure all changes are committed
git add .
git commit -m "Prepare for release"
git push

# 2. Create and push a tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# 3. Watch the workflow
# Go to: https://github.com/geekychris/history_viewer/actions
```

That's it! The workflow will:
- ✅ Build for macOS (Intel + Apple Silicon)
- ✅ Build for Linux (x86_64 + ARM64)
- ✅ Create a GitHub release
- ✅ Update Homebrew formula automatically

### Manual Trigger Method

```bash
# Use GitHub CLI
gh workflow run release.yml -f version=v1.0.0

# Or via GitHub web interface:
# 1. Go to Actions tab
# 2. Select "Build and Release"
# 3. Click "Run workflow"
# 4. Enter version: v1.0.0
```

## Verifying the Release

### Check GitHub Release

```bash
# Visit: https://github.com/geekychris/history_viewer/releases
# Or use GitHub CLI:
gh release view v1.0.0
```

### Test Homebrew Installation

```bash
# Tap and install
brew tap geekychris/history-viewer
brew install history-viewer

# Verify
history_viewer -h
```

### Test Manual Installation

```bash
# macOS Apple Silicon
curl -LO https://github.com/geekychris/history_viewer/releases/download/v1.0.0/history_viewer-darwin-arm64.tar.gz
tar -xzf history_viewer-darwin-arm64.tar.gz
./history_viewer-darwin-arm64 -h

# macOS Intel
curl -LO https://github.com/geekychris/history_viewer/releases/download/v1.0.0/history_viewer-darwin-amd64.tar.gz
tar -xzf history_viewer-darwin-amd64.tar.gz
./history_viewer-darwin-amd64 -h

# Linux x86_64
curl -LO https://github.com/geekychris/history_viewer/releases/download/v1.0.0/history_viewer-linux-amd64.tar.gz
tar -xzf history_viewer-linux-amd64.tar.gz
./history_viewer-linux-amd64 -h

# Linux ARM64
curl -LO https://github.com/geekychris/history_viewer/releases/download/v1.0.0/history_viewer-linux-arm64.tar.gz
tar -xzf history_viewer-linux-arm64.tar.gz
./history_viewer-linux-arm64 -h
```

## Common Issues

### Build Failed?

```bash
# Check the Actions logs
gh run list --workflow=release.yml
gh run view [run-id]

# Common fixes:
go mod tidy
git add go.mod go.sum
git commit -m "Update go modules"
git push
```

### Homebrew Update Failed?

```bash
# Manually update the formula
cd homebrew-history-viewer

# Download binaries
curl -LO https://github.com/geekychris/history_viewer/releases/download/v1.0.0/history_viewer-darwin-amd64.tar.gz
curl -LO https://github.com/geekychris/history_viewer/releases/download/v1.0.0/history_viewer-darwin-arm64.tar.gz

# Calculate checksums
shasum -a 256 history_viewer-darwin-amd64.tar.gz
shasum -a 256 history_viewer-darwin-arm64.tar.gz

# Update Formula/history-viewer.rb with new version and checksums
# Then:
git add Formula/history-viewer.rb
git commit -m "Update to v1.0.0"
git push
```

### Need to Delete a Release?

```bash
# Delete release and tag
gh release delete v1.0.0 --yes
git push --delete origin v1.0.0
git tag -d v1.0.0
```

## Version Numbering

Follow semantic versioning (semver):

- **v1.0.0** → First stable release
- **v1.1.0** → New features (backwards compatible)
- **v1.0.1** → Bug fixes
- **v2.0.0** → Breaking changes

## Quick Commands

```bash
# List all releases
gh release list

# View a specific release
gh release view v1.0.0

# Download release assets
gh release download v1.0.0

# List workflow runs
gh run list --workflow=release.yml

# Watch a workflow run
gh run watch

# Check Homebrew formula
brew info geekychris/history-viewer/history-viewer

# Update Homebrew formula
brew update
brew upgrade history-viewer

# Uninstall
brew uninstall history-viewer
brew untap geekychris/history-viewer
```

## Checklist for New Release

- [ ] All changes committed and pushed
- [ ] Version number decided (e.g., v1.0.0)
- [ ] CHANGELOG updated (if applicable)
- [ ] README updated (if needed)
- [ ] Create git tag: `git tag -a vX.Y.Z -m "Release vX.Y.Z"`
- [ ] Push tag: `git push origin vX.Y.Z`
- [ ] Monitor workflow: Actions tab on GitHub
- [ ] Verify release created successfully
- [ ] Test Homebrew installation
- [ ] Test manual installation (at least one platform)
- [ ] Update release notes on GitHub (optional)
- [ ] Announce release (optional)
