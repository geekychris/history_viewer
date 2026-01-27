# Release Process

This document describes how to create releases for the History Viewer project using GitHub Actions.

## Overview

The GitHub Actions workflow automatically builds binaries for:
- **macOS**: Intel (x86_64) and Apple Silicon (arm64)
- **Linux**: x86_64 (amd64) and ARM64

The workflow also:
- Creates GitHub releases with downloadable binaries
- Generates SHA256 checksums for verification
- Automatically updates the Homebrew tap formula for easy macOS installation

## Prerequisites

### 1. GitHub Repository Setup

Make sure your code is pushed to a GitHub repository.

### 2. Homebrew Tap Repository (for macOS installation)

Create a separate GitHub repository for your Homebrew tap:

```bash
# Create a new repository on GitHub named: homebrew-history-viewer
# Then locally:
mkdir -p homebrew-history-viewer/Formula
cd homebrew-history-viewer
git init
git remote add origin https://github.com/geekychris/homebrew-history-viewer.git
```

Create an initial placeholder formula:

```bash
cat > Formula/history-viewer.rb << 'EOF'
class HistoryViewer < Formula
  desc "Powerful Go-based tool for analyzing zsh command history with AI-powered insights"
  homepage "https://github.com/geekychris/history_viewer"
  version "0.0.1"
  
  if Hardware::CPU.arm?
    url "https://github.com/geekychris/history_viewer/releases/download/v0.0.1/history_viewer-darwin-arm64.tar.gz"
    sha256 "placeholder"
  else
    url "https://github.com/geekychris/history_viewer/releases/download/v0.0.1/history_viewer-darwin-amd64.tar.gz"
    sha256 "placeholder"
  end

  def install
    if Hardware::CPU.arm?
      bin.install "history_viewer-darwin-arm64" => "history_viewer"
    else
      bin.install "history_viewer-darwin-amd64" => "history_viewer"
    end
  end

  test do
    assert_match "history_viewer", shell_output("#{bin}/history_viewer -h")
  end
end
EOF

git add Formula/history-viewer.rb
git commit -m "Initial formula"
git push -u origin main
```

### 3. GitHub Personal Access Token

Create a Personal Access Token (PAT) for updating the Homebrew tap:

1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Name it: "Homebrew Tap Update Token"
4. Select scopes:
   - `repo` (Full control of private repositories)
   - `workflow` (Update GitHub Action workflows)
5. Click "Generate token" and copy the token

Add the token to your main repository secrets:

1. Go to your `history_viewer` repository
2. Settings → Secrets and variables → Actions
3. Click "New repository secret"
4. Name: `HOMEBREW_TAP_TOKEN`
5. Value: Paste your PAT
6. Click "Add secret"

## Creating a Release

### Option 1: Tag-based Release (Recommended)

Create and push a git tag:

```bash
# Create a new version tag
git tag -a v1.0.0 -m "Release version 1.0.0"

# Push the tag to GitHub
git push origin v1.0.0
```

This automatically triggers the workflow and creates a release.

### Option 2: Manual Workflow Dispatch

You can also manually trigger the workflow from GitHub:

1. Go to your repository on GitHub
2. Click "Actions" tab
3. Click "Build and Release" workflow
4. Click "Run workflow" button
5. Enter the version (e.g., `v1.0.0`)
6. Click "Run workflow"

## What Happens During Release

1. **Build Phase**: Four parallel builds create binaries:
   - macOS Intel (amd64)
   - macOS Apple Silicon (arm64)
   - Linux x86_64 (amd64)
   - Linux ARM64

2. **Release Creation**: 
   - Creates a GitHub release with the version tag
   - Uploads all four binaries as `.tar.gz` files
   - Generates SHA256 checksums
   - Adds installation instructions to the release notes

3. **Homebrew Update** (automatic for tag-based releases):
   - Downloads the macOS binaries
   - Calculates SHA256 hashes
   - Updates the Homebrew formula in your tap repository
   - Commits and pushes the changes

## After Release

### Verify the Release

1. Check the GitHub releases page
2. Download and test a binary:

```bash
# macOS Apple Silicon
curl -LO https://github.com/geekychris/history_viewer/releases/download/v1.0.0/history_viewer-darwin-arm64.tar.gz
tar -xzf history_viewer-darwin-arm64.tar.gz
./history_viewer-darwin-arm64 -h
```

### Test Homebrew Installation

```bash
# Tap the repository
brew tap geekychris/history-viewer

# Install the formula
brew install history-viewer

# Run the app
history_viewer -h
```

### Update Release Notes (Optional)

You can edit the release notes on GitHub to add:
- Detailed changelog
- Breaking changes
- New features
- Bug fixes

## Versioning

Follow [Semantic Versioning](https://semver.org/):
- **MAJOR** version (v2.0.0): Incompatible API changes
- **MINOR** version (v1.1.0): Add functionality (backwards compatible)
- **PATCH** version (v1.0.1): Bug fixes (backwards compatible)

## Troubleshooting

### Build Failures

Check the Actions tab for logs. Common issues:
- **CGO errors**: Platform-specific dependencies issue
- **Go module issues**: Run `go mod tidy` and commit changes

### Homebrew Update Failures

If the Homebrew formula update fails:
- Verify `HOMEBREW_TAP_TOKEN` secret is set correctly
- Check that the token has `repo` and `workflow` permissions
- Manually update the formula if needed (see below)

### Manual Homebrew Formula Update

If automatic update fails, update manually:

```bash
cd homebrew-history-viewer

# Download both macOS binaries and calculate checksums
curl -LO https://github.com/geekychris/history_viewer/releases/download/v1.0.0/history_viewer-darwin-amd64.tar.gz
curl -LO https://github.com/geekychris/history_viewer/releases/download/v1.0.0/history_viewer-darwin-arm64.tar.gz

AMD64_SHA=$(sha256sum history_viewer-darwin-amd64.tar.gz | awk '{print $1}')
ARM64_SHA=$(sha256sum history_viewer-darwin-arm64.tar.gz | awk '{print $1}')

# Update Formula/history-viewer.rb with new version and checksums
# Then commit and push
git add Formula/history-viewer.rb
git commit -m "Update to v1.0.0"
git push
```

## CI/CD Workflow Details

### Workflow File

Location: `.github/workflows/release.yml`

### Jobs

1. **build-macos**: Builds for macOS (amd64 and arm64)
2. **build-linux**: Builds for Linux (amd64 and arm64)
3. **create-release**: Creates GitHub release with all binaries
4. **update-homebrew**: Updates Homebrew tap formula (tag-based releases only)

### Build Tags

The workflow builds with `-tags web` to create web UI binaries that don't require GUI dependencies on the host system. This makes the binaries more portable.

To build with native UI support instead, modify the workflow to remove the `-tags web` flag.

## Repository Structure

```
history_viewer/                    # Main repository
├── .github/
│   └── workflows/
│       └── release.yml           # Release workflow
├── go.mod
├── main.go
└── ...

homebrew-history-viewer/          # Homebrew tap repository
└── Formula/
    └── history-viewer.rb         # Homebrew formula
```

## Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [Go Cross Compilation](https://go.dev/doc/install/source#environment)
