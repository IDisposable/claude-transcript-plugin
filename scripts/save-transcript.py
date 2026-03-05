#!/usr/bin/env python3
"""
Claude Code PreCompact hook: converts the current session's JSONL transcript
to a human-readable Markdown file before context gets compressed.

Default output: ~/.claude/transcripts/<timestamp>_<session-id>.md
Override via env: TRANSCRIPT_OUTPUT_DIR=/path/to/dir

Input (stdin JSON from Claude Code hook):
  { "session_id": "...", "transcript_path": "...", "cwd": "...", ... }
"""

import json
import os
import sys
from datetime import datetime
from pathlib import Path


# ---------------------------------------------------------------------------
# Formatting helpers
# ---------------------------------------------------------------------------

def format_tool_use(block):
    """Format a tool_use block as readable markdown."""
    name = block.get("name", "unknown")
    inp = block.get("input", {})

    if name == "Bash":
        cmd = inp.get("command", "")
        desc = inp.get("description", "")
        label = f" — {desc}" if desc else ""
        return f"**Tool: Bash**{label}\n```bash\n{cmd}\n```"
    elif name == "Read":
        fp = inp.get("file_path", "")
        return f"**Tool: Read** `{fp}`"
    elif name == "Write":
        fp = inp.get("file_path", "")
        content = inp.get("content", "")
        preview = content[:500] + ("..." if len(content) > 500 else "")
        return f"**Tool: Write** `{fp}`\n```\n{preview}\n```"
    elif name == "Edit":
        fp = inp.get("file_path", "")
        old = inp.get("old_string", "")[:200]
        new = inp.get("new_string", "")[:200]
        return f"**Tool: Edit** `{fp}`\n```diff\n- {old}\n+ {new}\n```"
    elif name == "Grep":
        pat = inp.get("pattern", "")
        path = inp.get("path", ".")
        return f"**Tool: Grep** pattern=`{pat}` path=`{path}`"
    elif name == "Glob":
        return f"**Tool: Glob** `{inp.get('pattern', '')}`"
    elif name == "Agent":
        desc = inp.get("description", "")
        stype = inp.get("subagent_type", "")
        return f"**Tool: Agent** ({stype}) {desc}"
    elif name == "WebSearch":
        return f"**Tool: WebSearch** `{inp.get('query', '')}`"
    elif name == "WebFetch":
        return f"**Tool: WebFetch** `{inp.get('url', '')}`"
    elif name == "Skill":
        return f"**Tool: Skill** `{inp.get('skill', '')}`"
    elif name == "AskUserQuestion":
        qs = inp.get("questions", [])
        parts = [f"**Question:** {q.get('question', '')}" for q in qs]
        return "\n".join(parts) if parts else f"**Tool: {name}**"
    else:
        return f"**Tool: {name}**"


def format_tool_result(block):
    """Format a tool_result block compactly."""
    tc = block.get("content", "")
    if isinstance(tc, list):
        texts = [t.get("text", "") for t in tc if t.get("type") == "text"]
        result_text = "\n".join(texts)
    elif isinstance(tc, str):
        result_text = tc
    else:
        result_text = str(tc)

    if not result_text.strip():
        return None

    # Truncate very long results
    if len(result_text) > 2000:
        result_text = result_text[:2000] + "\n... (truncated)"
    return (
        "<details><summary>Tool Result</summary>\n\n"
        f"```\n{result_text}\n```\n"
        "</details>"
    )


# ---------------------------------------------------------------------------
# Noise filtering
# ---------------------------------------------------------------------------

NOISE_PREFIXES = (
    "<local-command-caveat>",
    "<local-command-stdout>",
    "<command-name>/exit",
    "<command-name>/plugin",
    "<command-name>/help",
    "<command-name>/clear",
)


def is_system_noise(text):
    """Return True for system/command noise that isn't real conversation."""
    stripped = text.strip()
    return any(stripped.startswith(p) for p in NOISE_PREFIXES)


# ---------------------------------------------------------------------------
# Message classification
# ---------------------------------------------------------------------------

def classify_user_message(content):
    """Classify a user message as human input, tool result, or noise.

    Returns (kind, formatted_text) where kind is one of:
      "human", "tool_result", "mixed", "system", "noise"
    """
    if isinstance(content, str):
        if is_system_noise(content):
            return "noise", None
        if content.strip().startswith("<system-reminder>"):
            return "system", None
        return "human", content

    if isinstance(content, list):
        has_tool_result = any(b.get("type") == "tool_result" for b in content)
        has_text = any(b.get("type") == "text" for b in content)

        if has_tool_result and not has_text:
            parts = []
            for b in content:
                if b.get("type") == "tool_result":
                    fmt = format_tool_result(b)
                    if fmt:
                        parts.append(fmt)
            return "tool_result", ("\n\n".join(parts) if parts else None)

        if has_text and not has_tool_result:
            texts = [b.get("text", "") for b in content if b.get("type") == "text"]
            combined = "\n".join(texts)
            if is_system_noise(combined):
                return "noise", None
            return "human", combined

        # Mixed content
        parts = []
        for b in content:
            if b.get("type") == "text":
                t = b.get("text", "")
                if not is_system_noise(t):
                    parts.append(t)
            elif b.get("type") == "tool_result":
                fmt = format_tool_result(b)
                if fmt:
                    parts.append(fmt)
        return "mixed", ("\n\n".join(parts) if parts else None)

    return "noise", None


def format_assistant_content(content):
    """Format assistant message content to readable markdown."""
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        parts = []
        for block in content:
            btype = block.get("type", "")
            if btype == "text":
                text = block.get("text", "")
                if text.strip():
                    parts.append(text)
            elif btype == "tool_use":
                parts.append(format_tool_use(block))
            elif btype == "thinking":
                thinking = block.get("thinking", "")
                if thinking:
                    preview = thinking[:500]
                    if len(thinking) > 500:
                        preview += "..."
                    parts.append(
                        "<details><summary>Thinking</summary>\n\n"
                        f"{preview}\n"
                        "</details>"
                    )
        return "\n\n".join(parts)
    return str(content)


def format_timestamp(ts):
    """Format ISO timestamp to short HH:MM:SS string."""
    if not ts:
        return ""
    try:
        dt = datetime.fromisoformat(ts.replace("Z", "+00:00"))
        return f" _{dt.strftime('%H:%M:%S')}_"
    except (ValueError, TypeError):
        return ""


# ---------------------------------------------------------------------------
# Main conversion
# ---------------------------------------------------------------------------

def convert_transcript(jsonl_path, output_path, cwd):
    """Convert a Claude Code JSONL transcript to readable Markdown."""
    with open(jsonl_path, "r") as f:
        lines = f.readlines()

    project_name = Path(cwd).name if cwd else "unknown"
    session_start = None
    current_branch = ""

    with open(output_path, "w") as out:
        out.write(f"# Session Transcript: {project_name}\n\n")

        # Find session start time
        for line in lines:
            try:
                obj = json.loads(line.strip())
                ts = obj.get("timestamp", "")
                if ts:
                    session_start = ts
                    break
            except json.JSONDecodeError:
                continue

        if session_start:
            out.write(f"**Started:** {session_start}  \n")
        out.write(f"**Project:** `{cwd}`  \n")
        out.write(f"**Transcript source:** `{jsonl_path}`  \n\n")
        out.write("---\n\n")

        for line in lines:
            line = line.strip()
            if not line:
                continue
            try:
                obj = json.loads(line)
            except json.JSONDecodeError:
                continue

            msg_type = obj.get("type", "")
            timestamp = obj.get("timestamp", "")
            branch = obj.get("gitBranch", "")

            # Track branch changes
            if branch and branch != current_branch:
                current_branch = branch
                out.write(f"> *Branch: `{branch}`*\n\n")

            if msg_type == "user":
                if obj.get("isMeta"):
                    continue
                msg = obj.get("message", {})
                content = msg.get("content", "")
                kind, text = classify_user_message(content)

                if kind in ("noise", "system") or text is None:
                    continue

                ts_label = format_timestamp(timestamp)

                if kind == "human":
                    out.write(f"## User{ts_label}\n\n{text}\n\n---\n\n")
                elif kind == "tool_result":
                    out.write(
                        f"### Tool Response{ts_label}\n\n{text}\n\n---\n\n"
                    )
                elif kind == "mixed":
                    out.write(f"## User{ts_label}\n\n{text}\n\n---\n\n")

            elif msg_type == "assistant":
                msg = obj.get("message", {})
                content = msg.get("content", "")
                text = format_assistant_content(content)
                if not text.strip():
                    continue
                ts_label = format_timestamp(timestamp)
                out.write(f"## Claude{ts_label}\n\n{text}\n\n---\n\n")


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

def main():
    try:
        hook_input = json.load(sys.stdin)
    except (json.JSONDecodeError, EOFError):
        hook_input = {}

    session_id = hook_input.get("session_id", "unknown")
    transcript_path = hook_input.get("transcript_path", "")
    cwd = hook_input.get("cwd", "")

    if not transcript_path or not os.path.exists(transcript_path):
        print(
            f"transcript-saver: no transcript found at {transcript_path}",
            file=sys.stderr,
        )
        sys.exit(0)  # Exit cleanly — never block compaction

    # Output directory: env override or default
    output_dir = Path(
        os.environ.get(
            "TRANSCRIPT_OUTPUT_DIR",
            str(Path.home() / ".claude" / "transcripts"),
        )
    )
    output_dir.mkdir(parents=True, exist_ok=True)

    timestamp = datetime.now().strftime("%Y-%m-%d_%H%M%S")
    short_id = session_id[:8] if session_id != "unknown" else "none"
    output_file = output_dir / f"{timestamp}_{short_id}.md"

    try:
        convert_transcript(transcript_path, str(output_file), cwd)
        print(f"transcript-saver: saved {output_file}", file=sys.stderr)
    except Exception as e:
        print(f"transcript-saver: error — {e}", file=sys.stderr)
        sys.exit(0)  # Don't block compaction on error


if __name__ == "__main__":
    main()
