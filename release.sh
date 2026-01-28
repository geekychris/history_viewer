#!/bin/bash
set -e

# Get the latest tag
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
echo "Latest tag: $LATEST_TAG"

# Remove 'v' prefix and split version
VERSION=${LATEST_TAG#v}
IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION"

# Prompt for version bump type
echo ""
echo "Current version: $VERSION"
echo "Select version bump:"
echo "1) Patch (v$MAJOR.$MINOR.$((PATCH+1)))"
echo "2) Minor (v$MAJOR.$((MINOR+1)).0)"
echo "3) Major (v$((MAJOR+1)).0.0)"
echo "4) Custom"
read -p "Enter choice [1-4]: " choice

case $choice in
  1)
    NEW_VERSION="$MAJOR.$MINOR.$((PATCH+1))"
    ;;
  2)
    NEW_VERSION="$MAJOR.$((MINOR+1)).0"
    ;;
  3)
    NEW_VERSION="$((MAJOR+1)).0.0"
    ;;
  4)
    read -p "Enter custom version (without 'v' prefix): " NEW_VERSION
    ;;
  *)
    echo "Invalid choice"
    exit 1
    ;;
esac

NEW_TAG="v$NEW_VERSION"

# Confirm
echo ""
echo "Will create and push tag: $NEW_TAG"
read -p "Continue? [y/N]: " confirm
if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
  echo "Aborted"
  exit 0
fi

# Check for uncommitted changes
if ! git diff-index --quiet HEAD --; then
  echo ""
  echo "Warning: You have uncommitted changes"
  read -p "Continue anyway? [y/N]: " continue_dirty
  if [[ ! "$continue_dirty" =~ ^[Yy]$ ]]; then
    echo "Aborted"
    exit 0
  fi
fi

# Create and push tag
echo ""
echo "Creating tag $NEW_TAG..."
git tag -a "$NEW_TAG" -m "Release $NEW_TAG"

echo "Pushing tag to origin..."
git push origin "$NEW_TAG"

echo ""
echo "✅ Tag $NEW_TAG pushed successfully!"
echo ""
echo "GitHub Actions will now:"
echo "  1. Build binaries for macOS (arm64/amd64) and Linux (amd64)"
echo "  2. Create a GitHub release"
echo "  3. Update the Homebrew formula"
echo ""
echo "Monitor progress at:"
echo "  https://github.com/geekychris/history_viewer/actions"
echo ""
echo "Or run: gh run watch"
