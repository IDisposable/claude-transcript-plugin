# transcript-saver

A Claude Code plugin that automatically saves readable Markdown transcripts of your sessions before context compaction.

## What It Does

When Claude Code compacts your conversation context (either automatically or via `/compact`), this plugin intercepts the `PreCompact` event and converts the full JSONL transcript into a clean, human-readable Markdown file.

**Output includes:**
- Timestamped User / Claude message sections
- Git branch change markers
- Tool calls with syntax-highlighted commands
- Tool results in collapsible `<details>` blocks
- Thinking blocks in collapsible sections
- System noise filtered out

## Installation

### From Marketplace

Add this repository as a plugin marketplace, then install:

```bash
/plugin marketplace add IDisposable/claude-transcript-plugin
/plugin install transcript-saver@transcript-saver
```

Or install directly from GitHub:

```bash
claude plugin install transcript-saver@transcript-saver --source github:IDisposable/claude-transcript-plugin
```

### Local Development

```bash
claude --plugin-dir /path/to/claude-transcript-plugin
```

## Output Location

Transcripts are saved to:

```
~/.claude/transcripts/<timestamp>_<session-id>.md
```

Override with the `TRANSCRIPT_OUTPUT_DIR` environment variable:

```bash
export TRANSCRIPT_OUTPUT_DIR=~/my-transcripts
```

## How It Works

| Component | Purpose |
|---|---|
| `hooks/hooks.json` | Registers a `PreCompact` hook |
| `scripts/save-transcript.py` | Converts JSONL → Markdown |
| `skills/save-transcript/` | Manual `/transcript-saver:save-transcript` skill |

The hook receives the session's `transcript_path` from Claude Code via stdin JSON, reads the JSONL file, and writes a formatted Markdown file. It exits cleanly on any error so it never blocks compaction.

## Requirements

- Python 3.8+
- Claude Code with plugin support

## Example Output

```markdown
# Session Transcript: my-project

**Started:** 2026-03-05T15:21:47.027Z
**Project:** `/home/user/my-project`

---

> *Branch: `main`*

## User _15:21:47_

Please review the authentication module.

---

## Claude _15:21:50_

I'll start by reading the auth files.

**Tool: Read** `src/auth/handler.ts`

---

### Tool Response _15:21:51_

<details><summary>Tool Result</summary>
...
</details>
```

### Why?

See <img width="1433" height="706" alt="image" src="https://github.com/user-attachments/assets/49b95b7d-4282-4bd0-9427-2e7c1bb6110b" />


## Development

### Plugin Structure

```
claude-transcript-plugin/
├── .claude-plugin/
│   ├── plugin.json          # Plugin manifest (name, version, metadata)
│   └── marketplace.json     # Marketplace catalog for distribution
├── hooks/
│   └── hooks.json           # PreCompact hook registration
├── scripts/
│   └── save-transcript.py   # JSONL → Markdown converter
├── skills/
│   └── save-transcript/
│       └── SKILL.md         # Manual trigger skill
├── CHANGELOG.md
├── README.md
└── LICENSE
```

### Key conventions

- Only `plugin.json` and `marketplace.json` go inside `.claude-plugin/` — everything else lives at the repo root.
- The script uses `${CLAUDE_PLUGIN_ROOT}` so paths resolve correctly when installed from the plugin cache.
- The hook always exits `0` on error so it never blocks compaction.

### Testing locally

Load the plugin from your local checkout without installing:

```bash
claude --plugin-dir /path/to/claude-transcript-plugin
```

You can also test the converter script directly against any existing JSONL transcript:

```bash
echo '{"session_id": "test", "transcript_path": "/path/to/transcript.jsonl", "cwd": "/your/project"}' \
  | python3 scripts/save-transcript.py
```

Output will appear in `~/.claude/transcripts/` (or `$TRANSCRIPT_OUTPUT_DIR`).

### Publishing

This repo doubles as both a plugin and a marketplace (via `.claude-plugin/marketplace.json`).

1. Push to GitHub (`IDisposable/claude-transcript-plugin`).
2. Users add the marketplace and install:
   ```bash
   /plugin marketplace add IDisposable/claude-transcript-plugin
   /plugin install transcript-saver@transcript-saver
   ```
3. To submit to the official Anthropic plugin directory, see [claude-plugins-official](https://github.com/anthropics/claude-plugins-official).

### Versioning

Bump the `version` field in `.claude-plugin/plugin.json` (and the matching version in `marketplace.json`) before each release. Claude Code uses this to detect updates — if the version doesn't change, users won't see the update.

## License

MIT
