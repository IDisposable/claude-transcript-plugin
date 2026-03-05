---
name: save-transcript
description: Manually save a Markdown transcript of the current session. Use when the user asks to save, export, or snapshot the conversation.
---

# Save Transcript

Save the current session's conversation to a readable Markdown file.

## Steps

1. Find the current session's JSONL transcript file. It is located at the path shown in the session metadata, typically under `~/.claude/projects/`.

2. Run the transcript converter script:
   ```bash
   echo '{"session_id": "<session-id>", "transcript_path": "<path-to-jsonl>", "cwd": "<current-working-dir>"}' | python3 ${CLAUDE_PLUGIN_ROOT}/scripts/save-transcript.py
   ```

3. Report the output file path to the user.

## Notes

- Transcripts are saved to `~/.claude/transcripts/` by default.
- Override with `TRANSCRIPT_OUTPUT_DIR` environment variable.
- The hook also runs automatically before every context compaction.
