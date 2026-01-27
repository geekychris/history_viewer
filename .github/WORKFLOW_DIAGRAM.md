# GitHub Actions Workflow Diagram

## Workflow Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        TRIGGER EVENT                             │
│                                                                  │
│  Option 1: Push Tag        Option 2: Manual Trigger            │
│  git push origin v1.0.0    GitHub Actions UI                   │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                     PARALLEL BUILD JOBS                          │
│                                                                  │
│  ┌────────────────┐  ┌────────────────┐                        │
│  │  macOS Builder │  │  Linux Builder │                        │
│  │                │  │                │                        │
│  │  • Intel       │  │  • x86_64      │                        │
│  │  • ARM64       │  │  • ARM64       │                        │
│  └────────────────┘  └────────────────┘                        │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                      BUILD ARTIFACTS                             │
│                                                                  │
│  • history_viewer-darwin-amd64.tar.gz                           │
│  • history_viewer-darwin-arm64.tar.gz                           │
│  • history_viewer-linux-amd64.tar.gz                            │
│  • history_viewer-linux-arm64.tar.gz                            │
│  • SHA256 checksums for each                                    │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│                    CREATE GITHUB RELEASE                         │
│                                                                  │
│  • Upload all 4 binaries                                        │
│  • Upload checksums                                             │
│  • Generate release notes with install instructions             │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│               UPDATE HOMEBREW TAP (Tag Only)                     │
│                                                                  │
│  • Download macOS binaries                                      │
│  • Calculate SHA256 hashes                                      │
│  • Update Formula/history-viewer.rb                             │
│  • Commit and push to homebrew-history-viewer repo              │
└─────────────────────────────────────────────────────────────────┘
```

## Detailed Build Process

### macOS Build Job

```
┌──────────────────────────────────────────────────┐
│          macOS Runner (macos-latest)             │
├──────────────────────────────────────────────────┤
│  1. Checkout code                                │
│  2. Setup Go 1.23                                │
│  3. Download Go modules                          │
│  4. Build for amd64 (native)                     │
│     GOOS=darwin GOARCH=amd64 CGO_ENABLED=1       │
│     go build -tags web -o binary                 │
│  5. Build for arm64 (cross-compile)              │
│     GOOS=darwin GOARCH=arm64 CGO_ENABLED=1       │
│     CGO_CFLAGS="-target arm64-apple-macos11"     │
│     go build -tags web -o binary                 │
│  6. Create tarballs                              │
│  7. Upload artifacts                             │
└──────────────────────────────────────────────────┘
```

### Linux Build Job

```
┌──────────────────────────────────────────────────┐
│         Linux Runner (ubuntu-latest)             │
├──────────────────────────────────────────────────┤
│  1. Checkout code                                │
│  2. Setup Go 1.23                                │
│  3. Install dependencies                         │
│     amd64: native X11 libs                       │
│     arm64: cross-compiler + X11 libs             │
│  4. Download Go modules                          │
│  5. Build for amd64 (native)                     │
│     GOOS=linux GOARCH=amd64 CGO_ENABLED=1        │
│     go build -tags web -o binary                 │
│  6. Build for arm64 (cross-compile)              │
│     GOOS=linux GOARCH=arm64 CGO_ENABLED=1        │
│     CC=aarch64-linux-gnu-gcc                     │
│     go build -tags web -o binary                 │
│  7. Create tarballs                              │
│  8. Upload artifacts                             │
└──────────────────────────────────────────────────┘
```

## Release Creation Flow

```
┌─────────────────────────────────────────────────────┐
│          Download All Build Artifacts               │
└─────────────────────────────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│           Generate SHA256 Checksums                 │
└─────────────────────────────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│        Create GitHub Release with Tag               │
│                                                     │
│  Tag: v1.0.0                                       │
│  Name: Release v1.0.0                              │
│  Body: Installation instructions                   │
│  Files: 4 tarballs + 4 checksum files              │
└─────────────────────────────────────────────────────┘
```

## Homebrew Update Flow

```
┌─────────────────────────────────────────────────────┐
│      Wait 30s for Release Assets to Propagate       │
└─────────────────────────────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│      Download macOS Binaries from Release           │
│  • history_viewer-darwin-amd64.tar.gz              │
│  • history_viewer-darwin-arm64.tar.gz              │
└─────────────────────────────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│         Calculate SHA256 for Both Files             │
│  AMD64_SHA=$(sha256sum ...)                        │
│  ARM64_SHA=$(sha256sum ...)                        │
└─────────────────────────────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│   Checkout homebrew-history-viewer Repository       │
│   (using HOMEBREW_TAP_TOKEN secret)                │
└─────────────────────────────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│    Generate Updated Formula/history-viewer.rb       │
│                                                     │
│  class HistoryViewer < Formula                     │
│    version "1.0.0"                                 │
│    if Hardware::CPU.arm?                           │
│      url "...arm64.tar.gz"                         │
│      sha256 "<ARM64_SHA>"                          │
│    else                                            │
│      url "...amd64.tar.gz"                         │
│      sha256 "<AMD64_SHA>"                          │
│    end                                             │
│  end                                               │
└─────────────────────────────────────────────────────┘
                       ↓
┌─────────────────────────────────────────────────────┐
│          Commit and Push to Tap Repository          │
│  git commit -m "Update history-viewer to v1.0.0"   │
│  git push                                          │
└─────────────────────────────────────────────────────┘
```

## User Installation Flow

### Option 1: Homebrew (macOS)

```
User runs:
  brew tap geekychris/history-viewer
  brew install history-viewer
        ↓
Homebrew reads Formula/history-viewer.rb
        ↓
Detects CPU architecture (arm64 or amd64)
        ↓
Downloads appropriate tarball from GitHub Release
        ↓
Verifies SHA256 checksum
        ↓
Extracts binary to /usr/local/bin (or /opt/homebrew/bin)
        ↓
User runs: history_viewer
```

### Option 2: Manual Download

```
User downloads:
  curl -LO https://github.com/.../history_viewer-darwin-arm64.tar.gz
        ↓
Extract tarball:
  tar -xzf history_viewer-darwin-arm64.tar.gz
        ↓
Move to PATH:
  sudo mv history_viewer-darwin-arm64 /usr/local/bin/history_viewer
        ↓
Set permissions:
  chmod +x /usr/local/bin/history_viewer
        ↓
User runs: history_viewer
```

## Repository Structure

```
Your Repositories:
├── history_viewer (main project)
│   ├── .github/
│   │   ├── workflows/
│   │   │   └── release.yml          ← Workflow definition
│   │   ├── RELEASE_QUICKSTART.md    ← Quick reference
│   │   ├── SETUP_SUMMARY.md         ← Setup guide
│   │   └── WORKFLOW_DIAGRAM.md      ← This file
│   ├── go.mod
│   ├── main.go
│   ├── README.md                     ← Updated with install options
│   └── RELEASE.md                    ← Detailed release docs
│
└── homebrew-history-viewer (tap)
    └── Formula/
        └── history-viewer.rb         ← Auto-updated by workflow
```

## Data Flow Summary

```
Developer                        GitHub Actions                  Users
─────────                        ──────────────                  ─────

git tag v1.0.0      →     Workflow Triggered
git push origin v1.0.0          ↓
                           Build Binaries
                           (4 platforms)
                                ↓
                         Create Release
                         + Upload Assets
                                ↓
                         Update Homebrew
                         Formula
                                ↓
                                                ←   brew install
                                                    (macOS users)
                                                
                                                ←   curl + install
                                                    (Linux users)
```

## Timeline Example

```
T+0:00   Developer pushes tag v1.0.0
T+0:05   Workflow starts
T+0:10   macOS builds complete (parallel)
T+0:10   Linux builds complete (parallel)
T+0:12   Release created with all assets
T+0:45   Homebrew formula updated
T+1:00   Users can install via brew or manual download
```

## Key Features

✅ **Parallel Builds** - All 4 platforms build simultaneously
✅ **Cross-compilation** - Build ARM binaries on x86 runners
✅ **Automated Checksums** - SHA256 for security verification
✅ **Auto Homebrew Update** - No manual formula maintenance
✅ **Manual Trigger Option** - Test without creating tags
✅ **Comprehensive Documentation** - Multiple reference docs

## Build Tags

The workflow uses `-tags web` which:
- Builds web UI version (default)
- No GUI dependencies required on target system
- More portable binaries
- Easier to distribute

To build with native UI support instead, modify the workflow to remove `-tags web`.
