#!/usr/bin/env bash

# 在本机临时 MongoDB 副本集中检查、清洗并重新导出 Meowpick BSON。
# 不连接远程 MongoDB，不修改输入备份，也不部署或修改后端。

set -Eeuo pipefail
umask 077

readonly SCRIPT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly DEFAULT_DATABASE="meowpick"
readonly DEFAULT_MONGO_IMAGE="mongo:7.0.23-jammy"
readonly MIGRATION_IMAGE_TAG="meowpick-migrate-v2:local"
readonly CONFIRM_TEXT="APPLY-CLEAN-BSON"

COMMAND=""
INPUT=""
OUTPUT=""
WORKSPACE=""
DATABASE="$DEFAULT_DATABASE"
MONGO_IMAGE="$DEFAULT_MONGO_IMAGE"
CONFIRM=""

log() {
  printf '[bson-v2] %s\n' "$*"
}

die() {
  printf '[bson-v2] ERROR: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat <<'EOF'
用法：
  bash scripts/process-mongodb-bson-v2.sh dry-run \
    --input /绝对路径/dump \
    --workspace /绝对路径/work

  bash scripts/process-mongodb-bson-v2.sh apply \
    --workspace /绝对路径/work \
    --output /绝对路径/clean-dump \
    --confirm APPLY-CLEAN-BSON

  bash scripts/process-mongodb-bson-v2.sh status --workspace /绝对路径/work
  bash scripts/process-mongodb-bson-v2.sh cleanup --workspace /绝对路径/work

可选参数：
  --db meowpick                    输入备份中的数据库名，默认 meowpick
  --mongo-image mongo:7.0.23-jammy 临时 MongoDB 镜像，默认与测试环境一致

输入可以是 mongodump 目录或 archive，例如：
  /backup/20260902/meowpick/course.bson
  /backup/20260902/meowpick/course.metadata.json
  /backup/20260902/meowpick.archive.gz

dry-run 只修改临时副本并生成报告；apply 才会迁移临时副本并生成新的 BSON 目录。
任何命令都不会修改 --input。
EOF
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "缺少命令：$1"
}

absolute_path() {
  local path="$1"
  [[ "$path" == /* ]] || die "必须使用绝对路径：$path"
  printf '%s\n' "${path%/}"
}

parse_args() {
  [[ $# -ge 1 ]] || { usage; exit 1; }
  COMMAND="$1"
  shift
  if [[ "$COMMAND" == "-h" || "$COMMAND" == "--help" ]]; then
    usage
    exit 0
  fi
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --input) INPUT="${2:-}"; shift 2 ;;
      --output) OUTPUT="${2:-}"; shift 2 ;;
      --workspace) WORKSPACE="${2:-}"; shift 2 ;;
      --db) DATABASE="${2:-}"; shift 2 ;;
      --mongo-image) MONGO_IMAGE="${2:-}"; shift 2 ;;
      --confirm) CONFIRM="${2:-}"; shift 2 ;;
      -h|--help) usage; exit 0 ;;
      *) die "未知参数：$1" ;;
    esac
  done
  [[ "$DATABASE" =~ ^[A-Za-z0-9_-]+$ ]] || die "数据库名只允许字母、数字、下划线和短横线"
  [[ -n "$WORKSPACE" ]] || die "必须提供 --workspace"
  WORKSPACE="$(absolute_path "$WORKSPACE")"
}

state_file() {
  printf '%s/state/%s\n' "$WORKSPACE" "$1"
}

write_state() {
  printf '%s\n' "$2" >"$(state_file "$1")"
}

read_state() {
  local path
  path="$(state_file "$1")"
  [[ -f "$path" && ! -L "$path" ]] || die "迁移状态缺少：$1"
  tr -d '\r\n' <"$path"
}

validate_state_name() {
  [[ "$1" =~ ^meowpick-v2-[a-f0-9]{12}-(mongo|net|data)$ ]] || die "迁移状态中的 Docker 名称不合法：$1"
}

find_database_dir() {
  local input="$1"
  if [[ -d "$input/$DATABASE" ]]; then
    printf '%s\n' "$input/$DATABASE"
  elif [[ "$(basename "$input")" == "$DATABASE" ]]; then
    printf '%s\n' "$input"
  else
    die "输入中找不到数据库目录：$DATABASE"
  fi
}

validate_dump_dir() {
  local db_dir="$1"
  find "$db_dir" -maxdepth 1 -type f -name '*.bson' -print -quit | grep -q . || die "数据库目录中没有 .bson 文件：$db_dir"
  while IFS= read -r bson_file; do
    local metadata="${bson_file%.bson}.metadata.json"
    [[ -f "$metadata" ]] || die "缺少索引元数据文件：$metadata"
  done < <(find "$db_dir" -maxdepth 1 -type f -name '*.bson' -print)
}

docker_object_absent() {
  local kind="$1" name="$2"
  ! docker "$kind" inspect "$name" >/dev/null 2>&1
}

wait_for_primary() {
  local container="$1"
  local attempt
  for attempt in $(seq 1 60); do
    if docker exec "$container" mongosh --quiet --eval 'quit(db.hello().isWritablePrimary ? 0 : 1)' >/dev/null 2>&1; then
      return
    fi
    sleep 1
  done
  die "临时 MongoDB 在 60 秒内未成为主节点"
}

start_dry_run() {
  [[ -n "$INPUT" ]] || die "dry-run 必须提供 --input"
  INPUT="$(absolute_path "$INPUT")"
  [[ -e "$INPUT" ]] || die "输入备份不存在：$INPUT"
  [[ ! -e "$WORKSPACE" ]] || die "工作目录已经存在，拒绝覆盖：$WORKSPACE"

  local db_dir input_parent input_name input_kind run_id container network volume image_id migration_exit
  if [[ -d "$INPUT" ]]; then
    db_dir="$(find_database_dir "$INPUT")"
    validate_dump_dir "$db_dir"
    input_parent="$(dirname "$db_dir")"
    input_name="$(basename "$db_dir")"
    input_kind="directory"
  elif [[ "$INPUT" == *.archive.gz ]]; then
    gzip -t "$INPUT" || die "gzip archive 校验失败：$INPUT"
    input_parent="$(dirname "$INPUT")"
    input_name="$(basename "$INPUT")"
    input_kind="archive-gzip"
  elif [[ "$INPUT" == *.archive ]]; then
    input_parent="$(dirname "$INPUT")"
    input_name="$(basename "$INPUT")"
    input_kind="archive"
  else
    die "输入必须是 mongodump 目录、.archive 或 .archive.gz"
  fi
  run_id="$(printf '%s' "$WORKSPACE-$(date +%s)-$$" | shasum -a 256 | cut -c1-12)"
  container="meowpick-v2-${run_id}-mongo"
  network="meowpick-v2-${run_id}-net"
  volume="meowpick-v2-${run_id}-data"

  mkdir -m 700 "$WORKSPACE"
  mkdir -m 700 "$WORKSPACE/state" "$WORKSPACE/reports"
  write_state database "$DATABASE"
  write_state mongo_image "$MONGO_IMAGE"
  write_state container "$container"
  write_state network "$network"
  write_state volume "$volume"
  write_state input "$INPUT"
  write_state input_kind "$input_kind"
  write_state phase initializing

  docker_object_absent container "$container" || die "容器名已存在：$container"
  docker_object_absent network "$network" || die "网络名已存在：$network"
  docker_object_absent volume "$volume" || die "数据卷名已存在：$volume"

  log "构建独立迁移镜像（不会构建或部署后端）"
  docker build --file "$SCRIPT_ROOT/Dockerfile.migrate-v2" --tag "$MIGRATION_IMAGE_TAG" "$SCRIPT_ROOT"
  image_id="$(docker image inspect "$MIGRATION_IMAGE_TAG" --format '{{.Id}}')"
  write_state migration_image_id "$image_id"

  docker network create "$network" >/dev/null
  docker volume create "$volume" >/dev/null
  docker run --detach --name "$container" --network "$network" --volume "$volume:/data/db" \
    "$MONGO_IMAGE" mongod --replSet rs0 --bind_ip_all --port 27017 >/dev/null

  log "初始化隔离的单节点副本集"
  local attempt
  for attempt in $(seq 1 60); do
    if docker exec "$container" mongosh --quiet --eval \
      "rs.initiate({_id:'rs0',members:[{_id:0,host:'${container}:27017'}]})" >/dev/null 2>&1; then
      break
    fi
    sleep 1
  done
  wait_for_primary "$container"

  log "把输入 BSON 还原到临时数据库；输入目录以只读方式挂载"
  case "$input_kind" in
    directory)
      docker run --rm --network "$network" --volume "$input_parent:/input:ro" "$MONGO_IMAGE" \
        mongorestore --host "$container" --port 27017 --db "$DATABASE" "/input/$input_name"
      ;;
    archive-gzip)
      docker run --rm --network "$network" --volume "$input_parent:/input:ro" "$MONGO_IMAGE" \
        mongorestore --host "$container" --port 27017 --archive="/input/$input_name" --gzip \
        --nsInclude "${DATABASE}.*"
      ;;
    archive)
      docker run --rm --network "$network" --volume "$input_parent:/input:ro" "$MONGO_IMAGE" \
        mongorestore --host "$container" --port 27017 --archive="/input/$input_name" \
        --nsInclude "${DATABASE}.*"
      ;;
  esac

  log "执行只读迁移预检并生成报告"
  set +e
  docker run --rm --network "$network" --volume "$WORKSPACE/reports:/reports" "$image_id" \
    --uri "mongodb://${container}:27017/${DATABASE}?replicaSet=rs0" \
    --db "$DATABASE" --replica-set rs0 \
    --require-host "$container" --require-port 27017 --require-db "$DATABASE" \
    --report /reports/dry-run-plan.json
  migration_exit=$?
  set -e

  if [[ "$migration_exit" -eq 2 ]]; then
    write_state phase conflicts
    log "发现阻断冲突，没有应用迁移。报告：$WORKSPACE/reports/dry-run-plan.json"
    log "临时数据库被保留，可执行 status 查看；确认后执行 cleanup 清理。"
    exit 2
  fi
  [[ "$migration_exit" -eq 0 ]] || die "迁移预检执行失败，退出码：$migration_exit"
  write_state phase dry-run-passed
  log "dry-run 通过。请先阅读：$WORKSPACE/reports/dry-run-plan.json"
  log "确认后再执行 apply；在此之前不要执行 cleanup。"
}

load_runtime_state() {
  [[ -d "$WORKSPACE/state" && ! -L "$WORKSPACE/state" ]] || die "工作目录无有效迁移状态：$WORKSPACE"
  DATABASE="$(read_state database)"
  MONGO_IMAGE="$(read_state mongo_image)"
}

show_status() {
  load_runtime_state
  local phase container network volume input
  phase="$(read_state phase)"
  container="$(read_state container)"
  network="$(read_state network)"
  volume="$(read_state volume)"
  input="$(read_state input)"
  validate_state_name "$container"
  validate_state_name "$network"
  validate_state_name "$volume"
  printf 'phase=%s\ndatabase=%s\ninput=%s\ncontainer=%s\nnetwork=%s\nvolume=%s\n' \
    "$phase" "$DATABASE" "$input" "$container" "$network" "$volume"
  docker inspect "$container" --format 'containerRunning={{.State.Running}}' 2>/dev/null || printf 'containerRunning=false\n'
}

checksum_output() {
  local output="$1" manifest="$1/SHA256SUMS"
  : >"$manifest"
  while IFS= read -r file; do
    local relative digest
    relative="${file#"$output/"}"
    digest="$(shasum -a 256 "$file" | awk '{print $1}')"
    printf '%s  %s\n' "$digest" "$relative" >>"$manifest"
  done < <(find "$output" -type f ! -name SHA256SUMS -print | sort)
}

apply_and_export() {
  load_runtime_state
  [[ "$CONFIRM" == "$CONFIRM_TEXT" ]] || die "apply 必须提供 --confirm $CONFIRM_TEXT"
  [[ -n "$OUTPUT" ]] || die "apply 必须提供 --output"
  OUTPUT="$(absolute_path "$OUTPUT")"
  [[ ! -e "$OUTPUT" ]] || die "输出目录已经存在，拒绝覆盖：$OUTPUT"

  local phase container network volume image_id output_parent output_name output_kind
  phase="$(read_state phase)"
  [[ "$phase" == "dry-run-passed" ]] || die "只有 dry-run-passed 状态可以 apply，当前：$phase"
  container="$(read_state container)"
  network="$(read_state network)"
  volume="$(read_state volume)"
  image_id="$(read_state migration_image_id)"
  validate_state_name "$container"
  validate_state_name "$network"
  validate_state_name "$volume"
  docker inspect "$container" >/dev/null 2>&1 || die "临时 MongoDB 容器不存在；请重新 dry-run"
  docker image inspect "$image_id" >/dev/null 2>&1 || die "dry-run 使用的迁移镜像不存在；请重新 dry-run"
  wait_for_primary "$container"

  write_state phase applying
  log "应用已经审核的 dry-run 计划"
  docker run --rm --network "$network" --volume "$WORKSPACE/reports:/reports" "$image_id" \
    --uri "mongodb://${container}:27017/${DATABASE}?replicaSet=rs0" \
    --db "$DATABASE" --replica-set rs0 \
    --require-host "$container" --require-port 27017 --require-db "$DATABASE" \
    --apply-plan /reports/dry-run-plan.json

  log "执行迁移后检查"
  docker run --rm --network "$network" --volume "$WORKSPACE/reports:/reports" "$image_id" \
    --uri "mongodb://${container}:27017/${DATABASE}?replicaSet=rs0" \
    --db "$DATABASE" --replica-set rs0 \
    --require-host "$container" --require-port 27017 --require-db "$DATABASE" \
    --report /reports/postcheck-plan.json

  python3 - "$WORKSPACE/reports/postcheck-plan.json" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as handle:
    report = json.load(handle)
pending = sum(len(report.get(name) or []) for name in (
    "conflicts", "courseRepairs", "commentIdRepairs", "teacherTimeRepairs"
))
if pending:
    raise SystemExit(f"迁移后仍有 {pending} 个冲突或待修复项，拒绝导出")
PY

  output_parent="$(dirname "$OUTPUT")"
  output_name="$(basename "$OUTPUT")"
  mkdir -p "$output_parent"
  log "导出新的 BSON 备份：$OUTPUT"
  if [[ "$OUTPUT" == *.archive.gz ]]; then
    output_kind="archive-gzip"
    docker run --rm --network "$network" --volume "$output_parent:/output" "$MONGO_IMAGE" \
      mongodump --host "$container" --port 27017 --db "$DATABASE" --archive="/output/$output_name" --gzip
    gzip -t "$OUTPUT"
    shasum -a 256 "$OUTPUT" >"$OUTPUT.sha256"
    docker run --rm --network "$network" --volume "$output_parent:/output:ro" "$MONGO_IMAGE" \
      mongorestore --host "$container" --port 27017 --archive="/output/$output_name" --gzip \
      --nsInclude "${DATABASE}.*" --dryRun
  elif [[ "$OUTPUT" == *.archive ]]; then
    output_kind="archive"
    docker run --rm --network "$network" --volume "$output_parent:/output" "$MONGO_IMAGE" \
      mongodump --host "$container" --port 27017 --db "$DATABASE" --archive="/output/$output_name"
    shasum -a 256 "$OUTPUT" >"$OUTPUT.sha256"
    docker run --rm --network "$network" --volume "$output_parent:/output:ro" "$MONGO_IMAGE" \
      mongorestore --host "$container" --port 27017 --archive="/output/$output_name" \
      --nsInclude "${DATABASE}.*" --dryRun
  else
    output_kind="directory"
    docker run --rm --network "$network" --volume "$output_parent:/output" "$MONGO_IMAGE" \
      mongodump --host "$container" --port 27017 --db "$DATABASE" --out "/output/$output_name"
    checksum_output "$OUTPUT"
    docker run --rm --network "$network" --volume "$output_parent:/output:ro" "$MONGO_IMAGE" \
      mongorestore --host "$container" --port 27017 --db "$DATABASE" "/output/$output_name/$DATABASE" --dryRun
  fi
  write_state output "$OUTPUT"
  write_state output_kind "$output_kind"
  write_state phase exported
  log "处理完成。原始备份未修改；新备份和校验清单位于：$OUTPUT"
  log "检查无误后可执行 cleanup 清理临时容器、网络和数据卷。"
}

cleanup_runtime() {
  load_runtime_state
  local container network volume
  container="$(read_state container)"
  network="$(read_state network)"
  volume="$(read_state volume)"
  validate_state_name "$container"
  validate_state_name "$network"
  validate_state_name "$volume"
  docker rm --force "$container" >/dev/null 2>&1 || true
  docker network rm "$network" >/dev/null 2>&1 || true
  docker volume rm "$volume" >/dev/null 2>&1 || true
  write_state phase cleaned
  log "已删除本次临时 MongoDB 容器、隔离网络和数据卷；报告、原始备份和输出备份均保留。"
}

main() {
  parse_args "$@"
  require_command docker
  require_command find
  require_command grep
  require_command shasum
  require_command python3
  require_command gzip
  docker info >/dev/null 2>&1 || die "Docker Desktop/Engine 未运行或当前用户无权访问"
  case "$COMMAND" in
    dry-run) start_dry_run ;;
    apply) apply_and_export ;;
    status) show_status ;;
    cleanup) cleanup_runtime ;;
    *) usage; die "未知命令：$COMMAND" ;;
  esac
}

main "$@"
