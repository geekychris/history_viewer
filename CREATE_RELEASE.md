# Creating a Release

This guide explains how to create a new release that builds binaries for all supported platforms.

## Quick Release (Recommended)

Create and push a git tag:

```bash
# Create a version tag
git tag -a v1.0.0 -m "Release v1.0.0"

# Push the tag to GitHub
git push origin v1.0.0
```

This automatically triggers the release workflow.

## Manual Release Trigger

Alternatively, trigger the workflow manually:

### Using GitHub CLI

```bash
gh workflow run release.yml -f version=v1.0.0
```

### Using GitHub Web Interface

1. Go to your repository on GitHub
2. Click the **Actions** tab
3. Select **Build and Release** workflow
4. Click **Run workflow** button
5. Enter the version (e.g., `v1.0.0`)
6. Click **Run workflow**

## What Gets Built

The workflow automatically builds binaries for:

- **macOS Intel** (x86_64/amd64)
- **macOS Apple Silicon** (arm64)
- **Linux x86_64** (amd64)
- **Linux ARM64**

## What Happens During Release

1. Builds all 4 platform binaries in parallel
2. Creates a GitHub release with the version tag
3. Uploads all binaries as `.tar.gz` files
4. Generates SHA256 checksums
5. Updates Homebrew tap formula (if configured)

## Verifying the Release

Check your releases page:

```bash
# View the release
gh release view v1.0.0

# Or visit in browser
open https://github.com/YOUR_USERNAME/history_viewer/releases
```

## Testing the Release

### Test Homebrew Installation (macOS)

```bash
brew tap YOUR_USERNAME/history-viewer
brew install history-viewer
history_viewer -h
```

### Test Manual Installation

```bash
# macOS Apple Silicon
curl -LO https://github.com/YOUR_USERNAME/history_viewer/releases/download/v1.0.0/history_viewer-darwin-arm64.tar.gz
tar -xzf history_viewer-darwin-arm64.tar.gz
./history_viewer-darwin-arm64 -h

# macOS Intel
curl -LO https://github.com/YOUR_USERNAME/history_viewer/releases/download/v1.0.0/history_viewer-darwin-amd64.tar.gz
tar -xzf history_viewer-darwin-amd64.tar.gz
./history_viewer-darwin-amd64 -h

# Linux x86_64
curl -LO https://github.com/YOUR_USERNAME/history_viewer/releases/download/v1.0.0/history_viewer-linux-amd64.tar.gz
tar -xzf history_viewer-linux-amd64.tar.gz
./history_viewer-linux-amd64 -h

# Linux ARM64
curl -LO https://github.com/YOUR_USERNAME/history_viewer/releases/download/v1.0.0/history_viewer-linux-arm64.tar.gz
tar -xzf history_viewer-linux-arm64.tar.gz
./history_viewer-linux-arm64 -h
```

## Version Numbering

Follow semantic versioning:

- `v1.0.0` - First stable release
- `v1.1.0` - New features (backwards compatible)
- `v1.0.1` - Bug fixes
- `v2.0.0` - Breaking changes

## Troubleshooting

### Build Failed

Check the Actions tab for logs:

```bash
gh run list --workflow=release.yml
gh run view [run-id]
```

Common fix:

```bash
go mod tidy
git add go.mod go.sum
git commit -m "Update go modules"
git push
```

### Delete a Failed Release

```bash
gh release delete v1.0.0 --yes
git push --delete origin v1.0.0
git tag -d v1.0.0
```

## Additional Documentation

For detailed setup instructions, see:
- `RELEASE.md` - Full release process documentation
- `.github/RELEASE_QUICKSTART.md` - One-time setup guide
