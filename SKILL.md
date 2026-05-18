---
name: pp-nextcloud
description: "Printing Press CLI for Nextcloud. Nextcloud OCS v2, WebDAV, CalDAV, CardDAV, and app APIs for files, sharing, users, notes, calendar, Talk, activity,..."
author: "kelvincushman"
license: "Apache-2.0"
argument-hint: "<command> [args] | install cli|mcp"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - nextcloud-pp-cli
---

# Nextcloud — Printing Press CLI

## Prerequisites: Install the CLI

This skill drives the `nextcloud-pp-cli` binary. **You must verify the CLI is installed before invoking any command from this skill.** If it is missing, install it first:

1. Install via the Printing Press installer:
   ```bash
   npx -y @mvanhorn/printing-press install nextcloud --cli-only
   ```
2. Verify: `nextcloud-pp-cli --version`
3. Ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on `$PATH`.

If the `npx` install fails before this CLI has a public-library category, install Node or use the category-specific Go fallback after publish.

If `--version` reports "command not found" after install, the install step did not put the binary on `$PATH`. Do not proceed with skill commands until verification succeeds.

Nextcloud OCS v2, WebDAV, CalDAV, CardDAV, and app APIs for files, sharing, users, notes, calendar, Talk, activity, Deck, contacts, and more.

## Command Reference

**index-php** — Manage index php

- `nextcloud-pp-cli index-php create-deck-board` — Create a Deck board
- `nextcloud-pp-cli index-php create-deck-card` — Create a card in a Deck stack
- `nextcloud-pp-cli index-php create-note` — Create a note
- `nextcloud-pp-cli index-php delete-note` — Delete a note
- `nextcloud-pp-cli index-php get-deck-board` — Get a Deck board
- `nextcloud-pp-cli index-php get-note` — Get a note
- `nextcloud-pp-cli index-php list-deck-boards` — List Deck boards
- `nextcloud-pp-cli index-php list-deck-stacks` — List stacks in a Deck board
- `nextcloud-pp-cli index-php list-notes` — List all notes
- `nextcloud-pp-cli index-php update-note` — Update a note

**ocs** — Manage ocs

- `nextcloud-pp-cli ocs add-user-to-group` — Add user to group
- `nextcloud-pp-cli ocs clear-user-status-message` — Clear user status message
- `nextcloud-pp-cli ocs create-group` — Create a group
- `nextcloud-pp-cli ocs create-share` — Create a share
- `nextcloud-pp-cli ocs create-user` — Create a new user
- `nextcloud-pp-cli ocs delete-group` — Delete a group
- `nextcloud-pp-cli ocs delete-share` — Delete a share
- `nextcloud-pp-cli ocs delete-user` — Delete a user
- `nextcloud-pp-cli ocs disable-user` — Disable a user
- `nextcloud-pp-cli ocs enable-user` — Enable a user
- `nextcloud-pp-cli ocs get-capabilities` — Get server capabilities and version
- `nextcloud-pp-cli ocs get-group-members` — Get group members
- `nextcloud-pp-cli ocs get-share` — Get share details
- `nextcloud-pp-cli ocs get-user` — Get user details
- `nextcloud-pp-cli ocs get-user-groups` — Get groups for a user
- `nextcloud-pp-cli ocs get-user-status` — Get own user status
- `nextcloud-pp-cli ocs list-activity` — List activity feed
- `nextcloud-pp-cli ocs list-groups` — List groups
- `nextcloud-pp-cli ocs list-notifications` — List your notifications
- `nextcloud-pp-cli ocs list-rooms` — List Talk conversations
- `nextcloud-pp-cli ocs list-shares` — List all shares
- `nextcloud-pp-cli ocs list-user-statuses` — List all user statuses
- `nextcloud-pp-cli ocs list-users` — List users
- `nextcloud-pp-cli ocs read-messages` — Read messages from a Talk room
- `nextcloud-pp-cli ocs remove-user-from-group` — Remove user from group
- `nextcloud-pp-cli ocs send-message` — Send a message to a Talk room
- `nextcloud-pp-cli ocs send-notification` — Send a notification to a user
- `nextcloud-pp-cli ocs set-user-status-message` — Set user status message
- `nextcloud-pp-cli ocs update-share` — Update a share (permissions, expiry, password)
- `nextcloud-pp-cli ocs update-user` — Update user attributes

**remote-php** — Manage remote php

- `nextcloud-pp-cli remote-php create-contact` — Create/update a contact (CardDAV PUT)
- `nextcloud-pp-cli remote-php create-event` — Create a calendar event (CalDAV PUT)
- `nextcloud-pp-cli remote-php create-system-tag` — Create a system tag
- `nextcloud-pp-cli remote-php delete-contact` — Delete a contact
- `nextcloud-pp-cli remote-php delete-file` — Delete a file or directory
- `nextcloud-pp-cli remote-php download-file` — Download a file
- `nextcloud-pp-cli remote-php tag-file` — Tag a file with a system tag
- `nextcloud-pp-cli remote-php untag-file` — Remove a tag from a file
- `nextcloud-pp-cli remote-php upload-file` — Upload a file


### Finding the right command

When you know what you want to do but not which command does it, ask the CLI directly:

```bash
nextcloud-pp-cli which "<capability in your own words>"
```

`which` resolves a natural-language capability query to the best matching command from this CLI's curated feature index. Exit code `0` means at least one match; exit code `2` means no confident match — fall back to `--help` or use a narrower query.

## Auth Setup
Run `nextcloud-pp-cli auth setup` to print the URL and steps for getting a key (add `--launch` to open the URL). Then set:

```bash
export NEXTCLOUD_USERNAME="<your-key>"
```

Or persist it in `~/.config/nextcloud-pp-cli/config.toml`.

Run `nextcloud-pp-cli doctor` to verify setup.

## Agent Mode

Add `--agent` to any command. Expands to: `--json --compact --no-input --no-color --yes`.

- **Pipeable** — JSON on stdout, errors on stderr
- **Filterable** — `--select` keeps a subset of fields. Dotted paths descend into nested structures; arrays traverse element-wise. Critical for keeping context small on verbose APIs:

  ```bash
  nextcloud-pp-cli index-php create-deck-board --color example-value --agent --select id,name,status
  ```
- **Previewable** — `--dry-run` shows the request without sending
- **Offline-friendly** — sync/search commands can use the local SQLite store when available
- **Non-interactive** — never prompts, every input is a flag
- **Explicit retries** — use `--idempotent` only when an already-existing create should count as success, and `--ignore-missing` only when a missing delete target should count as success

### Response envelope

Commands that read from the local store or the API wrap output in a provenance envelope:

```json
{
  "meta": {"source": "live" | "local", "synced_at": "...", "reason": "..."},
  "results": <data>
}
```

Parse `.results` for data and `.meta.source` to know whether it's live or local. A human-readable `N results (live)` summary is printed to stderr only when stdout is a terminal AND no machine-format flag (`--json`, `--csv`, `--compact`, `--quiet`, `--plain`, `--select`) is set — piped/agent consumers and explicit-format runs get pure JSON on stdout.

## Agent Feedback

When you (or the agent) notice something off about this CLI, record it:

```
nextcloud-pp-cli feedback "the --since flag is inclusive but docs say exclusive"
nextcloud-pp-cli feedback --stdin < notes.txt
nextcloud-pp-cli feedback list --json --limit 10
```

Entries are stored locally at `~/.nextcloud-pp-cli/feedback.jsonl`. They are never POSTed unless `NEXTCLOUD_FEEDBACK_ENDPOINT` is set AND either `--send` is passed or `NEXTCLOUD_FEEDBACK_AUTO_SEND=true`. Default behavior is local-only.

Write what *surprised* you, not a bug report. Short, specific, one line: that is the part that compounds.

## Output Delivery

Every command accepts `--deliver <sink>`. The output goes to the named sink in addition to (or instead of) stdout, so agents can route command results without hand-piping. Three sinks are supported:

| Sink | Effect |
|------|--------|
| `stdout` | Default; write to stdout only |
| `file:<path>` | Atomically write output to `<path>` (tmp + rename) |
| `webhook:<url>` | POST the output body to the URL (`application/json` or `application/x-ndjson` when `--compact`) |

Unknown schemes are refused with a structured error naming the supported set. Webhook failures return non-zero and log the URL + HTTP status on stderr.

## Named Profiles

A profile is a saved set of flag values, reused across invocations. Use it when a scheduled agent calls the same command every run with the same configuration - HeyGen's "Beacon" pattern.

```
nextcloud-pp-cli profile save briefing --json
nextcloud-pp-cli --profile briefing index-php create-deck-board --color example-value
nextcloud-pp-cli profile list --json
nextcloud-pp-cli profile show briefing
nextcloud-pp-cli profile delete briefing --yes
```

Explicit flags always win over profile values; profile values win over defaults. `agent-context` lists all available profiles under `available_profiles` so introspecting agents discover them at runtime.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 4 | Authentication required |
| 5 | API error (upstream issue) |
| 7 | Rate limited (wait and retry) |
| 10 | Config error |

## Argument Parsing

Parse `$ARGUMENTS`:

1. **Empty, `help`, or `--help`** → show `nextcloud-pp-cli --help` output
2. **Starts with `install`** → ends with `mcp` → MCP installation; otherwise → see Prerequisites above
3. **Anything else** → Direct Use (execute as CLI command with `--agent`)

## MCP Server Installation

Install the MCP binary from this CLI's published public-library entry or pre-built release, then register it:

```bash
claude mcp add nextcloud-pp-mcp -- nextcloud-pp-mcp
```

Verify: `claude mcp list`

## Direct Use

1. Check if installed: `which nextcloud-pp-cli`
   If not found, offer to install (see Prerequisites at the top of this skill).
2. Match the user query to the best command from the Unique Capabilities and Command Reference above.
3. Execute with the `--agent` flag:
   ```bash
   nextcloud-pp-cli <command> [subcommand] [args] --agent
   ```
4. If ambiguous, drill into subcommand help: `nextcloud-pp-cli <command> --help`.
