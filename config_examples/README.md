# Session Configuration Examples

This directory contains example configurations for different work styles and use cases.

## Available Examples

### 1. `minimal.json` - Simple time-based
For users who just want basic session detection.

### 2. `project_based.json` - Project-focused work
For developers who work on multiple distinct projects and want sessions to represent "work on project X".

### 3. `task_based.json` - Activity-focused work
For users who switch between different types of tasks (coding, devops, file management) and want sessions grouped by activity type.

### 4. `deep_work.json` - Long focused sessions
For developers who do long, uninterrupted work sessions but want reasonable caps.

### 5. `rapid_switch.json` - Context switching
For users who rapidly switch between projects and tasks.

### 6. `devops.json` - Infrastructure work
Optimized for DevOps engineers who work with containers, cloud, and automation.

### 7. `research.json` - Exploration and learning
For users who explore codebases, read documentation, and experiment.

## How to Use

1. **Copy an example**:
   ```bash
   cp config_examples/project_based.json ~/.history_viewer.json
   ```

2. **Or merge with existing config**:
   ```bash
   # Edit your config
   nano ~/.history_viewer.json
   
   # Copy the session_heuristics section from an example
   ```

3. **Restart history viewer** for changes to take effect

4. **Iterate**: Start with one example, observe your sessions, then tweak parameters

## Creating Your Own

Mix and match parameters based on your workflow:

```json
{
  "session_heuristics": {
    "timeout_minutes": 30,              // How long before session ends
    "directory_change_breaks_session": false,  // Break on directory change?
    "category_change_threshold": 0,     // Break after N different command types (0 = off)
    "min_commands_per_session": 1,      // Minimum commands to be a session
    "max_session_duration_minutes": 0,  // Maximum session length (0 = no limit)
    "short_break_minutes": 5            // (Not yet implemented)
  }
}
```

## Tips

- **Start simple**: Use just `timeout_minutes` initially
- **One change at a time**: Add one heuristic, observe results, adjust
- **Different configs for different contexts**: Keep multiple configs and swap them based on what you're working on
- **Tune over time**: Your ideal settings may change as your workflow evolves

## Feedback

If you discover a great configuration for a specific use case, consider contributing it back!
