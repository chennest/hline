---
name: hline-usage
description: "Use when reading or editing files via SSH on a remote server that has hline (hcat + hsed) installed. Hash-anchored file operations for AI agents."
version: 0.1.0
author: chennest
license: MIT
---

# hline Usage Guide for AI Agents

## Overview

**hline** = `hcat` (view) + `hsed` (edit). Hash-anchored file operations for SSH environments. Zero dependencies, single binary.

- **hcat** — Read files with hash anchors on each line
- **hsed** — Edit files via those anchors (replace / append / prepend)

**Repo:** https://github.com/chennest/hline | **Install:** see [README](https://github.com/chennest/hline#install)

## When to Use

- SSH access to a remote server with `hcat`/`hsed` installed
- Need to read/edit files unreachable by native agent tools
- `sed` is too fragile for the edit

**Do NOT use for:** local files (use native tools), binary files, non-UTF-8 content.

## How It Works

Each line gets a 2-char hash anchor based on its content (not position). Hash binds to content only — insertions/deletions elsewhere won't invalidate anchors on untouched lines.

Format: `{line_number}#{hash}|{content}`

```
11#XJ| function hello() {
12#MB|   return "world";
13#QR| }
```

## hcat — View Files

```bash
hcat /path/to/file              # Full file
hcat /path/to/file 5-10         # Lines 5-10
hcat /path/to/file 5            # Line 5 to EOF
hcat /path/to/file 5 +10        # Line 5 + next 10 lines
hcat /path/to/file -A 3 5       # Line 5 + 3 after (space required)
hcat /path/to/file -B 2 -A 3 5  # Line 5 + 2 before + 3 after
```

⚠️ `-A`/`-B` require space: `-A 3` ✓, `-A3` ✗

## hsed — Edit Files

Usage: `hsed <file> <operation> <anchor(s)> << 'EOF'` then content, then `EOF`.

```bash
# Replace single line
hsed /path/to/file replace 11#XJ << 'EOF'
new content
EOF

# Replace range (start anchor, end anchor)
hsed /path/to/file replace 11#XJ 13#QR << 'EOF'
multi-line replacement
EOF

# Delete range (empty content)
hsed /path/to/file replace 11#XJ 13#QR << 'EOF'
EOF

# Insert after a line
hsed /path/to/file append 13#QR << 'EOF'
inserted content
EOF

# Insert before a line
hsed /path/to/file prepend 11#XJ << 'EOF'
inserted content
EOF

# Preview (dry-run, no write)
hsed -p /path/to/file replace 11#XJ << 'EOF'
new content
EOF
```

⚠️ Range anchors must be ascending (end after start).
⚠️ If edit content contains `EOF`, use a different delimiter like `ENDOFEDIT`.

## Typical Workflow

1. `hcat /etc/nginx/nginx.conf` — view file with anchors
2. Copy target anchor(s), e.g. `15#AB`
3. `hsed /etc/nginx/nginx.conf replace 15#AB << 'EOF'` — edit
4. `hcat /etc/nginx/nginx.conf 10-20` — verify the change

## Common Pitfalls

1. **Hash mismatch** — File changed between view and edit. `hsed` outputs the current correct anchor so you can retry immediately.
2. **EOF on its own line** — Terminator must have no leading/trailing whitespace.
3. **Range order** — End anchor must be after start anchor.
4. **-A/-B need space** — `-A 3` ✓, `-A3` ✗.
