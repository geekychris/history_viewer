#!/usr/bin/env bash
# Install the history_viewer preexec hook that logs $PWD per command.
#
# Copies the hook to ~/.config/history_viewer/cwd-hook.zsh and adds an
# idempotent BEGIN/END-marker source block to ~/.zshrc. Safe to re-run.
#
# Env:
#   ZDOTDIR            — override ~/.zshrc location
#   XDG_CONFIG_HOME    — override ~/.config
#   HISTORY_VIEWER_INSTALL_NONINTERACTIVE=1 — skip stdout blurb
set -euo pipefail

SCRIPT_DIR="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &>/dev/null && pwd )"
SRC="$SCRIPT_DIR/history-viewer-cwd-hook.zsh"

if [[ ! -f "$SRC" ]]; then
    echo "history_viewer install: hook source not found at $SRC" >&2
    exit 1
fi

TARGET_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/history_viewer"
HOOK_PATH="$TARGET_DIR/cwd-hook.zsh"
ZSHRC="${ZDOTDIR:-$HOME}/.zshrc"

mkdir -p "$TARGET_DIR"
cp "$SRC" "$HOOK_PATH"
chmod 644 "$HOOK_PATH"

BEGIN_MARK="# BEGIN history_viewer cwd hook"
END_MARK="# END history_viewer cwd hook"

added="no"
if [[ -f "$ZSHRC" ]] && grep -qF "$BEGIN_MARK" "$ZSHRC"; then
    :
else
    cat >> "$ZSHRC" <<EOF

$BEGIN_MARK
# Sources the history_viewer preexec hook that logs \$PWD per command
# to \$HOME/.zsh_cwd_history so the viewer can display exact cwd.
# Remove this block (and $HOOK_PATH) to disable.
[ -f "$HOOK_PATH" ] && source "$HOOK_PATH"
$END_MARK
EOF
    added="yes"
fi

if [[ "${HISTORY_VIEWER_INSTALL_NONINTERACTIVE:-0}" != "1" ]]; then
    echo "history_viewer shell hook:"
    echo "  hook installed at: $HOOK_PATH"
    if [[ "$added" == "yes" ]]; then
        echo "  added source block to: $ZSHRC"
    else
        echo "  source block already present in: $ZSHRC (skipped)"
    fi
    echo ""
    echo "New shells will log to \$HOME/.zsh_cwd_history automatically."
    echo "To activate in this shell now: exec zsh"
fi
