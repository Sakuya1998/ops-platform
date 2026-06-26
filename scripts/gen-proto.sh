#!/usr/bin/env bash
set -euo pipefail

PROTO_DIR="pkg/proto"

usage() {
  cat <<'EOF'
Usage: scripts/gen-proto.sh [options]

Options:
  --proto-dir DIR     Proto directory, default: pkg/proto
  -h, --help          Show help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --proto-dir)
      PROTO_DIR="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
PROTO_ROOT="$ROOT_DIR/$PROTO_DIR"

LOCAL_PROTOC="$ROOT_DIR/.tools/protoc-35.1/bin/protoc"
LOCAL_INCLUDE="$ROOT_DIR/.tools/protoc-35.1/include"
LOCAL_PROTOC_GEN_GO="$ROOT_DIR/.gopath/bin/protoc-gen-go"
LOCAL_PROTOC_GEN_GO_GRPC="$ROOT_DIR/.gopath/bin/protoc-gen-go-grpc"

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Required command '$1' was not found in PATH" >&2
    exit 1
  fi
}

PROTOC="protoc"
if [[ -x "$LOCAL_PROTOC" ]]; then
  PROTOC="$LOCAL_PROTOC"
else
  require_command protoc
fi

if [[ ! -d "$PROTO_ROOT" ]]; then
  echo "Proto directory not found: $PROTO_ROOT" >&2
  exit 1
fi

echo "Generating protobuf code..."
echo "  protoc: $PROTOC"

args=(
  "--proto_path=."
  "--go_out=."
  "--go_opt=paths=source_relative"
  "--go-grpc_out=."
  "--go-grpc_opt=paths=source_relative"
)

if [[ -d "$LOCAL_INCLUDE" ]]; then
  args+=("--proto_path=$LOCAL_INCLUDE")
fi
if [[ -x "$LOCAL_PROTOC_GEN_GO" ]]; then
  args+=("--plugin=protoc-gen-go=$LOCAL_PROTOC_GEN_GO")
else
  require_command protoc-gen-go
fi
if [[ -x "$LOCAL_PROTOC_GEN_GO_GRPC" ]]; then
  args+=("--plugin=protoc-gen-go-grpc=$LOCAL_PROTOC_GEN_GO_GRPC")
else
  require_command protoc-gen-go-grpc
fi

cd "$ROOT_DIR"
while IFS= read -r -d '' proto; do
  relative_proto="${proto#$ROOT_DIR/}"
  echo "  Compiling: $relative_proto"
  "$PROTOC" "${args[@]}" "$relative_proto"
done < <(find "$PROTO_ROOT" -type f -name '*.proto' -print0)

echo "Protobuf code generation complete!"
