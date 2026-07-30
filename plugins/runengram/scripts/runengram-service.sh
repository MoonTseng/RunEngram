#!/usr/bin/env bash
set -euo pipefail

runengram_home="${RUNENGRAM_HOME:-${HOME}/.local/share/runengram}"
state_dir="${RUNENGRAM_STATE_DIR:-${HOME}/.local/state/runengram}"
data_dir="${RUNENGRAM_DATA_DIR:-${runengram_home}/data}"
listen="${TASKLINE_LISTEN:-127.0.0.1:8787}"
port="${listen##*:}"
server_bin="${runengram_home}/current/bin/runengram-server"
pid_file="${state_dir}/server.pid"
log_file="${state_dir}/server.log"
health_url="http://127.0.0.1:${port}/healthz"

mkdir -p "${state_dir}" "${data_dir}/images" "${data_dir}/docs"

running_pid() {
  [[ -f "${pid_file}" ]] || return 1
  local pid
  pid="$(tr -d '[:space:]' < "${pid_file}")"
  [[ "${pid}" =~ ^[0-9]+$ ]] || return 1
  kill -0 "${pid}" 2>/dev/null || return 1
  printf '%s\n' "${pid}"
}

start_service() {
  local pid
  if pid="$(running_pid)"; then
    echo "RunEngram already running: pid=${pid} url=http://127.0.0.1:${port}"
    return 0
  fi
  rm -f "${pid_file}"
  [[ -x "${server_bin}" ]] || {
    echo "RunEngram server missing: ${server_bin}. Run runengram-setup." >&2
    exit 2
  }
  command -v python3 >/dev/null 2>&1 || {
    echo "python3 is required to launch detached local service." >&2
    exit 2
  }
  command -v curl >/dev/null 2>&1 || {
    echo "curl is required for health checks." >&2
    exit 2
  }
  : > "${log_file}"
  pid="$(
    TASKLINE_DB="${data_dir}/runengram.db" \
    TASKLINE_IMAGES_DIR="${data_dir}/images" \
    TASKLINE_DOCS_DIR="${data_dir}/docs" \
    TASKLINE_LISTEN="${listen}" \
    python3 - "${server_bin}" "${log_file}" <<'PY'
import os
import subprocess
import sys

log = open(sys.argv[2], "ab", buffering=0)
proc = subprocess.Popen(
    [sys.argv[1]],
    stdin=subprocess.DEVNULL,
    stdout=log,
    stderr=subprocess.STDOUT,
    env=os.environ.copy(),
    start_new_session=True,
)
print(proc.pid)
PY
  )"
  printf '%s\n' "${pid}" > "${pid_file}"
  for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20; do
    if curl -fsS --max-time 1 "${health_url}" >/dev/null 2>&1; then
      echo "RunEngram started: pid=${pid} url=http://127.0.0.1:${port}"
      return 0
    fi
    kill -0 "${pid}" 2>/dev/null || {
      echo "RunEngram exited during startup. Log: ${log_file}" >&2
      tail -n 30 "${log_file}" >&2 || true
      exit 1
    }
    sleep 0.25
  done
  echo "RunEngram did not become healthy. Log: ${log_file}" >&2
  exit 1
}

stop_service() {
  local pid
  if ! pid="$(running_pid)"; then
    rm -f "${pid_file}"
    echo "RunEngram already stopped."
    return 0
  fi
  kill "${pid}"
  for _ in 1 2 3 4 5 6 7 8 9 10; do
    kill -0 "${pid}" 2>/dev/null || {
      rm -f "${pid_file}"
      echo "RunEngram stopped."
      return 0
    }
    sleep 0.25
  done
  echo "RunEngram pid ${pid} did not stop after SIGTERM; inspect ${log_file}." >&2
  exit 1
}

status_service() {
  local pid
  if pid="$(running_pid)" && curl -fsS --max-time 1 "${health_url}" >/dev/null 2>&1; then
    echo "healthy: pid=${pid} url=http://127.0.0.1:${port} data=${data_dir}"
    return 0
  fi
  echo "stopped or unhealthy: url=http://127.0.0.1:${port} log=${log_file}" >&2
  return 1
}

open_ui() {
  local url="http://127.0.0.1:${port}"
  status_service >/dev/null || start_service
  case "$(uname -s)" in
    Darwin) open "${url}" ;;
    Linux)
      command -v xdg-open >/dev/null 2>&1 || {
        echo "${url}"
        return 0
      }
      xdg-open "${url}" >/dev/null 2>&1 &
      ;;
    *) echo "${url}" ;;
  esac
}

case "${1:-}" in
  start) start_service ;;
  stop) stop_service ;;
  restart) stop_service; start_service ;;
  status) status_service ;;
  open) open_ui ;;
  *)
    echo "usage: runengram {start|stop|restart|status|open}" >&2
    exit 2
    ;;
esac
