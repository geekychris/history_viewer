# Session Detection Heuristics

## Overview

Sessions are logical groupings of shell commands that represent a coherent work period. The history viewer uses multiple configurable heuristics to determine when one session ends and another begins.

## Current Implementation

Currently, sessions are determined **solely by time-based inactivity**. A new session starts when there's a gap larger than `session_timeout_minutes` (default: 30 minutes) between consecutive commands.

## Configurable Heuristics

All session detection parameters can be configured in `~/.history_viewer.json` under the `session_heuristics` section.

### 1. **timeout_minutes** (Primary Heuristic)
**Default:** `30`  
**Type:** Integer (minutes)

The primary timeout between commands. If more than this many minutes pass between commands, a new session begins.

**Example use cases:**
- `15`: Shorter sessions for rapid context switching
- `60`: Longer sessions for deep work
- `120`: Very long sessions for all-day projects

```json
{
  "session_heuristics": {
    "timeout_minutes": 30
  }
}
```

---

### 2. **directory_change_breaks_session**
**Default:** `false`  
**Type:** Boolean

When enabled, changing to a completely unrelated directory tree starts a new session. Two directories are considered "related" if:
- They are the same directory
- One is a subdirectory of the other
- They share the same parent directory

**Example:**
- `~/projects/app1` → `~/projects/app1/src` = Related (same session)
- `~/projects/app1` → `~/projects/app2` = Related (same parent, same session)
- `~/projects/app1` → `~/documents` = Unrelated (new session if enabled)

**When to enable:**
- You work on distinct projects in different directory trees
- You want sessions to represent "work on project X"

```json
{
  "session_heuristics": {
    "directory_change_breaks_session": true
  }
}
```

---

### 3. **category_change_threshold**
**Default:** `0` (disabled)  
**Type:** Integer (number of commands)

If you run N consecutive commands from a different category, start a new session. Categories include: `version-control`, `build`, `file-operations`, `navigation`, `dev-tools`, etc.

**Example with threshold = 3:**
```
git commit     (version-control)
git push       (version-control)  <- Working on git
docker build   (containers)       <- 1 different
docker run     (containers)       <- 2 different
docker ps      (containers)       <- 3 different → NEW SESSION
```

**When to enable:**
- You want sessions grouped by activity type
- Set to `3-5` for moderate sensitivity
- Set to `7-10` for low sensitivity

```json
{
  "session_heuristics": {
    "category_change_threshold": 5
  }
}
```

---

### 4. **min_commands_per_session**
**Default:** `1`  
**Type:** Integer (number of commands)

Minimum number of commands required to create a session. Prevents tiny, single-command sessions.

**Example with min = 3:**
- A session with only 1-2 commands will be merged into the previous session

**When to adjust:**
- Set to `3-5` to filter out trivial sessions
- Keep at `1` to preserve all command history

```json
{
  "session_heuristics": {
    "min_commands_per_session": 3
  }
}
```

---

### 5. **max_session_duration_minutes**
**Default:** `0` (disabled)  
**Type:** Integer (minutes)

Absolute maximum duration for any session, even without timeout gaps. Forces a new session after this duration.

**Example with max = 180 (3 hours):**
- Even if you're actively working, after 3 hours a new session begins
- Useful for breaking up all-day work into digestible chunks

**When to enable:**
- You have very long work sessions
- You want to break sessions by time-of-day (morning/afternoon)
- Set to `120-240` minutes (2-4 hours)

```json
{
  "session_heuristics": {
    "max_session_duration_minutes": 180
  }
}
```

---

### 6. **short_break_minutes**
**Default:** `5`  
**Type:** Integer (minutes)

⚠️ **Currently not implemented** - Reserved for future use.

Intended purpose: Define a "short break" duration that doesn't end a session (e.g., coffee break, quick meeting). Commands within a short break would remain in the same session even if they exceed `timeout_minutes`.

---

## Example Configurations

### Default (Time-based only)
```json
{
  "session_heuristics": {
    "timeout_minutes": 30,
    "directory_change_breaks_session": false,
    "category_change_threshold": 0,
    "min_commands_per_session": 1,
    "max_session_duration_minutes": 0,
    "short_break_minutes": 5
  }
}
```

### Project-focused (Directory-based sessions)
```json
{
  "session_heuristics": {
    "timeout_minutes": 45,
    "directory_change_breaks_session": true,
    "category_change_threshold": 0,
    "min_commands_per_session": 3,
    "max_session_duration_minutes": 0
  }
}
```

### Task-focused (Category-based sessions)
```json
{
  "session_heuristics": {
    "timeout_minutes": 20,
    "directory_change_breaks_session": false,
    "category_change_threshold": 5,
    "min_commands_per_session": 2,
    "max_session_duration_minutes": 120
  }
}
```

### Deep work (Long sessions with caps)
```json
{
  "session_heuristics": {
    "timeout_minutes": 60,
    "directory_change_breaks_session": false,
    "category_change_threshold": 0,
    "min_commands_per_session": 5,
    "max_session_duration_minutes": 240
  }
}
```

### Rapid switching (Short, focused sessions)
```json
{
  "session_heuristics": {
    "timeout_minutes": 10,
    "directory_change_breaks_session": true,
    "category_change_threshold": 3,
    "min_commands_per_session": 2,
    "max_session_duration_minutes": 60
  }
}
```

---

## How Heuristics Interact

Heuristics are evaluated in order, and **any single heuristic can trigger a new session**:

1. **Timeout check** (always active)
2. **Directory change** (if enabled)
3. **Category change** (if threshold > 0)
4. **Max duration** (if > 0)

After any heuristic triggers, the **min_commands_per_session** check ensures the session being closed meets the minimum threshold.

---

## Recommendations

**Start simple:**
1. Use only `timeout_minutes` initially (default behavior)
2. Observe your session patterns
3. Add one heuristic at a time based on your workflow

**Common workflows:**

| Workflow Type | Recommended Settings |
|--------------|---------------------|
| Full-stack web dev | `timeout=30`, `directory_change=true`, `min_commands=3` |
| DevOps/Infrastructure | `timeout=20`, `category_change=5`, `max_duration=120` |
| Research/Exploration | `timeout=45`, `min_commands=1` |
| Rapid prototyping | `timeout=15`, `directory_change=true`, `category_change=3` |

---

## Testing Your Configuration

After updating `~/.history_viewer.json`:

1. Restart the history viewer
2. Observe session boundaries in the UI
3. Adjust parameters iteratively
4. Consider creating separate config profiles for different work modes

---

## Future Enhancements

Potential additional heuristics under consideration:

- **Command patterns**: Start new session on specific commands (e.g., `tmux new`, `warp open`)
- **Time-of-day breaks**: Automatic session breaks at specific times (lunch, end of day)
- **Working hours detection**: Ignore overnight gaps differently than daytime gaps
- **Project markers**: Detect project boundaries via git repos or marker files
- **Machine learning**: Learn your personal session patterns over time
