# CI/CD Setup Summary

This document summarizes the GitHub Actions CI/CD setup for the History Viewer project.

## What Was Created

### 1. GitHub Actions Workflow
**File:** `.github/workflows/release.yml`

A comprehensive workflow that:
- ✅ Builds for **macOS** (Intel x86_64 and Apple Silicon arm64)
- ✅ Builds for **Linux** (x86_64 and ARM64)
- ✅ Creates GitHub releases with downloadable binaries
- ✅ Generates SHA256 checksums for verification
- ✅ Automatically updates Homebrew tap formula

### 2. Documentation Files
- **RELEASE.md** - Comprehensive release process documentation
- **.github/RELEASE_QUICKSTART.md** - Quick reference guide for releases
- **.github/SETUP_SUMMARY.md** - This file

### 3. Updated README.md
Added installation options including:
- Homebrew installation (macOS)
- Pre-built binary downloads (all platforms)
- Build from source instructions

## How It Works

### Workflow Trigger
The workflow runs when:
1. **Tag pushed:** `git push origin v1.0.0`
2. **Manual trigger:** Via GitHub Actions UI

### Build Matrix
Four parallel build jobs:
```
macOS Intel (darwin-amd64)     Linux x86_64 (linux-amd64)
macOS ARM64 (darwin-arm64)     Linux ARM64 (linux-arm64)
```

### Artifacts Generated
For each platform:
- Compiled binary
- Tarball (`.tar.gz`)
- SHA256 checksum

### Homebrew Integration
When a tag is pushed:
1. Workflow downloads macOS binaries
2. Calculates SHA256 hashes
3. Updates formula in `homebrew-history-viewer` repository
4. Commits and pushes automatically

## Next Steps

### 1. Set Up Homebrew Tap (First Time Only)

Create the Homebrew tap repository on GitHub:
```bash
# Create repository: homebrew-history-viewer
gh repo create geekychris/homebrew-history-viewer --public

# Clone and set up
git clone https://github.com/geekychris/homebrew-history-viewer.git
cd homebrew-history-viewer
mkdir -p Formula

# Create initial formula (placeholder)
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

# Commit and push
git add Formula/history-viewer.rb
git commit -m "Initial formula"
git push -u origin main
```

### 2. Configure GitHub Token

Create a Personal Access Token:
1. Go to: https://github.com/settings/tokens/new
2. Name: "Homebrew Tap Update"
3. Scopes: `repo`, `workflow`
4. Generate and copy token

Add to repository secrets:
1. Go to: https://github.com/geekychris/history_viewer/settings/secrets/actions
2. New secret: `HOMEBREW_TAP_TOKEN`
3. Paste token value

### 3. Create Your First Release

```bash
# Ensure all changes are committed
git add .
git commit -m "Add CI/CD workflow"
git push

# Create and push a tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# Watch the workflow
gh run watch
```

### 4. Verify the Release

Check that everything worked:
```bash
# View release
gh release view v1.0.0

# Test Homebrew installation (after workflow completes)
brew tap geekychris/history-viewer
brew install history-viewer
history_viewer -h
```

## Architecture Details

### Build Process

**macOS Builds:**
- Runs on `macos-latest` GitHub runner
- Uses Xcode command line tools (pre-installed)
- Cross-compiles for both Intel and ARM64
- Uses CGO flags for ARM64 cross-compilation

**Linux Builds:**
- Runs on `ubuntu-latest` GitHub runner
- Installs X11 development libraries
- For ARM64: Uses `gcc-aarch64-linux-gnu` cross-compiler
- For x86_64: Native compilation

### Binary Naming Convention
```
history_viewer-{os}-{arch}
  darwin-amd64   (macOS Intel)
  darwin-arm64   (macOS Apple Silicon)
  linux-amd64    (Linux x86_64)
  linux-arm64    (Linux ARM64)
```

### Release Assets
Each release includes:
- 4 tarball files (`.tar.gz`)
- 4 checksum files
- Auto-generated release notes with installation instructions

## Maintenance

### Updating the Workflow

Edit `.github/workflows/release.yml` to:
- Change Go version
- Modify build flags
- Update dependencies
- Add new platforms

### Manually Updating Homebrew Formula

If automatic update fails:
```bash
cd homebrew-history-viewer

# Download latest release binaries
VERSION="v1.0.0"
curl -LO https://github.com/geekychris/history_viewer/releases/download/${VERSION}/history_viewer-darwin-amd64.tar.gz
curl -LO https://github.com/geekychris/history_viewer/releases/download/${VERSION}/history_viewer-darwin-arm64.tar.gz

# Calculate checksums
AMD64_SHA=$(shasum -a 256 history_viewer-darwin-amd64.tar.gz | awk '{print $1}')
ARM64_SHA=$(shasum -a 256 history_viewer-darwin-arm64.tar.gz | awk '{print $1}')

echo "AMD64: $AMD64_SHA"
echo "ARM64: $ARM64_SHA"

# Update Formula/history-viewer.rb with new version and checksums
# Then commit and push
git add Formula/history-viewer.rb
git commit -m "Update to ${VERSION}"
git push
```

## Best Practices

### Version Numbers
Follow semantic versioning (semver):
- **MAJOR.MINOR.PATCH** (e.g., v1.2.3)
- Increment MAJOR for breaking changes
- Increment MINOR for new features
- Increment PATCH for bug fixes

### Release Checklist
- [ ] All changes committed and pushed
- [ ] Version number decided
- [ ] README updated (if needed)
- [ ] Create and push git tag
- [ ] Monitor workflow completion
- [ ] Test one platform manually
- [ ] Test Homebrew installation
- [ ] Edit release notes (optional)

### Security
- ✅ Use repository secrets for sensitive tokens
- ✅ Limit token permissions to minimum required
- ✅ Use `@v4` or `@v5` for GitHub Actions (pinned versions)
- ✅ Review workflow logs for sensitive data exposure

## Troubleshooting

### Build Fails with CGO Error
- Check platform dependencies are installed
- Verify cross-compilation flags are correct
- Look for missing C libraries

### Homebrew Update Fails
- Verify `HOMEBREW_TAP_TOKEN` is set correctly
- Check token has required permissions
- Ensure tap repository exists and is accessible

### Release Assets Missing
- Check workflow completed successfully
- Look for upload errors in workflow logs
- Verify artifact names match expected patterns

### Test Installation Fails
- Check binary has execute permissions
- Verify architecture matches your machine
- Try running with `./binary -h` to see errors

## Resources

- [GitHub Actions Docs](https://docs.github.com/en/actions)
- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [Go Cross Compilation](https://go.dev/doc/install/source#environment)
- [Semantic Versioning](https://semver.org/)

## Support

For issues or questions:
1. Check workflow logs in GitHub Actions tab
2. Review this documentation
3. Check [RELEASE.md](../RELEASE.md) for detailed instructions
4. Check [RELEASE_QUICKSTART.md](RELEASE_QUICKSTART.md) for quick commands
