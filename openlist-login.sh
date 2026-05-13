#!/usr/bin/env bash
set -euo pipefail

# Defaults. CLI flags below override these values.
DEFAULT_OPENLIST_URL="http://localhost:35244"
DEFAULT_USERNAME="admin"
DEFAULT_PASSWORD=""
DEFAULT_PASSWDHASH=""
DEFAULT_TOKEN_FILE="./token.txt"
OPENLIST_STATIC_HASH_SALT="https://github.com/alist-org/alist"

OPENLIST_URL="$DEFAULT_OPENLIST_URL"
USERNAME="$DEFAULT_USERNAME"
PASSWORD="$DEFAULT_PASSWORD"
PASSWDHASH="$DEFAULT_PASSWDHASH"
TOKEN_FILE="$DEFAULT_TOKEN_FILE"
OTP_CODE=""
PASSWORD_FROM_CLI=0
PASSWDHASH_FROM_CLI=0
PRINT_PASSWDHASH_ONLY=0

usage() {
  cat <<'EOF'
用法:
  ./openlist-login.sh [options]

参数:
  --url, --base-url URL        OpenList 地址
  -u, --username USERNAME      OpenList 用户名
  -p, --password PASSWORD      OpenList 密码
  --passwd PASSWORD            --password 的别名
  --passwdhash HASH            OpenList 支持的密码杂凑，和密码二选一
  --print-passwdhash           只计算并打印 passwdhash，不登录、不写 token
  -t, --token-file FILE        token 输出文件，默认 ./token.txt
  --otp-code CODE              二步验证验证码，启用 OTP 时填写
  -h, --help                   显示帮助

示例:
  ./openlist-login.sh --url http://localhost:35244 -u admin -p 'your-password'
  ./openlist-login.sh -p 'your-password' --print-passwdhash
  ./openlist-login.sh --url http://localhost:35244 -u admin --passwdhash 'sha256-hash'
  ./openlist-login.sh --base-url http://openlist:5244 --username admin --passwd 'your-password' --token-file ./token.txt
EOF
}

die() {
  printf '错误: %s\n' "$*" >&2
  exit 1
}

need_arg() {
  local opt="${1:-}"
  local val="${2:-}"
  [[ -n "$val" ]] || die "$opt 需要一个参数值"
}

json_escape() {
  local s="$1"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  s="${s//$'\r'/\\r}"
  s="${s//$'\t'/\\t}"
  printf '%s' "$s"
}

build_payload() {
  local username_json password_json otp_json
  username_json="$(json_escape "$USERNAME")"
  password_json="$(json_escape "$PASSWDHASH")"
  otp_json="$(json_escape "$OTP_CODE")"

  if [[ -n "$OTP_CODE" ]]; then
    printf '{"username":"%s","password":"%s","otp_code":"%s"}' "$username_json" "$password_json" "$otp_json"
  else
    printf '{"username":"%s","password":"%s"}' "$username_json" "$password_json"
  fi
}

extract_json_value() {
  local response="$1"
  local jq_expr="$2"
  local sed_expr="$3"

  if command -v jq >/dev/null 2>&1; then
    printf '%s' "$response" | jq -r "$jq_expr"
  else
    printf '%s' "$response" | sed -n "$sed_expr"
  fi
}

sha256_hex() {
  local input="$1"

  if command -v sha256sum >/dev/null 2>&1; then
    printf '%s' "$input" | sha256sum | cut -d ' ' -f 1
  elif command -v shasum >/dev/null 2>&1; then
    printf '%s' "$input" | shasum -a 256 | cut -d ' ' -f 1
  elif command -v openssl >/dev/null 2>&1; then
    printf '%s' "$input" | openssl dgst -sha256 -r | cut -d ' ' -f 1
  else
    die "需要先安装 sha256sum、shasum 或 openssl 其中之一，用于计算密码杂凑"
  fi
}

openlist_passwd_hash() {
  local passwd="$1"
  sha256_hex "$passwd-$OPENLIST_STATIC_HASH_SALT"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --url|--base-url|--openlist-url)
      need_arg "$1" "${2:-}"
      OPENLIST_URL="$2"
      shift 2
      ;;
    -u|--username)
      need_arg "$1" "${2:-}"
      USERNAME="$2"
      shift 2
      ;;
    -p|--password|--passwd)
      need_arg "$1" "${2:-}"
      [[ "$PASSWDHASH_FROM_CLI" -eq 0 ]] || die "passwd 和 passwdhash 只能二选一"
      PASSWORD="$2"
      PASSWDHASH=""
      PASSWORD_FROM_CLI=1
      shift 2
      ;;
    --passwdhash|--password-hash|--passwd-hash)
      need_arg "$1" "${2:-}"
      [[ "$PASSWORD_FROM_CLI" -eq 0 ]] || die "passwd 和 passwdhash 只能二选一"
      PASSWDHASH="$2"
      PASSWORD=""
      PASSWDHASH_FROM_CLI=1
      shift 2
      ;;
    -t|--token-file)
      need_arg "$1" "${2:-}"
      TOKEN_FILE="$2"
      shift 2
      ;;
    --otp-code|--otp)
      need_arg "$1" "${2:-}"
      OTP_CODE="$2"
      shift 2
      ;;
    --print-passwdhash|--hash-only)
      PRINT_PASSWDHASH_ONLY=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "未知参数: $1"
      ;;
  esac
done

if [[ -n "$PASSWORD" && -n "$PASSWDHASH" ]]; then
  die "passwd 和 passwdhash 只能二选一"
fi

if [[ -z "$PASSWORD" && -z "$PASSWDHASH" ]]; then
  read -r -s -p "OpenList 密码: " PASSWORD
  printf '\n' >&2
fi

if [[ -n "$PASSWORD" ]]; then
  PASSWDHASH="$(openlist_passwd_hash "$PASSWORD")"
fi
[[ -n "$PASSWDHASH" ]] || die "密码或密码杂凑不能为空"

if [[ ! "$PASSWDHASH" =~ ^[0-9A-Fa-f]{64}$ ]]; then
  die "passwdhash 必须是 64 位十六进制 SHA-256"
fi
PASSWDHASH="${PASSWDHASH,,}"

if [[ "$PRINT_PASSWDHASH_ONLY" -eq 1 ]]; then
  printf '%s\n' "$PASSWDHASH"
  exit 0
fi

command -v curl >/dev/null 2>&1 || die "需要先安装 curl"
[[ -n "$OPENLIST_URL" ]] || die "OpenList 地址不能为空"
[[ -n "$USERNAME" ]] || die "用户名不能为空"

OPENLIST_URL="${OPENLIST_URL%/}"
payload="$(build_payload)"

response="$(
  curl -sS \
    -X POST "$OPENLIST_URL/api/auth/login/hash" \
    -H 'Content-Type: application/json;charset=UTF-8' \
    --data "$payload"
)" || die "登录请求失败"

code="$(extract_json_value "$response" '.code // empty' 's/.*"code"[[:space:]]*:[[:space:]]*\([0-9][0-9]*\).*/\1/p')"
message="$(extract_json_value "$response" '.message // empty' 's/.*"message"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
token="$(extract_json_value "$response" '.data.token // empty' 's/.*"token"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"

if [[ "$code" != "200" || -z "$token" ]]; then
  [[ -n "$message" ]] || message="$response"
  die "登录失败: code=${code:-未知}, message=$message"
fi

token_dir="$(dirname "$TOKEN_FILE")"
mkdir -p "$token_dir"
umask 077
printf '%s\n' "$token" > "$TOKEN_FILE"

printf '%s\n' "$token"
printf 'token 已写入 %s\n' "$TOKEN_FILE" >&2
