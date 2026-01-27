# Session Configuration Quick Start

## TL;DR

Sessions group your commands into logical work periods. Configure how sessions are detected by editing `~/.history_viewer.json`.

## Quick Setup

1. **Create config file** (if it doesn't exist):
```bash
cat > ~/.history_viewer.json << 'EOF'
{
  "session_heuristics": {
    "timeout_minutes": 30
  }
}
EOF
```

2. **Restart history viewer** for changes to take effect

## Most Common Tweaks

### Make sessions shorter/longer
```json
{
  "session_heuristics": {
    "timeout_minutes": 15  // 15 min inactivity = new session
  }
}
```

### Split sessions when changing projects
```json
{
  "session_heuristics": {
    "timeout_minutes": 30,
    "directory_change_breaks_session": true  // cd to different project = new session
  }
}
```

### Hide tiny sessions
```json
{
  "session_heuristics": {
    "timeout_minutes": 30,
    "min_commands_per_session": 5  // Need at least 5 commands to be a session
  }
}
```

### Cap session length
```json
{
  "session_heuristics": {
    "timeout_minutes": 30,
    "max_session_duration_minutes": 120  // Max 2 hours per session
  }
}
```

## All Parameters

| Parameter | Default | What It Does |
|-----------|---------|--------------|
| `timeout_minutes` | `30` | Minutes of inactivity before new session |
| `directory_change_breaks_session` | `false` | New session when changing project directories |
| `category_change_threshold` | `0` (off) | New session after N commands of different type |
| `min_commands_per_session` | `1` | Minimum commands to create a session |
| `max_session_duration_minutes` | `0` (off) | Maximum session length |
| `short_break_minutes` | `5` | ⚠️ Not yet implemented |

## Full Example

```json
{
  "history_file": "~/.zsh_history",
  "port": 8080,
  "session_heuristics": {
    "timeout_minutes": 30,
    "directory_change_breaks_session": true,
    "category_change_threshold": 0,
    "min_commands_per_session": 3,
    "max_session_duration_minutes": 180,
    "short_break_minutes": 5
  }
}
```

## Need More Details?

See [SESSION_HEURISTICS.md](SESSION_HEURISTICS.md) for comprehensive documentation.
