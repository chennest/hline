---
name: hline-usage
description: "Use when reading or editing files via SSH on a remote server that has hline (hcat + hsed) installed. Hash-anchored file operations for AI agents."
version: 0.1.0
author: chennest
license: MIT
---

# hline Usage Guide for AI Agents

## Overview

**hline** is a pair of CLI tools (`hcat` + `hsed`) for hash-anchored file viewing and editing, designed for AI agents operating via SSH.

- **hcat** — Read files with hash anchors attached to each line
- **hsed** — Edit files using those hash anchors (replace / append / prepend)

**Repo:** https://github.com/chennest/hline

## When to Use

- Agent has SSH access to a remote server with `hcat`/`hsed` installed
- Need to read or edit files that `read_file`/`write_file` can't reach
- `sed` is too fragile for the edit at hand

**Do NOT use for:**
- Local files where native agent tools work fine
- Binary files or files with non-UTF-8 content

## How It Works

Each line gets a 2-character hash anchor computed from its content (not line number):

```
11#XJ| function hello() {
12#MB|   return "world";
13#QR| }
```

Format: `{line_number}#{hash}|{content}`

- The hash binds to content, not position — inserting/deleting lines won't invalidate anchors on untouched lines
- Hash collision rate: ~0.15% (26² = 676 combinations, 2-char letters A-Z)

## hcat — View Files

```bash
# View entire file
hcat /path/to/file.conf

# Lines 5-10
hcat /path/to/file.conf 5-10

# From line 5 to end
hcat /path/to/file.conf 5

# Line 5 + next 10 lines
hcat /path/to/file.conf 5 +10

# Line 5 with context (-A after, -B before, space required)
hcat /path/to/file.conf -A 3 5
hcat /path/to/file.conf -B 2 -A 3 5
```

**Important:** `-A`/`-B` require space: `-A 3`, NOT `-A3`.

## hsed — Edit Files

Usage: `hsed <file> <operation> <anchor(s)> << 'EOF'` followed by content, then `EOF`.

### replace — Replace line(s)

```bash
# Single line
hsed /path/to/file.conf replace 11#XJ << 'EOF'
  console.log("hi");
EOF

# Range replace
hsed /path/to/file.conf replace 11#XJ 13#QR << 'EOF'
  return "hello world";
EOF

# Delete (empty content)
hsed /path/to/file.conf replace 11#XJ 13#QR << 'EOF'
EOF
```

### append — Insert after line

```bash
hsed /path/to/file.conf append 13#QR << 'EOF'

function added() {
  return true;
}
EOF
```

### prepend — Insert before line

```bash
hsed /path/to/file.conf prepend 11#XJ << 'EOF'
// comment before
EOF
```

### Preview mode (dry-run)

Add `-p` or `--preview` to validate without writing:

```bash
hsed -p /path/to/file.conf replace 11#XJ << 'EOF'
  new content
EOF
```

## Typical Workflow

1. `hcat /etc/nginx/nginx.conf` — view file with anchors
2. Identify target line(s) and copy anchor(s), e.g. `15#AB`
3. `hsed /etc/nginx/nginx.conf replace 15#AB << 'EOF'` — edit using anchor
4. `hcat /etc/nginx/nginx.conf 10-20` — verify the change

## Common Pitfalls

1. **Hash mismatch on edit** — If the file changed between view and edit, the hash won't match. `hsed` outputs the current correct `line#hash|content` so you can retry immediately.

2. **Trailing newline in heredoc** — The `EOF` terminator must be on its own line with no leading/trailing whitespace.

3. **Range anchors must be in order** — `hsed file replace 13#QR 11#XJ` will fail. End anchor must be after start anchor.

4. **Don't use `-A3` without space** — `hcat file -A 3 5` ✓, `hcat file -A3 5` ✗.

5. **heredoc content includes EOF** — If your edit content contains the string `EOF`, use a different delimiter like `ENDOFEDIT`.

## Quick Install

```bash
# Linux amd64
curl -sL https://github.com/chennest/hline/releases/latest/download/hline-linux-amd64.tar.gz | sudo tar xz -C /usr/local/bin

# Linux arm64
curl -sL https://github.com/chennest/hline/releases/latest/download/hline-linux-arm64.tar.gz | sudo tar xz -C /usr/local/bin

# macOS arm64 (Apple Silicon)
curl -sL https://github.com/chennest/hline/releases/latest/download/hline-macos-arm64.tar.gz | sudo tar xz -C /usr/local/bin

# macOS amd64 (Intel)
curl -sL https://github.com/chennest/hline/releases/latest/download/hline-macos-amd64.tar.gz | sudo tar xz -C /usr/local/bin

# Windows amd64
curl -sL https://github.com/chennest/hline/releases/latest/download/hline-windows-amd64.zip -o hline.zip && Expand-Archive -Path hline.zip -DestinationPath $env:USERPROFILE\bin -Force; Remove-Item hline.zip

# China proxy (replace OS/arch as needed)
curl -sL https://ghfast.top/https://github.com/chennest/hline/releases/latest/download/hline-linux-amd64.tar.gz | sudo tar xz -C /usr/local/bin
```

Package managers also available: `.deb` (Debian/Ubuntu), `.rpm` (RHEL/CentOS), `.pkg.tar.zst` (Arch Linux). See [README](https://github.com/chennest/hline) for details.

## Verification Checklist

- [ ] `hcat` and `hsed` are on PATH: `hcat -v`
- [ ] File is UTF-8 text (not binary)
- [ ] Anchors copied exactly from `hcat` output (no extra spaces)
- [ ] `hsed` heredoc delimiter `EOF` doesn't appear in edit content
- [ ] After edit, verify with `hcat` at the relevant line range
