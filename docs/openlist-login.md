# OpenList 登录脚本

`openlist-login.sh` 用于调用 OpenList 登录接口获取登录 token，并把 token 写入文件。

脚本默认使用 OpenList 的杂凑密码登录接口：

```text
POST /api/auth/login/hash
```

## 基本用法

```bash
./openlist-login.sh --url http://localhost:35244 -u admin -p 'your-password'
```

登录成功后：

- stdout 输出 token 本体
- stderr 输出 token 写入位置
- 默认写入 `./token.txt`

指定 token 文件：

```bash
./openlist-login.sh \
  --url http://localhost:35244 \
  -u admin \
  -p 'your-password' \
  --token-file ./token.txt
```

## 参数

- `--url`, `--base-url`：OpenList 地址
- `-u`, `--username`：OpenList 用户名
- `-p`, `--password`, `--passwd`：OpenList 明文密码
- `--passwdhash`, `--password-hash`, `--passwd-hash`：OpenList 支持的密码杂凑
- `--print-passwdhash`, `--hash-only`：只计算并打印 `passwdhash`，不登录、不写 token
- `-t`, `--token-file`：token 输出文件，默认 `./token.txt`
- `--otp-code`, `--otp`：二步验证验证码，启用 OTP 时填写
- `-h`, `--help`：显示帮助

`passwd` 和 `passwdhash` 二选一。两者都不传时，脚本会交互式隐藏输入密码。

## passwdhash

OpenList 的 `passwdhash` 计算方式是：

```text
sha256("${passwd}-https://github.com/alist-org/alist")
```

也就是明文密码、一个短横线、固定盐值拼接后计算 SHA-256，输出 64 位小写十六进制。

只计算 `passwdhash`：

```bash
./openlist-login.sh -p 'your-password' --print-passwdhash
```

直接用 `passwdhash` 登录：

```bash
./openlist-login.sh \
  --url http://localhost:35244 \
  -u admin \
  --passwdhash '64位sha256杂凑'
```

## 默认值

脚本开头可以直接修改默认值：

```bash
DEFAULT_OPENLIST_URL="http://localhost:35244"
DEFAULT_USERNAME="admin"
DEFAULT_PASSWORD=""
DEFAULT_PASSWDHASH=""
DEFAULT_TOKEN_FILE="./token.txt"
```

命令行参数会覆盖这些默认值。
