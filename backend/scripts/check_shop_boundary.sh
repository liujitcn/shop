#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
backend_root="$repo_root/backend"

check_absent() {
  local description="$1"
  local pattern="$2"
  shift 2
  local matches
  matches="$(rg -n --glob '*.go' --glob '*.proto' --glob '*.ts' "$pattern" "$@" || true)"
  if [[ -n "$matches" ]]; then
    printf '商城解耦边界检查失败：%s\n%s\n' "$description" "$matches" >&2
    exit 1
  fi
}

cd "$backend_root"
check_absent \
  '基础 Go 模块不得导入商城服务或商城协议' \
  '"shop/(service/shop|api/gen/go/shop)' \
  pkg service/base service/system
check_absent \
  '通用协议不得保留商城 SSE 或推荐调试枚举' \
  'AdvanceDataType|SseRefresh(Target|Reason)' \
  api/proto/common/v1

cd "$repo_root"
frontend_targets=(
  frontend/admin/src/api/base
  frontend/admin/src/rpc/base/v1
  frontend/admin/src/rpc/common/v1
)
if [[ -d frontend/app/src/rpc/base/v1 ]]; then
  frontend_targets+=(frontend/app/src/rpc/base/v1)
fi
if [[ -d frontend/app/src/rpc/common/v1 ]]; then
  frontend_targets+=(frontend/app/src/rpc/common/v1)
fi
check_absent \
  '基础前端 SSE 与通用 RPC 类型不得保留商城工作台或推荐调试类型' \
  'SseRefresh|AdvanceDataType' \
  "${frontend_targets[@]}"

printf '商城解耦边界检查通过。\n'
