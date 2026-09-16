#!/usr/bin/env zsh
# history_viewer preexec hook — log $PWD alongside every command.
#
# Zsh's built-in ~/.zsh_history file records timestamps and commands but
# not the working directory each command ran in. Without that, the viewer
# has to *infer* cwd from `cd` chains, which drifts badly because every
# terminal writes into the same history file. This hook fixes the root
# cause by writing a parallel log with cwd baked in.
#
# The main ~/.zsh_history is left completely untouched so other tools
# (fzf history search, atuin, syntax-highlighting, etc.) keep working.
#
# File written:  ${HISTORY_VIEWER_CWD_LOG:-$HOME/.zsh_cwd_history}
# Line format:   <epoch>\t<pwd>\t<cmd>
# In <cmd>, literal newlines are escaped to `\n` and literal tabs to
# `\t` so each record fits on one line.
#
# To disable: remove the source line from ~/.zshrc.

# Skip if we're not interactive — no history to enrich.
[[ -o interactive ]] || return 0

# EPOCHSECONDS lives in the zsh/datetime module; loading is a no-op if
# it's already loaded.
zmodload zsh/datetime 2>/dev/null

_history_viewer_log_cwd() {
    emulate -L zsh
    # $1 is the command about to run (aliases expanded). Escape tab and
    # newline so the record fits on one physical line.
    local cmd=${1//$'\n'/\\n}
    cmd=${cmd//$'\t'/\\t}
    local pwd_esc=${PWD//$'\t'/\\t}
    local ts=${EPOCHSECONDS:-$(date +%s)}
    print -r -- "${ts}	${pwd_esc}	${cmd}" >> "${HISTORY_VIEWER_CWD_LOG:-$HOME/.zsh_cwd_history}"
}

autoload -Uz add-zsh-hook
add-zsh-hook preexec _history_viewer_log_cwd
