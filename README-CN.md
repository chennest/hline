# hline

[![Latest Release](https://img.shields.io/github/v/release/chennest/hline.svg)](https://github.com/chennest/hline/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8.svg)](https://go.dev)
[![Downloads](https://img.shields.io/github/downloads/chennest/hline/total.svg)](https://github.com/chennest/hline/releases)
[![Stars](https://img.shields.io/github/stars/chennest/hline.svg)](https://github.com/chennest/hline/stargazers)

**[English](README.md)**

> `hcat` & `hsed` — 基于 hash 锚点的 CLI 工具，专为 AI 文件查看和编辑设计。

AI 编排文件编辑经常出错。`sed` 需要转义正则，`cat <<EOF` 没有完整性校验——多数失败不是模型的问题，而是编辑工具的问题。

**hline** 通过为每行标注内容 hash 解决这个问题。用 `hcat` 查看，用 `hsed` 编辑。零依赖，单二进制，SSH 优先。

设计思路参考了 [oh-my-pi](https://github.com/can1357/oh-my-pi) 和 [oh-my-openagent](https://github.com/can1357/oh-my-openagent)，感谢开源。

## 安装

### 快速安装（Linux / macOS）

```bash
# Linux amd64
curl -sL https://github.com/chennest/hline/releases/latest/download/hline-linux-amd64.tar.gz | sudo tar xz -C /usr/local/bin

# Linux arm64
curl -sL https://github.com/chennest/hline/releases/latest/download/hline-linux-arm64.tar.gz | sudo tar xz -C /usr/local/bin

# macOS arm64 (Apple Silicon)
curl -sL https://github.com/chennest/hline/releases/latest/download/hline-macos-arm64.tar.gz | sudo tar xz -C /usr/local/bin

# macOS amd64 (Intel)
curl -sL https://github.com/chennest/hline/releases/latest/download/hline-macos-amd64.tar.gz | sudo tar xz -C /usr/local/bin

# 国内加速（使用 ghfast 代理）
curl -sL https://ghfast.top/https://github.com/chennest/hline/releases/latest/download/hline-linux-amd64.tar.gz | sudo tar xz -C /usr/local/bin
```

### 包管理器安装

```bash
# 从 releases 下载 .deb
curl -LO https://github.com/chennest/hline/releases/latest/download/hline_linux_amd64.deb
sudo dpkg -i hline_linux_amd64.deb
```

### RHEL / Rocky / CentOS / Fedora

```bash
# 从 releases 下载 .rpm
curl -LO https://github.com/chennest/hline/releases/latest/download/hline_linux_amd64.rpm
sudo rpm -i hline_linux_amd64.rpm
```

### Arch Linux

```bash
# 从 releases 下载 .pkg.tar.zst
curl -LO https://github.com/chennest/hline/releases/latest/download/hline_linux_amd64.pkg.tar.zst
sudo pacman -U hline_linux_amd64.pkg.tar.zst
```

### macOS

```bash
# 从 releases 下载
curl -LO https://github.com/chennest/hline/releases/latest/download/hline-macos-arm64.tar.gz
tar xzf hline-macos-arm64.tar.gz
sudo mv hcat hsed /usr/local/bin/
```

### 从源码编译

```bash
git clone https://github.com/chennest/hline.git && cd hline
go build -o hcat ./cmd/hcat && go build -o hsed ./cmd/hsed
sudo mv hcat hsed /usr/local/bin/
```

## 工作原理

### hcat — 带 hash 锚点查看文件

每行标注 2 位字母 hash：

```
$ hcat /etc/nginx/nginx.conf 5-10

 5#VK| server {
 6#XJ|     listen 80;
 7#MB|     server_name example.com;
 8#QR|     root /var/www/html;
 9#TN| }
10#WS|
```

AI 读取输出后，通过 `6#XJ` 这样的锚点定位行进行编辑。hash 仅基于**行内容**计算——上方插入或删除行不会影响下方锚点。

### hsed — 通过锚点编辑

三种操作：

```bash
# 替换单行
hsed /etc/nginx/nginx.conf replace 6#XJ << 'EOF'
    listen 443 ssl;
EOF

# 替换范围
hsed /etc/nginx/nginx.conf replace 6#XJ 7#MB << 'EOF'
    listen 443 ssl;
    server_name new.example.com;
EOF

# 删除行（空内容）
hsed /etc/nginx/nginx.conf replace 6#XJ 7#MB << 'EOF'
EOF

# 在行后插入
hsed /etc/nginx/nginx.conf append 9#TN << 'EOF'

    location /api {
        proxy_pass http://127.0.0.1:3000;
    }
EOF

# 在行前插入
hsed /etc/nginx/nginx.conf prepend 5#VK << 'EOF'
# Managed by hline
EOF
```

### 批量编辑（原子）

一次 SSH 调用完成多处修改，全有全无语义。任一 hash 不匹配或冲突检测失败时，文件不会被改动。

```bash
hsed /etc/nginx/nginx.conf batch << 'EOF'
replace 6#XJ
    listen 443 ssl;
---
append 9#TN

    location /api {
        proxy_pass http://127.0.0.1:3000;
    }
---
prepend 5#VK
# Managed by hline
EOF
```

- **原子性** — 任一 hash 不匹配或冲突 → 文件不变，exit 1
- **冲突检测** — replace 范围不能重叠；append/prepend 目标不能落在 replace 区间内
- **`-p`** 同单条操作，支持 dry-run 预览

末尾 `---` 可选。空 replace 内容 = 删除对应行。

### Hash 校验

每次编辑时，`hsed` 会重新计算目标行的 hash：

- ✅ **匹配** → 执行编辑
- ❌ **不匹配** → 拒绝并显示当前状态：

```
ERROR: hash mismatch at line 6
  expected: 6#XJ
  current:  6#PM|     listen 8080;
```

AI 可以直接复制正确的锚点重试。

### 范围语法（hcat）

```bash
hcat file.conf              # 全文件
hcat file.conf 5-10         # 第 5-10 行
hcat file.conf 5 +10        # 从第 5 行起看 10 行
hcat file.conf 5            # 从第 5 行到末尾
hcat file.conf -A 3 5       # 第 5 行 + 后 3 行
hcat file.conf -B 2 -A 3 5  # 第 5 行 + 前 2 后 3
```

## 为什么需要 hline

| 问题 | hline 解决方案 |
|------|---------------|
| `sed` 正则转义噩梦 | 通过 hash 字面量匹配内容 |
| AI 无法验证文件状态 | hash 校验内容未变更 |
| 多次编辑竞态条件 | 每个锚点自带完整性检查 |
| 服务器无需额外框架 | 单二进制，SSH 优先设计 |

## Hash 算法

- 使用 [xxhash](https://github.com/zeebo/xxh3)，速度极快
- 映射到 26 字母表：`A-Z`
- 每行 2 位 hash（676 种组合，实际无冲突）
- 仅绑定行内容——不受其他位置插入/删除影响

## 对比

| | oh-my-pi Hashline | oh-my-openagent | **hline** |
|---|---|---|---|
| 形态 | Agent 框架 (Bun/TS) | OpenCode 插件 (Bun/TS) | **独立 CLI (Go)** |
| 安装 | npm 包 | npm 插件 | **单二进制 / deb / rpm** |
| 远程编辑 | 通过 agent | 通过 agent | **SSH + heredoc** |
| 依赖 | Bun 运行时 | Bun + OpenCode | **无** |
| 编辑接口 | Agent 工具调用 (JSON) | Agent 工具调用 (JSON) | **Stdin (heredoc)** |

## 致谢

感谢 [LINUX DO](https://linux.do) 开源社区。

## License

MIT

## AI Agent 集成

面向 AI Agent 的使用指南见 [`hline-usage/SKILL.md`](hline-usage/SKILL.md)，可直接放入 agent 的 skill 目录使用。

### Hermes Agent

```bash
mkdir -p ~/.hermes/skills/devops/hline-usage
cp hline-usage/SKILL.md ~/.hermes/skills/devops/hline-usage/
```

Hermes 自动发现 `~/.hermes/skills/` 下的 skill，下次会话生效。

### Claude Code

```bash
# 全局（所有项目可用）
mkdir -p ~/.claude/skills/hline-usage
cp hline-usage/SKILL.md ~/.claude/skills/hline-usage/

# 或项目级
mkdir -p .claude/skills/hline-usage
cp hline-usage/SKILL.md .claude/skills/hline-usage/
```

`.claude/skills/` 下的 skill 可通过 `/hline-usage` 调用或自动激活。

### OpenCode

```bash
# 全局
mkdir -p ~/.config/opencode/skills/hline-usage
cp hline-usage/SKILL.md ~/.config/opencode/skills/hline-usage/

# 或项目级（也支持 .agents/skills/ 和 .claude/skills/）
mkdir -p .opencode/skills/hline-usage
cp hline-usage/SKILL.md .opencode/skills/hline-usage/
```

OpenCode 从 `.opencode/skills/`、`.agents/skills/`、`.claude/skills/` 发现 skill，agent 按需通过 `skill` 工具加载。

### OpenAI Codex

```bash
# 全局（所有项目可用）
mkdir -p ~/.agents/skills/hline-usage
cp hline-usage/SKILL.md ~/.agents/skills/hline-usage/

# 或项目级
mkdir -p .agents/skills/hline-usage
cp hline-usage/SKILL.md .agents/skills/hline-usage/
```

Codex 自动发现 `.agents/skills/` 目录下的 skill，使用 `$hline-usage` 调用。

### OpenClaw

```bash
# 全局（managed skills）
mkdir -p ~/.openclaw/skills/hline-usage
cp hline-usage/SKILL.md ~/.openclaw/skills/hline-usage/

# 或 workspace 级
mkdir -p skills/hline-usage
cp hline-usage/SKILL.md skills/hline-usage/
```

OpenClaw 使用兼容 [AgentSkills](https://agentskills.io/) 的 skill 目录，下次会话生效。

### Cursor / Windsurf / 其他编辑器

将 `hline-usage/SKILL.md` 内容复制到项目的 `.cursorrules` 或对应的指令文件中。
