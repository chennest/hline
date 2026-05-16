# hline

> `hcat` & `hsed` — 基于 hash 锚点的 CLI 工具，专为 AI 文件查看和编辑设计。

AI 编排文件编辑经常出错。`sed` 需要转义正则，`cat <<EOF` 没有完整性校验——多数失败不是模型的问题，而是编辑工具的问题。

**hline** 通过为每行标注内容 hash 解决这个问题。用 `hcat` 查看，用 `hsed` 编辑。零依赖，单二进制，SSH 优先。

设计思路参考了 [oh-my-pi](https://github.com/can1357/oh-my-pi) 和 [oh-my-openagent](https://github.com/can1357/oh-my-openagent) 两个项目，感谢开源。

## 安装

```bash
# 下载二进制（Linux amd64）
curl -fsSL https://github.com/chennest/hline/releases/latest/download/hline-linux-amd64.tar.gz | tar xz -C /usr/local/bin

# 或从源码编译
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
hsed replace 6#XJ << 'EOF'
    listen 443 ssl;
EOF

# 替换范围
hsed replace 6#XJ 7#MB << 'EOF'
    listen 443 ssl;
    server_name new.example.com;
EOF

# 删除行（空内容）
hsed replace 6#XJ 7#MB << 'EOF'
EOF

# 在行后插入
hsed append 9#TN << 'EOF'

    location /api {
        proxy_pass http://127.0.0.1:3000;
    }
EOF

# 在行前插入
hsed prepend 5#VK << 'EOF'
# Managed by hline
EOF
```

### Hash 校验

每次编辑时，`hsed` 会重新计算目标行的 hash：

- ✅ **匹配** → 执行编辑
- ❌ **不匹配** → 拒绝并显示当前状态：

```
ERROR: hash mismatch at line 6
  expected: 6#XJ
  current:  6#PM|     listen 8080;    proxy_pass http://127.0.0.1:3000;
EOF
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
| 安装 | npm 包 | npm 插件 | **单二进制** |
| 远程编辑 | 通过 agent | 通过 agent | **SSH + heredoc** |
| 依赖 | Bun 运行时 | Bun + OpenCode | **无** |
| 编辑接口 | Agent 工具调用 (JSON) | Agent 工具调用 (JSON) | **Stdin (heredoc)** |

## License

MIT
