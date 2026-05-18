# nextcloud-pp-cli

A complete Nextcloud CLI for AI agents and power users — covering every major Nextcloud API surface in one binary: files, sharing, users, groups, notes, calendar, Talk, Deck, contacts, activity, and more.

Built by [Kelvin Cushman](https://github.com/kelvincushman) · Generated with [CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)

---

## Why this exists

Every existing Nextcloud CLI is narrow: file sync only, share creation only, or a single-app wrapper. Agents need a single binary that covers **all** Nextcloud apps with:

- **JSON output everywhere** (`--json`, `--agent`, `--select id,name`)
- **Offline-first SQLite cache** — sync once, search forever without hitting the server
- **WebDAV + OCS + CalDAV + CardDAV** all wired up — no separate tools
- **Agent-safe defaults** — no interactive prompts, deterministic exit codes
- **Transcendence commands** only possible by combining multiple APIs (storage summary, stale share reports, access trees, activity digests)

## Install

### From source (Go)

```bash
git clone https://github.com/kelvincushman/nextcloud-pp-cli.git
cd nextcloud-pp-cli
go build -o nextcloud-pp-cli ./cmd/nextcloud-pp-cli
go build -o nextcloud-pp-mcp ./cmd/nextcloud-pp-mcp
```

### Pre-built binary

Download from the [Releases](https://github.com/kelvincushman/nextcloud-pp-cli/releases) page for your platform.

On Unix/macOS:
```bash
chmod +x nextcloud-pp-cli
sudo mv nextcloud-pp-cli /usr/local/bin/
```

## Quick Start

### 1. Set credentials

```bash
export NEXTCLOUD_HOST="http://your-nextcloud-server"
export NEXTCLOUD_USERNAME="admin"
export NEXTCLOUD_PASSWORD="your-password"
```

Persist them in `~/.config/nextcloud-pp-cli/config.toml`:

```toml
base_url = "http://your-nextcloud-server"
base_path = ""

[headers]
  "OCS-APIRequest" = "true"
```

### 2. Check connectivity

```bash
nextcloud-pp-cli doctor
```

### 3. Try it

```bash
# List your files
nextcloud-pp-cli remote-php list-files admin

# List users
nextcloud-pp-cli ocs list-users

# See recent activity
nextcloud-pp-cli ocs list-activity --json

# Get a full AI-agent context dump
nextcloud-pp-cli context --json
```

---

## Commands

### Files (WebDAV)

| Command | Description |
|---------|-------------|
| `remote-php list-files <user> [path]` | List files and directories (PROPFIND). Use `--recursive` for tree view. |
| `remote-php upload-file <user> <dest>` | Upload a file. Use `--overwrite` to replace existing. |
| `remote-php download-file <user> <path>` | Download a file to stdout or `--deliver file:<path>`. |
| `remote-php delete-file <user> <path>` | Delete a file or directory. Use `--recursive`. |
| `remote-php mkdir <user> <path>` | Create a directory (MKCOL). Use `--parents`. |
| `remote-php move <user> <src> <dest>` | Move or rename (MOVE). Use `--overwrite`. |
| `remote-php copy <user> <src> <dest>` | Copy a file or directory (COPY). |

### Sharing

| Command | Description |
|---------|-------------|
| `ocs list-shares` | List all shares. Filter by `--path`, `--shared-with-me`. |
| `ocs create-share --path /file --share-type 3` | Create a public link (type 3), user share (0), or group share (1). |
| `ocs update-share <id> --permissions 1 --expire-date 2026-12-31` | Update share permissions or expiry. |
| `ocs delete-share <id>` | Remove a share. |
| `shares-stale --days 30` | **Report shares that have expired or haven't been updated in N days.** |
| `access-tree <path>` | **Show every user and group that has access to a path.** |

### Users & Groups

| Command | Description |
|---------|-------------|
| `ocs list-users` | List all users. |
| `ocs get-user <id>` | Get detailed user info (quota, groups, email). |
| `ocs create-user --userid alice --email alice@example.com` | Create a user. |
| `ocs disable-user <id>` / `ocs enable-user <id>` | Enable/disable a user. |
| `ocs list-groups` | List all groups. |
| `ocs get-group-members <group>` | List members of a group. |
| `ocs add-user-to-group <user> --groupid editors` | Add a user to a group. |
| `storage-summary` | **Summarize quota usage across all users. Use `--threshold 80` to flag over-quota users.** |
| `quota-check --threshold 80` | **Exit code 1 if any user exceeds the threshold (for monitoring).** |

### Notes

| Command | Description |
|---------|-------------|
| `index-php list-notes` | List notes. Filter by `--category`. |
| `index-php create-note --title "Meeting" --content "..."` | Create a note. |
| `index-php update-note <id>` | Edit note title, content, or category. |
| `index-php delete-note <id>` | Delete a note. |

### Talk (Chat)

| Command | Description |
|---------|-------------|
| `ocs list-rooms` | List Talk conversations. |
| `ocs read-messages <token>` | Read messages from a conversation. Use `--limit`. |
| `ocs send-message <token> --message "Hello"` | Send a message to a room. |

### Deck (Project Boards)

| Command | Description |
|---------|-------------|
| `index-php list-deck-boards` | List all Deck boards. |
| `index-php get-deck-board <id>` | Get a board with its stacks. |
| `index-php list-deck-stacks <boardId>` | List stacks (columns) in a board. |
| `index-php create-deck-card <boardId> <stackId>` | Create a new card. |

### Calendar & Contacts

| Command | Description |
|---------|-------------|
| `remote-php create-event <user> <calendar> <file.ics>` | Create a calendar event from iCal data. |
| `remote-php create-contact <user> contacts <file.vcf>` | Add a contact from vCard data. |
| `remote-php delete-contact <user> contacts <file.vcf>` | Remove a contact. |

### Activity Feed

| Command | Description |
|---------|-------------|
| `ocs list-activity` | List activity feed. Filter by `--filter files`, `--object-type`, `--since <id>`. |
| `digest --since 24h` | **Activity digest grouped by user and file for the last N hours/days.** |

### User Status

| Command | Description |
|---------|-------------|
| `ocs get-user-status` | Get your current status. |
| `ocs set-user-status-message --message "In a meeting" --status-icon 🗓️` | Set your status. |
| `ocs clear-user-status-message` | Clear status. |
| `ocs list-user-statuses` | See all users' statuses. |

### Notifications

| Command | Description |
|---------|-------------|
| `ocs list-notifications` | List your pending notifications. |
| `ocs send-notification <userId> --short-message "Alert"` | Send an admin notification. |

### Tags

| Command | Description |
|---------|-------------|
| `remote-php create-system-tag --name "reviewed"` | Create a system tag. |
| `remote-php tag-file <fileId> <tagId>` | Tag a file. |
| `remote-php untag-file <fileId> <tagId>` | Remove a tag. |

### Intelligence / Transcendence

These commands require combining multiple API calls — only possible with a unified CLI:

| Command | Description |
|---------|-------------|
| `storage-summary` | Storage usage across all users with totals. `--threshold 80` filters to over-quota only. |
| `shares-stale --days 30` | Shares that are expired or haven't been updated. |
| `access-tree /path/to/file` | Full access tree: who can see this file (users + groups expanded). |
| `digest --since 24h` | Activity digest: what changed, grouped by user and file. |
| `quota-check --threshold 80` | Monitoring check — exits 1 if any user exceeds quota threshold. |
| `sync-status` | Compare local SQLite cache vs live API to detect drift. |
| `context --json` | **Single-call context dump** for AI agents: capabilities, quota, recent files, shares, activity, Talk rooms, notifications. |

---

## AI Agent Usage

This CLI is optimized for use as an MCP server or direct agent tool:

```bash
# Run the MCP server — exposes all commands as agent tools
./nextcloud-pp-mcp

# One-shot agent mode (JSON + compact + no prompts)
nextcloud-pp-cli ocs list-users --agent

# Get full context in one call
nextcloud-pp-cli context --json

# Introspect all commands at runtime
nextcloud-pp-cli agent-context --pretty
```

**Claude Desktop / MCP config** (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "nextcloud": {
      "command": "/path/to/nextcloud-pp-mcp",
      "env": {
        "NEXTCLOUD_HOST": "http://your-nextcloud-server",
        "NEXTCLOUD_USERNAME": "admin",
        "NEXTCLOUD_PASSWORD": "your-password"
      }
    }
  }
}
```

Exit codes: `0` success · `2` usage error · `3` not found · `4` auth error · `5` API error · `7` rate limited · `10` config error

---

## Output Formats

```bash
# Human table (default in terminal, JSON when piped)
nextcloud-pp-cli remote-php list-files admin

# JSON for agents and scripts
nextcloud-pp-cli remote-php list-files admin --json

# Filter fields
nextcloud-pp-cli ocs list-users --json --select id,displayname,email

# Compact (key fields only)
nextcloud-pp-cli ocs list-shares --compact

# Dry run
nextcloud-pp-cli ocs create-share --path /Documents --share-type 3 --dry-run
```

---

## Sync & Offline Search

```bash
# Sync all data to local SQLite cache
nextcloud-pp-cli sync

# After syncing, commands run from cache (instant, no network)
nextcloud-pp-cli remote-php list-files admin --data-source local

# Check sync status and cache health
nextcloud-pp-cli sync-status
```

---

## Configuration

Config file: `~/.config/nextcloud-pp-cli/config.toml`

```toml
# Required: your Nextcloud server URL
base_url = "http://your-nextcloud-server"
base_path = ""

# Required: add OCS-APIRequest header globally
[headers]
  "OCS-APIRequest" = "true"
```

Environment variables:

| Variable | Required | Description |
|----------|----------|-------------|
| `NEXTCLOUD_HOST` | Yes (alt to base_url) | Nextcloud server URL |
| `NEXTCLOUD_USERNAME` | Yes | Nextcloud username |
| `NEXTCLOUD_PASSWORD` | Yes | Nextcloud password or app password |

For production use, generate an **App Password** in Nextcloud Settings → Security → App Passwords instead of your account password.

---

## Troubleshooting

**404 on all OCS commands**
- Check that `OCS-APIRequest: true` is set globally (either in `config.toml` `[headers]` or via a shell function)
- Verify `base_url` is set to your Nextcloud server (not `base_path`)

**Auth errors (exit code 4)**
- Run `nextcloud-pp-cli doctor`
- Check `echo $NEXTCLOUD_USERNAME` and `echo $NEXTCLOUD_PASSWORD`
- Generate an App Password in Nextcloud Settings → Security

**Not found (exit code 3)**
- Check the resource ID: use the `list` command to see available items

---

## Credits

Built by **[Kelvin Cushman](https://github.com/kelvincushman)** (kelvin.cushman@gmail.com)

Generated with **[CLI Printing Press](https://github.com/mvanhorn/cli-printing-press)** by [@mvanhorn](https://github.com/mvanhorn) — the tool that researches the API landscape, absorbs competing tools, and generates a production-ready Go CLI with MCP server, offline SQLite cache, and agent-native output.

---

## License

Apache-2.0 — see [LICENSE](LICENSE)
