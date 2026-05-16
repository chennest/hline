# hline

> `hcat` & `hsed` — hash-anchored CLI tools for AI-friendly file viewing and editing.

AI agents struggle with file editing. `sed` requires regex escaping, `cat <<EOF` has no integrity checks, and most agent failures aren't the model's fault — they're the edit tool's fault.

**hline** solves this by tagging every line with a content hash. View with `hcat`, edit with `hsed`. Zero dependencies, single binary, SSH-first.

设计思路参考了 [oh-my-pi](https://github.com/can1357/oh-my-pi) 和 [oh-my-openagent](https://github.com/can1357/oh-my-openagent) 两个项目，感谢开源。

## Install

```bash
# Download binary (Linux amd64)
curl -fsSL https://github.com/chennest/hline/releases/latest/download/hline-linux-amd64.tar.gz | tar xz -C /usr/local/bin

# Or build from source
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
hsed replace 6#XJ << 'EOF'
    listen 443 ssl;
EOF

# Replace a range
hsed replace 6#XJ 7#MB << 'EOF'
    listen 443 ssl;
    server_name new.example.com;
EOF

# Delete lines (empty content)
hsed replace 6#XJ 7#MB << 'EOF'
EOF

# Insert after a line
hsed append 9#TN << 'EOF'

    location /api {
        proxy_pass http://127.0.0.1:3000;
    }
EOF

# Insert before a line
hsed prepend 5#VK << 'EOF'
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
  current:  6#PM|     listen 8080;    proxy_pass http://127.0.0.1:3000;
EOF
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
| Install | npm package | npm plugin | **Single binary** |
| Remote editing | Via agent | Via agent | **SSH + heredoc** |
| Dependencies | Bun runtime | Bun + OpenCode | **None** |
| Edit interface | Agent tool call (JSON) | Agent tool call (JSON) | **Stdin (heredoc)** |

## License

MIT
