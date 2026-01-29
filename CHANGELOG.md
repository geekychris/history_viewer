# Changelog

All notable changes to this project will be documented in this file.

## [1.2.4] - 2026-01-29

### Added
- Linux ARM64 build support using GitHub's free ARM64 runners for public repositories

## [1.2.3] - 2026-01-29

### Fixed
- Fixed GitHub Actions workflow to use HOMEBREW_TAP_TOKEN secret for Homebrew tap updates

## [0.1.2] - 2026-01-29

### Fixed
- Fixed GitHub Actions workflow to use correct secret name for Homebrew tap updates

## [0.1.1] - 2026-01-29

### Added
- **Stable Session IDs**: Sessions now have persistent hash-based IDs (e.g., `sess_abc123`) that remain stable across restarts
- **Session Persistence**: Session boundaries are saved to `~/.config/history_viewer/sessions.json`
- **Sequence Numbers**: Added display-friendly sequence numbers (e.g., "Session #5") while maintaining stable IDs for navigation
- New `session_index.go` module for managing session persistence

### Fixed
- **"Go to Session" Navigation**: Now works reliably from command search results
- **"Show All Commands" Button**: Fixed to work with string-based session IDs
- **Session Assignment**: All commands now get assigned to sessions (no more empty `session_id` fields)
- **Metadata Operations**: Fixed to use sequence numbers for notes, tags, and session metadata
- **Frontend onclick Handlers**: Properly quote all string session IDs in JavaScript

### Changed
- `Session.ID`: Changed from `int` to `string` for stable identification
- `HistoryEntry.SessionID`: Changed from `int` to `string`
- Added `Session.SequenceNumber`: New `int` field for display purposes
- Parser now updates both session commands and original entries with session IDs

### Build
- Added `pkg-config` to Ubuntu dependencies
- Disabled Linux ARM64 build (cross-compilation complexity with CGO/Fyne)
- Successful builds for: macOS (Intel & Apple Silicon), Linux (x86_64)

## [0.1.0] - 2026-01-26

### Added
- Initial release
- Web-based zsh history viewer
- Session grouping with configurable heuristics
- Command categorization
- AI analysis integration with Ollama
- Export to multiple formats (JSON, CSV, Markdown, Zsh)
- Notes and tags for sessions
- Native UI support (experimental)
- GitHub Actions CI/CD with GoReleaser
