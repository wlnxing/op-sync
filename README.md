# op-sync

`op-sync` 是一个给 OpenList 用的“文件夹同步小工具”。

你可以把它理解成：  
把 A 目录里的文件，增量同步到 B 目录。

## 这个工具适合做什么

- 想定时/手动把一个目录同步到另一个目录
- 不想每次都全部复制
- 想尽量避免重复任务和无意义覆盖
- 复制是按“文件”执行的，不是一次性整目录复制
- 只处理“需要更新”的文件，不会每次全量重传
- 同名文件时：
  - 源文件更大：覆盖目标文件
  - 源文件不更大：跳过
- 目标目录没有的文件：自动复制过去
- 目标目录没有的子目录：自动创建
- 如果 OpenList 里已经有相同复制任务在跑：不重复提交
- 支持黑名单（通配符）：命中的文件/路径不参与同步

## 快速上手

1. 准备 `config.json`（可参考仓库内 `config.example.json`）
2. 准备 token 文件（可参考 `token.example.txt`）
3. 运行：

```bash
go run ./cmd/openlist-sync
```

## 常用命令

```bash
# 使用默认配置（当前目录 config.json）
go run ./cmd/openlist-sync

# 指定配置文件
go run ./cmd/openlist-sync --config /path/to/config.json

# 先预览，不真正复制
go run ./cmd/openlist-sync --config ./config.json -dry-run -log-level info
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
  "log_level": "error",
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
