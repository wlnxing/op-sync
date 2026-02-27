# op-sync

[![Release](https://img.shields.io/github/v/release/wlnxing/op-sync?label=release)](https://github.com/wlnxing/op-sync/releases/latest)
[![Release Build](https://github.com/wlnxing/op-sync/actions/workflows/release.yml/badge.svg)](https://github.com/wlnxing/op-sync/actions/workflows/release.yml)
[![Go](https://img.shields.io/badge/go-1.22%2B-00ADD8?logo=go)](https://go.dev/)

把 OpenList 里的一个目录，同步到另一个目录的命令行工具。

采用逐个文件比对，再按文件复制的方式同步，不会直接整目录复制。

## 同步规则

- 只同步需要更新的文件，不全量重传
- 目标没有该文件：复制
- 同名文件且源文件更大：覆盖
- 同名文件且源文件不更大：跳过
- 目标缺少子目录：自动创建
- 如果 OpenList 里已有相同复制任务在进行：跳过
- 命中黑名单通配符的文件/路径：不参与同步

## 适用场景

- 两个目录做日常增量同步
- 只想补新文件或覆盖更大的源文件
- 想排除临时文件、缓存目录等

## 快速上手

1. 前往 [Releases](https://github.com/wlnxing/op-sync/releases/latest) 下载对应系统的最新包
2. 解压并进入解压目录（Linux示例）：

```bash
tar -xzf openlist-sync-linux-amd64.tar.gz
cd openlist-sync-linux-amd64
```

3. 基于示例文件准备运行配置：

```bash
cp config.example.json config.json
cp token.example.txt token.txt
```

4. 编辑 `config.json`，至少确认以下字段：
- `base_url`：OpenList 地址
- `src`：源目录
- `dst`：目标目录

5. 在 `token.txt` 中填入 OpenList token，然后执行：

```bash
chmod +x ./openlist-sync
./openlist-sync
```

## 常用命令

```bash
# 使用默认配置（当前目录 config.json）
./openlist-sync

# 指定配置文件
./openlist-sync --config /path/to/config.json

# 先预览，不真正复制
./openlist-sync --config ./config.json -dry-run -log-level info
```

## 配置文件示例

```json
{
  "base_url": "http://localhost:35244",
  "token_file": "token.txt",
  "src": "/test/source",
  "dst": "/test/target",
  "blacklist": [
    "*.tmp",
    ".DS_Store",
    "cache/*"
  ],
  "log_level": "info",
  "per_page": 0,
  "timeout": "30s",
  "dry_run": false
}
```

## 参数（可选）

- `--config`：配置文件路径，默认 `./config.json`
- `-src`：源目录
- `-dst`：目标目录
- `-base-url`：OpenList 地址，默认 `http://localhost:35244`
- `-token-file`：token 文件路径，默认 `token.txt`
- `-exclude`：黑名单通配符，可重复传，或用逗号分隔
- `-dry-run`：只看计划，不执行复制
- `-log-level`：`debug | info | error`，默认 `info`
- `-per-page`：列表分页，默认 `0`（让 OpenList 返回目录全部文件）
- `-timeout`：单次 API 请求超时，默认 `30s`

说明：
- 参数优先级：`命令行 > config.json > 默认值`
- `debug` 会显示每个文件的详细计划
- 黑名单规则：
  - 不含 `/` 的模式（如 `*.tmp`）按文件名匹配
  - 含 `/` 的模式（如 `cache/*`）按相对路径匹配

## 编译

```bash
# 本机构建
make build

# 交叉编译（mac arm64, linux amd64, linux arm64）
make cross
```
