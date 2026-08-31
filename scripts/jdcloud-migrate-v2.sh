#!/usr/bin/env bash

set -Eeuo pipefail
umask 077

readonly BACKEND_CONTAINER="test-meowpick-backend"
readonly BACKEND_COMPOSE="/home/eagle233/repos/test/meowpick/docker-compose.yml"
readonly BACKEND_CONFIG="/home/eagle233/repos/test/meowpick/config.yaml"
readonly BACKEND_IMAGE="boyuanclub/meowpick-backend:latest"
readonly MONGO_CONTAINER="test-mongo"
readonly MONGO_COMPOSE="/home/eagle233/portainer/srv/compose/1/v1/docker-compose.yml"
readonly MONGO_CONFIG_DIR="/home/eagle233/repos/test/mongo/config"
readonly MONGO_KEYFILE="${MONGO_CONFIG_DIR}/mongodb-keyfile"
readonly DOCKER_NETWORK="test-net"
readonly EXPECTED_MONGO_HOST="test-mongo"
readonly EXPECTED_MONGO_PORT="27017"
readonly EXPECTED_DATABASE="meowpick"
readonly EXPECTED_HOSTNAME="Eagle233-JDCloud"
readonly EXPECTED_MACHINE_ID_SHA256="7f4a670ce540009b2c8234fd8fedf3be9d264fe1ae39c7a7fef4e99fe2772b3a"
readonly STATE_ROOT="/home/eagle233/migrations"
readonly STATE_POINTER="${STATE_ROOT}/.meowpick-v2-current"
readonly RUN_DIR_PREFIX="${STATE_ROOT}/meowpick-v2-"
readonly SCRIPT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

RUN_DIR=""
MONGO_URI=""
MONGO_DB=""

log() {
  printf '[migrate-v2] %s\n' "$*"
}

die() {
  printf '[migrate-v2] ERROR: %s\n' "$*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "缺少命令: $1"
}

require_exact_container() {
  local expected="$1"
  local actual
  actual="$(docker inspect --format '{{.Name}}' "$expected" 2>/dev/null || true)"
  [[ "$actual" == "/${expected}" ]] || die "找不到精确容器: $expected"
}

require_network_attachment() {
  local attached
  attached="$(docker inspect --format '{{if index .NetworkSettings.Networks "test-net"}}test-net{{end}}' "$1")"
  [[ "$attached" == "$DOCKER_NETWORK" ]] || die "容器 $1 未连接到 $DOCKER_NETWORK"
}

guard_environment() {
  [[ "$(hostname)" == "$EXPECTED_HOSTNAME" ]] || die "拒绝主机: $(hostname)"
  [[ "$(sha256sum /etc/machine-id | awk '{print $1}')" == "$EXPECTED_MACHINE_ID_SHA256" ]] || die "机器指纹不匹配"
  [[ -z "${DOCKER_HOST:-}" ]] || die "拒绝 DOCKER_HOST 覆盖: $DOCKER_HOST"
  [[ -z "${DOCKER_CONTEXT:-}" ]] || die "拒绝 DOCKER_CONTEXT 覆盖: $DOCKER_CONTEXT"
  [[ "$(docker context show)" == "default" ]] || die "Docker context 必须是 default"
  [[ "$(docker context inspect default --format '{{(index .Endpoints "docker").Host}}')" == "unix:///var/run/docker.sock" ]] || die "default context 不是本机 Docker socket"

  local mongo_project mongo_service mongo_image mongo_mounts mongo_ports backend_project backend_service
  mongo_project="$(docker inspect --format '{{index .Config.Labels "com.docker.compose.project"}}' "$MONGO_CONTAINER")"
  mongo_service="$(docker inspect --format '{{index .Config.Labels "com.docker.compose.service"}}' "$MONGO_CONTAINER")"
  mongo_image="$(docker inspect --format '{{.Config.Image}}' "$MONGO_CONTAINER")"
  mongo_mounts="$(docker inspect --format '{{range .Mounts}}{{println .Source "->" .Destination}}{{end}}' "$MONGO_CONTAINER")"
  mongo_ports="$(docker port "$MONGO_CONTAINER" 27017/tcp)"
  backend_project="$(docker inspect --format '{{index .Config.Labels "com.docker.compose.project"}}' "$BACKEND_CONTAINER")"
  backend_service="$(docker inspect --format '{{index .Config.Labels "com.docker.compose.service"}}' "$BACKEND_CONTAINER")"

  [[ "$mongo_project" == "test" && "$mongo_service" == "$MONGO_CONTAINER" ]] || die "test-mongo Compose 标签不匹配"
  [[ "$mongo_image" == "mongo:7.0.23-jammy" ]] || die "test-mongo 镜像不匹配: $mongo_image"
  [[ "$mongo_mounts" == *'/home/eagle233/repos/test/mongo/data -> /data/db'* ]] || die "test-mongo 数据挂载不匹配"
  [[ "$mongo_mounts" == *'/home/eagle233/repos/test/mongo/config -> /data/configdb'* ]] || die "test-mongo 配置挂载不匹配"
  [[ "$mongo_ports" == "0.0.0.0:27015" || "$mongo_ports" == "[::]:27015" || "$mongo_ports" == $'0.0.0.0:27015\n[::]:27015' ]] || die "test-mongo 宿主端口不匹配: $mongo_ports"
  [[ "$backend_project" == "test-meowpick" && "$backend_service" == "$BACKEND_CONTAINER" ]] || die "后端 Compose 标签不匹配"
}

load_config() {
  local -a config_lines
  mapfile -t config_lines < <(python3 - "$BACKEND_CONFIG" <<'PY'
from pathlib import Path
import re
import sys

text = Path(sys.argv[1]).read_text()
sections = list(re.finditer(r"(?m)^Mongo:\s*\n(?P<body>(?:^[ \t]+[^\n]*(?:\n|$))*)", text))
if len(sections) != 1:
    raise SystemExit(f"Mongo 配置段必须恰好一个，当前为 {len(sections)} 个")
body = sections[0].group("body")
urls = re.findall(r"(?m)^[ \t]+URL:\s*[\"']([^\"']+)[\"']\s*$", body)
databases = re.findall(r"(?m)^[ \t]+DB:\s*[\"']([^\"']+)[\"']\s*$", body)
if len(urls) != 1 or len(databases) != 1:
    raise SystemExit("Mongo 段必须恰好包含一个 URL 和一个 DB")
print(urls[0])
print(databases[0])
PY
)
  [[ "${#config_lines[@]}" -eq 2 ]] || die "读取 Mongo 配置失败"
  MONGO_URI="${config_lines[0]}"
  MONGO_DB="${config_lines[1]}"
}

guard_target() {
  guard_environment
  load_config
  MONGO_URI="$MONGO_URI" MONGO_DB="$MONGO_DB" python3 - <<'PY'
import os
from urllib.parse import urlsplit

uri = urlsplit(os.environ["MONGO_URI"])
database = os.environ["MONGO_DB"]
if uri.scheme not in {"mongodb", "mongodb+srv"}:
    raise SystemExit("Mongo URI 协议不正确")
if uri.hostname != "test-mongo" or uri.port != 27017:
    raise SystemExit(f"拒绝目标 MongoDB: {uri.hostname}:{uri.port}")
if uri.path != "/meowpick" or database != "meowpick":
    raise SystemExit(f"拒绝目标数据库: uriPath={uri.path!r} db={database!r}")
PY
  require_exact_container "$MONGO_CONTAINER"
  require_network_attachment "$MONGO_CONTAINER"
}

load_run_dir() {
  [[ -f "$STATE_POINTER" ]] || die "没有当前迁移目录，请先执行 init"
  [[ ! -L "$STATE_POINTER" ]] || die "迁移状态指针不能是符号链接"
  [[ "$(stat -c '%u:%a' "$STATE_POINTER")" == "$(id -u):600" ]] || die "迁移状态指针权限或所有者不正确"
  local candidate basename_value
  candidate="$(<"$STATE_POINTER")"
  RUN_DIR="$(realpath -e "$candidate")"
  [[ "$(dirname "$RUN_DIR")" == "$STATE_ROOT" ]] || die "迁移目录不在允许父目录: $RUN_DIR"
  basename_value="$(basename "$RUN_DIR")"
  [[ "$basename_value" =~ ^meowpick-v2-[0-9]{8}-[0-9]{6}$ ]] || die "迁移目录名称不合法: $basename_value"
  [[ -d "$RUN_DIR" ]] || die "迁移目录不存在: $RUN_DIR"
  [[ "$(stat -c '%u:%a' "$RUN_DIR")" == "$(id -u):700" ]] || die "迁移目录权限或所有者不正确"
}

require_backend_stopped() {
  local running
  running="$(docker inspect --format '{{.State.Running}}' "$BACKEND_CONTAINER")"
  [[ "$running" == "false" ]] || die "$BACKEND_CONTAINER 仍在运行"
}

replica_state() {
  docker exec "$MONGO_CONTAINER" mongosh --quiet --eval 'JSON.stringify({setName:db.hello().setName || "",primary:db.hello().isWritablePrimary === true})'
}

require_primary() {
  local state
  state="$(replica_state)"
  [[ "$state" == *'"setName":"rs0"'* && "$state" == *'"primary":true'* ]] || die "test-mongo 不是 rs0 主节点: $state"
}

preflight() {
  require_command docker
  require_command git
  require_command python3
  require_command sha256sum
  require_command realpath
  require_command gzip
  require_command cmp
  require_command df
  require_command stat
  require_command install
  require_command seq
  require_command sudo
  sudo -n true || die "需要免密 sudo 才能维护 Portainer Compose"
  require_exact_container "$BACKEND_CONTAINER"
  require_exact_container "$MONGO_CONTAINER"
  require_network_attachment "$BACKEND_CONTAINER"
  guard_target
  [[ -f "$BACKEND_COMPOSE" ]] || die "后端 Compose 不存在"
  sudo test -f "$MONGO_COMPOSE" || die "MongoDB Compose 不存在"
  docker exec "$MONGO_CONTAINER" mongodump --version >/dev/null
  docker exec "$MONGO_CONTAINER" mongorestore --version >/dev/null
  log "目标确认: test-mongo:27017/meowpick"
  log "当前 MongoDB 拓扑: $(replica_state)"
}

init_run() {
  preflight
  local run_id
  run_id="$(date +%Y%m%d-%H%M%S)"
  RUN_DIR="${RUN_DIR_PREFIX}${run_id}"
  install -d -m 700 "$STATE_ROOT"
  if [[ -e "$STATE_POINTER" && "${MEOWPICK_CONFIRM_NEW_RUN:-}" != "NEW-test-mongo-run" ]]; then
    die "已有迁移状态指针；如确认开始新一轮，请设置 MEOWPICK_CONFIRM_NEW_RUN=NEW-test-mongo-run"
  fi
  mkdir -m 700 "$RUN_DIR"
  printf '%s\n' "$RUN_DIR" >"$STATE_POINTER"
  chmod 600 "$STATE_POINTER"
  log "本次迁移目录: $RUN_DIR"
}

backup() {
  load_run_dir
  guard_target
  [[ "${MEOWPICK_CONFIRM_EXCLUSIVE_WRITER:-}" == "ONLY-test-meowpick-writes-meowpick" ]] || die "请先确认没有其他程序写 meowpick，并设置 MEOWPICK_CONFIRM_EXCLUSIVE_WRITER=ONLY-test-meowpick-writes-meowpick"
  [[ ! -e "$RUN_DIR/meowpick-before.archive.gz" ]] || die "备份文件已存在，拒绝覆盖"
  log "停止后端，冻结 Meowpick 写入"
  docker stop "$BACKEND_CONTAINER" >/dev/null
  require_backend_stopped

  docker inspect --format '{{.Image}}' "$BACKEND_CONTAINER" >"$RUN_DIR/old-backend-image-id.txt"
  stat -c '%u:%g:%a' "$BACKEND_CONFIG" >"$RUN_DIR/config.before.meta"
  sudo stat -c '%u:%g:%a' "$MONGO_COMPOSE" >"$RUN_DIR/docker-compose.before.meta"
  install -m 600 "$BACKEND_CONFIG" "$RUN_DIR/config.before.yaml"
  sudo install -o "$(id -u)" -g "$(id -g)" -m 600 "$MONGO_COMPOSE" "$RUN_DIR/docker-compose.before.yml"
  docker exec "$MONGO_CONTAINER" sh -c 'exec mongosh --quiet --username "$MONGO_INITDB_ROOT_USERNAME" --password "$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin --eval "$1"' sh 'print(db.getSiblingDB("meowpick").stats().storageSize)' >"$RUN_DIR/meowpick-storage-size.txt"
  [[ "$(<"$RUN_DIR/meowpick-storage-size.txt")" =~ ^[0-9]+$ ]] || die "无法读取 meowpick storageSize"
  chmod 600 "$RUN_DIR/meowpick-storage-size.txt"

  local temporary_backup="$RUN_DIR/.meowpick-before.archive.gz.tmp"
  if ! docker exec "$MONGO_CONTAINER" sh -c 'exec mongodump --username "$MONGO_INITDB_ROOT_USERNAME" --password "$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin --db meowpick --archive --gzip' >"$temporary_backup"; then
    rm -f "$temporary_backup"
    die "mongodump 失败"
  fi
  gzip -t "$temporary_backup"
  mv "$temporary_backup" "$RUN_DIR/meowpick-before.archive.gz"
  chmod 600 "$RUN_DIR/meowpick-before.archive.gz"
  sha256sum "$RUN_DIR/meowpick-before.archive.gz" >"$RUN_DIR/meowpick-before.archive.gz.sha256"
  sha256sum -c "$RUN_DIR/meowpick-before.archive.gz.sha256"

  if ! docker exec -i "$MONGO_CONTAINER" sh -c 'exec mongorestore --dryRun --username "$MONGO_INITDB_ROOT_USERNAME" --password "$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin --archive --gzip' <"$RUN_DIR/meowpick-before.archive.gz" >"$RUN_DIR/mongorestore-dry-run.log" 2>&1; then
    die "mongorestore --dryRun 未通过，查看 $RUN_DIR/mongorestore-dry-run.log"
  fi
  chmod 600 "$RUN_DIR/mongorestore-dry-run.log"
  log "备份已生成并通过 gzip、SHA-256 和 mongorestore dry-run 检查"
  log "请把 $RUN_DIR/meowpick-before.archive.gz 及校验文件复制到服务器外"
}

write_replica_compose() {
  sudo python3 - "$MONGO_COMPOSE" <<'PY'
from pathlib import Path
import os
import stat
import sys
import tempfile

path = Path(sys.argv[1])
info = path.stat()
text = path.read_text()
start = text.index("  test-mongo:\n")
end = text.index("  test-redis:\n", start)
section = text[start:end]

required_mount = "/home/eagle233/repos/test/mongo/config:/data/configdb"
if required_mount not in section:
    raise SystemExit("test-mongo 缺少预期的 /data/configdb 挂载")
expected_command = '    command: ["mongod", "--replSet", "rs0", "--keyFile", "/data/configdb/mongodb-keyfile", "--bind_ip_all"]\n'
if "command:" in section:
    if section.count(expected_command) == 1 and section.count("command:") == 1:
        raise SystemExit(0)
    raise SystemExit("test-mongo 已有非规范 command，拒绝覆盖")
if "--replSet" in section:
    raise SystemExit("test-mongo 存在不完整副本集参数，拒绝覆盖")
marker = "    restart: always\n"
if section.count(marker) != 1:
    raise SystemExit("找不到唯一 restart 插入点")
section = section.replace(marker, marker + expected_command, 1)
updated = text[:start] + section + text[end:]

fd, temporary = tempfile.mkstemp(prefix=path.name + ".", dir=path.parent)
try:
    with os.fdopen(fd, "w") as handle:
        handle.write(updated)
        handle.flush()
        os.fsync(handle.fileno())
    os.chmod(temporary, stat.S_IMODE(info.st_mode))
    os.chown(temporary, info.st_uid, info.st_gid)
    os.replace(temporary, path)
finally:
    if os.path.exists(temporary):
        os.unlink(temporary)
PY
}

append_replica_uri() {
  python3 - "$BACKEND_CONFIG" <<'PY'
from pathlib import Path
import os
import stat
import sys
import tempfile
from urllib.parse import parse_qsl, urlencode, urlsplit, urlunsplit
import re

path = Path(sys.argv[1])
info = path.stat()
text = path.read_text()
mongo = re.search(r"(?m)^Mongo:\s*\n(?P<body>(?:^[ \t]+[^\n]*(?:\n|$))*)", text)
if not mongo:
    raise SystemExit("找不到 Mongo 配置段")
body = mongo.group("body")
matches = list(re.finditer(r"(?m)^(?P<prefix>[ \t]+URL:\s*[\"'])(?P<uri>mongodb[^\"']+)(?P<suffix>[\"']\s*)$", body))
db_matches = re.findall(r"(?m)^[ \t]+DB:\s*[\"']([^\"']+)[\"']\s*$", body)
if len(matches) != 1 or db_matches != ["meowpick"]:
    raise SystemExit("Mongo 段必须恰好包含目标 URL 和 DB=meowpick")
match = matches[0]
uri = urlsplit(match.group("uri"))
if uri.hostname != "test-mongo" or uri.port != 27017 or uri.path != "/meowpick":
    raise SystemExit("拒绝修改非 test-mongo/meowpick URI")
query = dict(parse_qsl(uri.query, keep_blank_values=True))
if query.get("replicaSet") not in {None, "rs0"}:
    raise SystemExit("Mongo URI 已指定其他 replicaSet")
query["replicaSet"] = "rs0"
updated_uri = urlunsplit((uri.scheme, uri.netloc, uri.path, urlencode(query), uri.fragment))
updated_body = body[:match.start()] + match.group("prefix") + updated_uri + match.group("suffix") + body[match.end():]
updated = text[:mongo.start("body")] + updated_body + text[mongo.end("body"):]

fd, temporary = tempfile.mkstemp(prefix=path.name + ".", dir=path.parent)
try:
    with os.fdopen(fd, "w") as handle:
        handle.write(updated)
        handle.flush()
        os.fsync(handle.fileno())
    os.chmod(temporary, stat.S_IMODE(info.st_mode))
    os.chown(temporary, info.st_uid, info.st_gid)
    os.replace(temporary, path)
finally:
    if os.path.exists(temporary):
        os.unlink(temporary)
PY
}

prepare_replica() {
  load_run_dir
  require_backend_stopped
  [[ -s "$RUN_DIR/meowpick-before.archive.gz" ]] || die "必须先完成 backup"
  sha256sum -c "$RUN_DIR/meowpick-before.archive.gz.sha256"
  [[ "${MEOWPICK_CONFIRM_PREPARE:-}" == "PREPARE-test-mongo-rs0" ]] || die "请设置 MEOWPICK_CONFIRM_PREPARE=PREPARE-test-mongo-rs0"

  if ! sudo test -e "$MONGO_KEYFILE"; then
    local mongo_image_id
    mongo_image_id="$(docker inspect --format '{{.Image}}' "$MONGO_CONTAINER")"
    [[ -n "$mongo_image_id" ]] || die "无法读取 test-mongo 镜像 ID"
    sudo docker run --rm --user root --volume "$MONGO_CONFIG_DIR:/keydir" "$mongo_image_id" bash -ceu 'set -o noclobber; umask 077; openssl rand -base64 756 > /keydir/mongodb-keyfile; chown mongodb:mongodb /keydir/mongodb-keyfile; chmod 400 /keydir/mongodb-keyfile'
  fi
  local key_meta
  key_meta="$(sudo stat -c '%u:%g:%a' "$MONGO_KEYFILE")"
  [[ "$key_meta" == "999:999:400" ]] || die "keyFile 权限必须是 999:999:400，当前为 $key_meta"

  write_replica_compose
  if ! sudo docker compose -f "$MONGO_COMPOSE" config --quiet; then
    local compose_uid compose_gid compose_mode
    IFS=: read -r compose_uid compose_gid compose_mode <"$RUN_DIR/docker-compose.before.meta"
    sudo install -o "$compose_uid" -g "$compose_gid" -m "$compose_mode" "$RUN_DIR/docker-compose.before.yml" "$MONGO_COMPOSE"
    die "修改后的 MongoDB Compose 校验失败，已恢复原文件"
  fi
  log "MongoDB Compose 已通过语法校验；未输出完整 diff，避免泄露相邻环境变量"
  sudo docker compose -f "$MONGO_COMPOSE" up -d "$MONGO_CONTAINER"

  local initiate_js
  initiate_js='try { const s=rs.status(); if (s.set !== "rs0") { throw new Error("unexpected replica set: " + s.set); } print("already initialized"); } catch (e) { if (e.code === 94 || e.codeName === "NotYetInitialized") { printjson(rs.initiate({_id:"rs0",members:[{_id:0,host:"test-mongo:27017"}]})); } else { throw e; } }'
  local ready_attempt mongo_ready="false"
  for ready_attempt in $(seq 1 60); do
    if docker exec "$MONGO_CONTAINER" mongosh --quiet --eval 'quit(db.hello().ok === 1 ? 0 : 1)' >/dev/null 2>&1; then
      mongo_ready="true"
      break
    fi
    sleep 2
  done
  [[ "$mongo_ready" == "true" ]] || die "MongoDB 在 120 秒内未就绪"
  docker exec "$MONGO_CONTAINER" sh -c 'exec mongosh --quiet --username "$MONGO_INITDB_ROOT_USERNAME" --password "$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin --eval "$1"' sh "$initiate_js"

  local attempt state=""
  for attempt in $(seq 1 60); do
    state="$(replica_state 2>/dev/null || true)"
    if [[ "$state" == *'"setName":"rs0"'* && "$state" == *'"primary":true'* ]]; then
      break
    fi
    sleep 2
  done
  [[ "$state" == *'"setName":"rs0"'* && "$state" == *'"primary":true'* ]] || die "rs0 在 120 秒内未成为主节点"

  append_replica_uri
  guard_target
  require_primary
  log "test-mongo 已准备为 rs0 单节点主节点"
}

build_images() {
  load_run_dir
  [[ ! -e "$RUN_DIR/migration-v2-plan.json" ]] || die "已经生成 dry-run 计划，禁止重新构建镜像"
  [[ "$(git -C "$SCRIPT_ROOT" branch --show-current)" == "Eagle233" ]] || die "迁移源码必须位于 Eagle233 分支"
  [[ -z "$(git -C "$SCRIPT_ROOT" status --porcelain)" ]] || die "迁移源码存在未提交修改"
  [[ "$(git -C "$SCRIPT_ROOT" rev-parse HEAD)" == "$(git -C "$SCRIPT_ROOT" rev-parse origin/Eagle233)" ]] || die "当前 HEAD 与 origin/Eagle233 不一致"
  local commit short migrate_image backend_image
  commit="$(git -C "$SCRIPT_ROOT" rev-parse HEAD)"
  short="${commit:0:12}"
  migrate_image="meowpick-migrate-v2:${short}"
  backend_image="meowpick-backend-v2:${short}"
  printf '%s\n' "$commit" >"$RUN_DIR/source-commit.txt"
  printf '%s\n' "$migrate_image" >"$RUN_DIR/migrate-image.txt"
  printf '%s\n' "$backend_image" >"$RUN_DIR/backend-image.txt"
  docker build --tag "$backend_image" "$SCRIPT_ROOT"
  docker tag "$backend_image" "$migrate_image"
  docker image inspect --format '{{.Id}}' "$migrate_image" >"$RUN_DIR/migrate-image-id.txt"
  docker image inspect --format '{{.Id}}' "$backend_image" >"$RUN_DIR/backend-image-id.txt"
  log "迁移和后端镜像均来自 commit $commit"
}

require_recorded_source() {
  [[ "$(git -C "$SCRIPT_ROOT" branch --show-current)" == "Eagle233" ]] || die "当前源码不在 Eagle233 分支"
  [[ -z "$(git -C "$SCRIPT_ROOT" status --porcelain)" ]] || die "当前源码存在未提交修改"
  [[ "$(git -C "$SCRIPT_ROOT" rev-parse HEAD)" == "$(<"$RUN_DIR/source-commit.txt")" ]] || die "当前源码 commit 与 dry-run 记录不一致"
  [[ "$(git -C "$SCRIPT_ROOT" rev-parse HEAD)" == "$(git -C "$SCRIPT_ROOT" rev-parse origin/Eagle233)" ]] || die "当前 HEAD 与 origin/Eagle233 不一致"
}

run_migration() {
  local mode="$1" report="$2" migrate_image
  load_run_dir
  guard_target
  require_primary
  require_backend_stopped
  [[ -s "$RUN_DIR/meowpick-before.archive.gz" ]] || die "缺少迁移前备份"
  sha256sum -c "$RUN_DIR/meowpick-before.archive.gz.sha256"
  require_recorded_source
  migrate_image="$(<"$RUN_DIR/migrate-image.txt")"
  docker image inspect "$migrate_image" >/dev/null
  [[ "$(docker image inspect --format '{{.Id}}' "$migrate_image")" == "$(<"$RUN_DIR/migrate-image-id.txt")" ]] || die "迁移镜像 ID 已变化"
  docker run --rm --user "$(id -u):$(id -g)" --network "$DOCKER_NETWORK" --volume "$BACKEND_CONFIG:/server-config/config.yaml:ro" --volume "$RUN_DIR:/migration" "$migrate_image" /app/migrate-v2 --config /server-config/config.yaml --require-host "$EXPECTED_MONGO_HOST" --require-port "$EXPECTED_MONGO_PORT" --require-db "$EXPECTED_DATABASE" "$mode" "/migration/$report"
}

dry_run() {
  run_migration --report migration-v2-plan.json
  sha256sum "$RUN_DIR/migration-v2-plan.json" >"$RUN_DIR/migration-v2-plan.json.sha256"
  cp "$RUN_DIR/migrate-image-id.txt" "$RUN_DIR/dry-run-migrate-image-id.txt"
  cp "$RUN_DIR/source-commit.txt" "$RUN_DIR/dry-run-source-commit.txt"
  chmod 600 "$RUN_DIR/migration-v2-plan.json.sha256" "$RUN_DIR/dry-run-migrate-image-id.txt" "$RUN_DIR/dry-run-source-commit.txt"
  log "请人工审核 $RUN_DIR/migration-v2-plan.json"
}

apply_plan() {
  load_run_dir
  [[ ! -e "$RUN_DIR/applied.ok" ]] || die "本次计划已经 apply，拒绝重复执行"
  [[ "${MEOWPICK_CONFIRM_APPLY:-}" == "APPLY-test-mongo-meowpick" ]] || die "请设置 MEOWPICK_CONFIRM_APPLY=APPLY-test-mongo-meowpick"
  [[ "${MEOWPICK_CONFIRM_BACKUP:-}" == "BACKUP-COPIED-OFFSERVER" ]] || die "请先把备份复制到服务器外，并设置 MEOWPICK_CONFIRM_BACKUP=BACKUP-COPIED-OFFSERVER"
  [[ -s "$RUN_DIR/migration-v2-plan.json" ]] || die "缺少 dry-run 计划"
  sha256sum -c "$RUN_DIR/migration-v2-plan.json.sha256"
  [[ "$(<"$RUN_DIR/migrate-image-id.txt")" == "$(<"$RUN_DIR/dry-run-migrate-image-id.txt")" ]] || die "dry-run 后迁移镜像记录发生变化"
  [[ "$(<"$RUN_DIR/source-commit.txt")" == "$(<"$RUN_DIR/dry-run-source-commit.txt")" ]] || die "dry-run 后源码 commit 记录发生变化"
  run_migration --apply-plan migration-v2-plan.json
  {
    printf 'plan='; awk '{print $1}' "$RUN_DIR/migration-v2-plan.json.sha256"
    printf 'commit=%s\n' "$(<"$RUN_DIR/source-commit.txt")"
    printf 'image=%s\n' "$(<"$RUN_DIR/migrate-image-id.txt")"
  } >"$RUN_DIR/applied.ok"
  chmod 600 "$RUN_DIR/applied.ok"
}

postcheck() {
  load_run_dir
  [[ -f "$RUN_DIR/applied.ok" ]] || die "尚未完成 apply"
  run_migration --report migration-v2-postcheck.json
  python3 - "$RUN_DIR/migration-v2-postcheck.json" <<'PY'
import json
import sys

with open(sys.argv[1]) as handle:
    report = json.load(handle)
checks = {
    "conflicts": report.get("conflicts"),
    "courseRepairs": report.get("courseRepairs"),
    "commentIdRepairs": report.get("commentIdRepairs"),
    "teacherTimeRepairs": report.get("teacherTimeRepairs"),
}
failed = {key: value for key, value in checks.items() if value}
if report.get("legacyMappingIdCount") != 0:
    failed["legacyMappingIdCount"] = report.get("legacyMappingIdCount")
if failed:
    raise SystemExit(f"postcheck 未通过: {failed}")
print("postcheck 通过；mappings 是完整映射清单，不要求为空")
PY
  touch "$RUN_DIR/postcheck.ok"
  chmod 600 "$RUN_DIR/postcheck.ok"
}

deploy_backend() {
  load_run_dir
  [[ -f "$RUN_DIR/postcheck.ok" ]] || die "尚未通过 postcheck"
  require_primary
  local backend_image backend_image_id compose_image
  backend_image="$(<"$RUN_DIR/backend-image.txt")"
  backend_image_id="$(<"$RUN_DIR/backend-image-id.txt")"
  docker image inspect "$backend_image" >/dev/null
  [[ "$(docker image inspect --format '{{.Id}}' "$backend_image")" == "$backend_image_id" ]] || die "后端镜像 ID 已变化"
  compose_image="$(docker compose -f "$BACKEND_COMPOSE" config --format json | python3 -c 'import json,sys; data=json.load(sys.stdin); print(data["services"]["test-meowpick-backend"]["image"])')"
  [[ "$compose_image" == "$BACKEND_IMAGE" ]] || die "后端 Compose 镜像不是 $BACKEND_IMAGE: $compose_image"
  docker tag "$backend_image" "$BACKEND_IMAGE"
  docker compose -f "$BACKEND_COMPOSE" up -d --pull never --force-recreate "$BACKEND_CONTAINER"
  sleep 3
  [[ "$(docker inspect --format '{{.State.Running}}' "$BACKEND_CONTAINER")" == "true" ]] || die "新后端容器未保持运行"
  [[ "$(docker inspect --format '{{.Image}}' "$BACKEND_CONTAINER")" == "$backend_image_id" ]] || die "新后端容器实际镜像 ID 不匹配"
  docker logs --tail 120 "$BACKEND_CONTAINER"
}

rollback_all() {
  load_run_dir
  guard_environment
  require_exact_container "$MONGO_CONTAINER"
  require_network_attachment "$MONGO_CONTAINER"
  [[ ! -e "$RUN_DIR/rolled-back.ok" ]] || die "本次迁移已经完成回滚，拒绝重复执行"
  [[ "${MEOWPICK_CONFIRM_ROLLBACK:-}" == "ROLLBACK-test-mongo-meowpick" ]] || die "请设置 MEOWPICK_CONFIRM_ROLLBACK=ROLLBACK-test-mongo-meowpick"
  [[ -s "$RUN_DIR/meowpick-before.archive.gz" ]] || die "缺少迁移前备份"
  sha256sum -c "$RUN_DIR/meowpick-before.archive.gz.sha256"
  docker stop "$BACKEND_CONTAINER" >/dev/null 2>&1 || true

  # 先恢复旧配置并拉起 MongoDB。这样 prepare-replica 中途失败、MongoDB
  # 无法启动时，rollback 仍能进入后续的备份恢复阶段。
  local config_uid config_gid config_mode compose_uid compose_gid compose_mode
  IFS=: read -r config_uid config_gid config_mode <"$RUN_DIR/config.before.meta"
  IFS=: read -r compose_uid compose_gid compose_mode <"$RUN_DIR/docker-compose.before.meta"
  sudo install -o "$config_uid" -g "$config_gid" -m "$config_mode" "$RUN_DIR/config.before.yaml" "$BACKEND_CONFIG"
  sudo install -o "$compose_uid" -g "$compose_gid" -m "$compose_mode" "$RUN_DIR/docker-compose.before.yml" "$MONGO_COMPOSE"
  sudo docker compose -f "$MONGO_COMPOSE" config --quiet
  sudo docker compose -f "$MONGO_COMPOSE" up -d "$MONGO_CONTAINER"

  local ready_attempt mongo_ready="false"
  for ready_attempt in $(seq 1 60); do
    if docker exec "$MONGO_CONTAINER" mongosh --quiet --eval 'quit(db.hello().ok === 1 ? 0 : 1)' >/dev/null 2>&1; then
      mongo_ready="true"
      break
    fi
    sleep 2
  done
  [[ "$mongo_ready" == "true" ]] || die "恢复旧 Compose 后 MongoDB 在 120 秒内未就绪"

  local backup_storage_bytes available_bytes required_bytes
  backup_storage_bytes="$(<"$RUN_DIR/meowpick-storage-size.txt")"
  available_bytes="$(df --output=avail -B1 /home/eagle233/repos/test/mongo/data | awk 'NR==2 {print $1}')"
  required_bytes="$((backup_storage_bytes + backup_storage_bytes / 2 + 1073741824))"
  [[ "$available_bytes" -gt "$required_bytes" ]] || die "可用磁盘不足：临时恢复至少要求备份 storageSize 的 1.5 倍再加 1 GiB"

  local scratch_db="meowpick_v2_restore_check" scratch_exists_js manifest_js drop_scratch_js
  scratch_exists_js='print(db.adminCommand({listDatabases:1,nameOnly:true}).databases.some(x => x.name === "meowpick_v2_restore_check") ? "yes" : "no")'
  manifest_js='const d=db.getSiblingDB("DB_NAME"); const out={}; d.getCollectionNames().sort().forEach(c => { out[c]={count:d.getCollection(c).countDocuments({}),indexes:d.getCollection(c).getIndexes().map(i => i.name).sort()}; }); print(JSON.stringify(out));'
  drop_scratch_js='db.getSiblingDB("meowpick_v2_restore_check").dropDatabase()'
  local scratch_exists
  scratch_exists="$(docker exec "$MONGO_CONTAINER" sh -c 'exec mongosh --quiet --username "$MONGO_INITDB_ROOT_USERNAME" --password "$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin --eval "$1"' sh "$scratch_exists_js")"
  if [[ "$scratch_exists" == "no" ]]; then
    if ! docker exec -i "$MONGO_CONTAINER" sh -c 'exec mongorestore --stopOnError --username "$MONGO_INITDB_ROOT_USERNAME" --password "$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin --archive --gzip --nsFrom="meowpick.*" --nsTo="meowpick_v2_restore_check.*"' <"$RUN_DIR/meowpick-before.archive.gz"; then
      docker exec "$MONGO_CONTAINER" sh -c 'exec mongosh --quiet --username "$MONGO_INITDB_ROOT_USERNAME" --password "$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin --eval "$1"' sh "$drop_scratch_js" >/dev/null 2>&1 || true
      die "临时恢复失败；已尝试删除本次创建的临时数据库，当前 meowpick 未删除"
    fi
    touch "$RUN_DIR/scratch-ready.ok"
    chmod 600 "$RUN_DIR/scratch-ready.ok"
  elif [[ ! -f "$RUN_DIR/scratch-ready.ok" ]]; then
    die "临时恢复数据库已经存在但不属于本次恢复"
  fi
  docker exec "$MONGO_CONTAINER" sh -c 'exec mongosh --quiet --username "$MONGO_INITDB_ROOT_USERNAME" --password "$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin --eval "$1"' sh "${manifest_js/DB_NAME/$scratch_db}" >"$RUN_DIR/rollback-scratch-manifest.json"
  chmod 600 "$RUN_DIR/rollback-scratch-manifest.json"

  local drop_js='db.getSiblingDB("meowpick").dropDatabase()'
  docker exec "$MONGO_CONTAINER" sh -c 'exec mongosh --quiet --username "$MONGO_INITDB_ROOT_USERNAME" --password "$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin --eval "$1"' sh "$drop_js"
  docker exec -i "$MONGO_CONTAINER" sh -c 'exec mongorestore --stopOnError --username "$MONGO_INITDB_ROOT_USERNAME" --password "$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin --archive --gzip' <"$RUN_DIR/meowpick-before.archive.gz"
  docker exec "$MONGO_CONTAINER" sh -c 'exec mongosh --quiet --username "$MONGO_INITDB_ROOT_USERNAME" --password "$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin --eval "$1"' sh "${manifest_js/DB_NAME/$EXPECTED_DATABASE}" >"$RUN_DIR/rollback-target-manifest.json"
  chmod 600 "$RUN_DIR/rollback-target-manifest.json"
  cmp "$RUN_DIR/rollback-scratch-manifest.json" "$RUN_DIR/rollback-target-manifest.json" || die "恢复后集合计数或索引清单与临时恢复不一致；保持后端停止"
  docker exec "$MONGO_CONTAINER" sh -c 'exec mongosh --quiet --username "$MONGO_INITDB_ROOT_USERNAME" --password "$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin --eval "$1"' sh "$drop_scratch_js"
  rm -f "$RUN_DIR/scratch-ready.ok"

  local old_image_id
  old_image_id="$(<"$RUN_DIR/old-backend-image-id.txt")"
  docker image inspect "$old_image_id" >/dev/null
  docker tag "$old_image_id" "$BACKEND_IMAGE"
  docker compose -f "$BACKEND_COMPOSE" up -d --pull never --force-recreate "$BACKEND_CONTAINER"
  sleep 3
  [[ "$(docker inspect --format '{{.State.Running}}' "$BACKEND_CONTAINER")" == "true" ]] || die "旧后端容器未保持运行"
  [[ "$(docker inspect --format '{{.Image}}' "$BACKEND_CONTAINER")" == "$old_image_id" ]] || die "旧后端容器实际镜像 ID 不匹配"
  printf 'rolled_back_at=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >"$RUN_DIR/rolled-back.ok"
  chmod 600 "$RUN_DIR/rolled-back.ok"
  log "数据库、Mongo Compose、后端配置和旧后端镜像已恢复"
}

usage() {
  cat <<'EOF'
用法: bash scripts/jdcloud-migrate-v2.sh <命令>

命令:
  preflight         只读检查目标容器、网络、配置和 MongoDB 拓扑
  init              创建权限为 700 的本次迁移目录
  backup            停止后端并生成、校验迁移前备份
  prepare-replica   将 test-mongo 准备成 rs0；需要确认变量
  build             从当前 Eagle233 commit 构建固定迁移/后端镜像
  dry-run           只读生成迁移计划
  apply             正式写入迁移；需要确认变量
  postcheck         只读迁移后检查
  deploy            部署与迁移工具相同 commit 的后端镜像
  rollback          恢复数据库、配置和旧镜像；需要确认变量
EOF
}

main() {
  local command="${1:-}"
  case "$command" in
    preflight) preflight ;;
    init) init_run ;;
    backup) backup ;;
    prepare-replica) prepare_replica ;;
    build) build_images ;;
    dry-run) dry_run ;;
    apply) apply_plan ;;
    postcheck) postcheck ;;
    deploy) deploy_backend ;;
    rollback) rollback_all ;;
    *) usage; exit 64 ;;
  esac
}

main "$@"
