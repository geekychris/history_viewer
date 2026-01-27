# Custom Category Patterns

You can now define custom regex patterns to categorize commands in the History Viewer. This allows you to create your own categories beyond the built-in ones.

## Using the UI

1. Click the **⚙️ Settings** button in the top-right corner
2. In the **Custom Command Categories** section, you'll see:
   - A list of your current custom patterns
   - Input fields to add new patterns
   - A collapsible section showing all built-in categories

3. To add a custom pattern:
   - **Simple Mode** (recommended): Just enter the command name
     - Category: `my-tools`
     - Command: `history_viewer`
   - **Advanced Mode**: Use regex patterns for multiple commands or complex matching
     - Category: `cloud-tools`
     - Pattern: `^(aws|gcloud|az)\s`
   - Click **➕ Add**

4. Click **💾 Save Settings** and refresh the page

## Pattern Examples

### Infrastructure as Code
```json
{
  "category": "terraform",
  "pattern": "^terraform\\s"
}
```

### AI/LLM Tools
```json
{
  "category": "ai-tools",
  "pattern": "^(ollama|llm|aider|cursor)\\s"
}
```

### Cloud CLIs
```json
{
  "category": "cloud-cli",
  "pattern": "^(aws|gcloud|az|doctl|linode-cli)\\s"
}
```

### Custom Scripts
```json
{
  "category": "my-scripts",
  "pattern": "^\\./scripts/"
}
```

### Language-specific Tools
```json
{
  "category": "rust-dev",
  "pattern": "^(cargo|rustc|rustup|rust-analyzer)\\s"
}
```

## Manual Configuration

You can also add custom patterns directly to `~/.history_viewer.json`:

```json
{
  "custom_category_patterns": [
    {
      "category": "terraform",
      "pattern": "^terraform\\s"
    },
    {
      "category": "ai-tools", 
      "pattern": "^(ollama|llm|aider)\\s"
    }
  ]
}
```

## Pattern Priority

Custom patterns are checked **before** built-in patterns, so you can:
- Override built-in categorization
- Add new categories for tools not covered by defaults
- Create more specific categories (e.g., separate `rust-dev` from general `dev-tools`)

## Built-in Categories

The following categories are built into the system:
- **version-control**: git, hg, svn, bzr, cvs
- **build**: make, cmake, cargo, npm, yarn, gradle, mvn, go build, gcc, etc.
- **file-operations**: cp, mv, rm, mkdir, chmod, cat, rsync, etc.
- **navigation**: cd, ls, pwd, tree, find, locate, which
- **dev-tools**: vim, code, gdb, valgrind, strace
- **system-admin**: sudo, systemctl, kill, ps, top, htop, df, du
- **network**: curl, wget, ssh, ping, dig, nc
- **containers**: docker, kubectl, helm, podman
- **database**: psql, mysql, mongo, redis-cli
- **editor**: vim, emacs, nano
- **search**: grep, ag, rg, ack
- **package-manager**: apt, brew, pip, npm, cargo

## Regex Tips

- Start patterns with `^` to match from the beginning of the command
- Use `\s` to match whitespace (space, tab)
- Use `(cmd1|cmd2|cmd3)` for multiple commands
- Escape special characters with `\`
- Test your regex at https://regex101.com

### Common Patterns
- Match specific command: `^mycommand\s`
- Match command with subcommands: `^kubectl\s+(get|describe|apply)`
- Match script directory: `^\./scripts/`
- Match commands starting with prefix: `^tf-`
