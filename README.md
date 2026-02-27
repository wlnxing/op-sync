# op-sync

一个基于 Go 的 OpenList 目录增量同步工具。  
它不会直接调用“复制文件夹”，而是先对比源/目标目录中的文件，再按文件调用 OpenList 的复制接口。

## 功能

- 递归扫描源目录和目标目录（通过 `/api/fs/list`）
- 按“相对路径 + 文件大小”判断是否需要同步
- 同名文件时：
  - 源文件更大：覆盖复制
  - 否则：跳过
- 复制前检查未完成复制任务（`/api/admin/task/copy/undone`）
  - 如果发现相同 `源文件 -> 目标目录` 任务正在进行，跳过提交
- 按文件调用 `/api/fs/copy`（`names` 每次一个文件）
- 目标目录不存在时自动创建

## 要求

- Go 1.22+
- 可访问的 OpenList 实例（默认 `http://localhost:35244`）
- 有效 token（默认从 `token.txt` 读取）
- 配置文件（默认当前目录 `config.json`）

## 快速开始

```bash
# 使用当前目录 config.json
go run ./cmd/openlist-sync

# 指定配置文件路径
go run ./cmd/openlist-sync --config /path/to/config.json

# 命令行参数会覆盖配置文件
go run ./cmd/openlist-sync --config ./config.json -dry-run -log-level info
```

## 配置文件

默认读取当前目录 `config.json`，也可以通过 `--config` 指定路径。

示例：

```json
{
  "base_url": "http://localhost:35244",
  "token_file": "token.txt",
  "src": "/test/source",
  "dst": "/test/target",
  "log_level": "error",
  "per_page": 0,
  "timeout": "30s",
  "dry_run": false
}
```

## 参数

- `--config`：配置文件路径，默认 `./config.json`
- `-src`：源目录（若配置文件里未提供则必填）
- `-dst`：目标目录（若配置文件里未提供则必填）
- `-base-url`：OpenList 地址，默认 `http://localhost:35244`
- `-token-file`：token 文件路径，默认 `token.txt`
- `-log-level`：日志级别，可选 `debug|info|error`，默认 `error`
- `-per-page`：列表分页大小，默认 `0`（让 OpenList 返回该目录全部文件）
- `-timeout`：HTTP 超时，默认 `30s`
- `-dry-run`：只输出同步计划，不提交复制

说明：
- 参数优先级：`命令行参数 > 配置文件 > 内置默认值`。
- 默认 `error` 级别只打印错误日志。
- 如果希望看到过程日志，请设置 `-log-level info`。
- 如果希望看到每个文件的 `PLAN` 明细，请设置 `-log-level debug`。

## 同步规则说明

对每个源文件（相对路径为 key）：

1. 目标不存在该文件：复制
2. 目标存在同名文件且源文件更大：覆盖复制
3. 其他情况：跳过

注意：
- 当前仅比较文件名（相对路径）和大小，不比较 hash、修改时间。
- 目标目录中多余文件不会删除。

## 退出码

- `0`：执行成功
- `1`：执行过程中有失败项
- `2`：参数错误或初始化失败（如 token 读取失败）

## 开发与测试

```bash
go test ./...
go build ./...
```

## 编译

```bash
# 本机构建
make build

# 交叉编译（mac arm64, linux amd64, linux arm64）
make cross
```
