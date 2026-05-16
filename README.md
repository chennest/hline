# hline

[![Latest Release](https://img.shields.io/github/v/release/chennest/hline.svg)](https://github.com/chennest/hline/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8.svg)](https://go.dev)

**[中文文档](README-CN.md)**

> `hcat` & `hsed` — Hash-anchored CLI tools for AI-friendly file viewing and editing.

AI agents struggle with file editing. `sed` requires regex escaping, `cat <<EOF` has no integrity checks — most agent failures aren't the model's fault, they're the edit tool's fault.

**hline** solves this by tagging every line with a content hash. View with `hcat`, edit with `hsed`. Zero dependencies, single binary, SSH-first.

Inspired by [oh-my-pi](https://github.com/can1357/oh-my-pi) and [oh-my-openagent](https://github.com/can1357/oh-my-openagent).

## Install

### Quick Install (Linux / macOS)

```bash
# Linux amd64
curl -sL https://github.com/chennest/hline/releases/latest/download/hline-linux-amd64.tar.gz | sudo tar xz -C /usr/local/bin

# Linux arm64
curl -sL https://github.com/chennest/hline/releases/latest/download/hline-linux-arm64.tar.gz | sudo tar xz -C /usr/local/bin

# macOS arm64 (Apple Silicon)
curl -sL https://github.com/chennest/hline/releases/latest/download/hline-macos-arm64.tar.gz | sudo tar xz -C /usr/local/bin

# macOS amd64 (Intel)
curl -sL https://github.com/chennest/hline/releases/latest/download/hline-macos-amd64.tar.gz | sudo tar xz -C /usr/local/bin

# China (use ghfast proxy)
curl -sL https://ghfast.top/https://github.com/chennest/hline/releases/latest/download/hline-linux-amd64.tar.gz | sudo tar xz -C /usr/local/bin
```

### Package Install

```bash
# Download .deb from releases
curl -LO https://github.com/chennest/hline/releases/latest/download/hline_linux_amd64.deb
sudo dpkg -i hline_linux_amd64.deb
```

### RHEL / Rocky / CentOS / Fedora

```bash
# Download .rpm from releases
curl -LO https://github.com/chennest/hline/releases/latest/download/hline_linux_amd64.rpm
sudo rpm -i hline_linux_amd64.rpm
```

### Arch Linux

```bash
# Download .pkg.tar.zst from releases
curl -LO https://github.com/chennest/hline/releases/latest/download/hline_linux_amd64.pkg.tar.zst
sudo pacman -U hline_linux_amd64.pkg.tar.zst
```

### macOS

```bash
# Download from releases
curl -LO https://github.com/chennest/hline/releases/latest/download/hline-macos-arm64.tar.gz
tar xzf hline-macos-arm64.tar.gz
sudo mv hcat hsed /usr/local/bin/
```

### From Source

```bash
git clone https://github.com/chennest/hline.git && cd hline
go build -o hcat ./cmd/hcat && go build -o hsed ./cmd/hsed
sudo mv hcat hsed /usr/local/bin/
```

## How It Works

### hcat — View with Hash Anchors

Every line gets a 2-letter content hash:

```
$ hcat /etc/nginx/nginx.conf 5-10

 5#VK| server {
 6#XJ|     listen 80;
 7#MB|     server_name example.com;
 8#QR|     root /var/www/html;
 9#TN| }
10#WS|
```

The AI reads this output and references anchors like `6#XJ` when editing. The hash is computed from **line content only** — insertions and deletions above don't invalidate anchors below.

### hsed — Edit by Anchor

Three operations:

```bash
# Replace a single line
HLINE_FILE=/etc/nginx/nginx.conf hsed replace 6#XJ << 'EOF'
    listen 443 ssl;
EOF

# Replace a range
HLINE_FILE=/etc/nginx/nginx.conf hsed replace 6#XJ 7#MB << 'EOF'
    listen 443 ssl;
    server_name new.example.com;
EOF

# Delete lines (empty content)
HLINE_FILE=/etc/nginx/nginx.conf hsed replace 6#XJ 7#MB << 'EOF'
EOF

# Insert after a line
HLINE_FILE=/etc/nginx/nginx.conf hsed append 9#TN << 'EOF'

    location /api {
        proxy_pass http://127.0.0.1:3000;
    }
EOF

# Insert before a line
HLINE_FILE=/etc/nginx/nginx.conf hsed prepend 5#VK << 'EOF'
# Managed by hline
EOF
```

### Hash Validation

On every edit, `hsed` recomputes the hash for the target line(s):

- ✅ **Match** → Edit applied
- ❌ **Mismatch** → Rejected with current state:

```
ERROR: hash mismatch at line 6
  expected: 6#XJ
  current:  6#PM|     listen 8080;
```

The AI can copy the correct anchor directly and retry.

### Range Syntax (hcat)

```bash
hcat file.conf              # Full file
hcat file.conf 5-10         # Lines 5-10
hcat file.conf 5 +10        # 10 lines starting from line 5
hcat file.conf 5            # From line 5 to EOF
hcat file.conf -A 3 5       # Line 5 + 3 lines after
hcat file.conf -B 2 -A 3 5  # Line 5 + 2 before + 3 after
```

## Why

| Problem | hline Solution |
|---------|---------------|
| `sed` regex escaping nightmare | Literal content match via hash |
| AI can't verify file state | Hash validates content hasn't changed |
| Multi-edit race conditions | Each anchor is an integrity check |
| No framework needed on server | Single binary, SSH-first design |

## Hash Algorithm

- Uses [xxhash](https://github.com/zeebo/xxh3) for speed
- Maps to 26-letter alphabet: `A-Z`
- 2-letter hash per line (676 combinations, collision-free in practice)
- Content-only binding — resilient to insertions/deletions elsewhere

## Comparison

| | oh-my-pi Hashline | oh-my-openagent | **hline** |
|---|---|---|---|
| Form factor | Agent framework (Bun/TS) | OpenCode plugin (Bun/TS) | **Standalone CLI (Go)** |
| Install | npm package | npm plugin | **Single binary / deb / rpm** |
| Remote editing | Via agent | Via agent | **SSH + heredoc** |
| Dependencies | Bun runtime | Bun + OpenCode | **None** |
| Edit interface | Agent tool call (JSON) | Agent tool call (JSON) | **Stdin (heredoc)** |

## License

MIT
