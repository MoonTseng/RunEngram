---
name: runengram-doctor
description: Diagnose a local RunEngram installation, service, CLI identity, storage, and Codex plugin version. Use when RunEngram does not open, runengram cannot connect, or a teammate needs an installation health check.
---

# RunEngram Doctor

Run read-only checks first:

```bash
runengram-service status
runengram version
runengram status
runengram run --help
runengram learning edit --help
test -w "${RUNENGRAM_HOME:-$HOME/.local/share/runengram}/data"
curl -fsS http://127.0.0.1:8787/healthz
```

Interpret failures:

- `runengram: command not found`: run `runengram-setup`.
- server stopped: run `runengram-service start`.
- health check fails while PID is alive: inspect `~/.local/state/runengram/server.log`, then `runengram-service restart`.
- CLI reports unregistered: follow `runengram status`; register only when it reports `registered=false`.
- invalid identity or token: repair existing local config. Never silently replace identity.
- `run` or `learning edit` command missing: plugin/runtime versions differ;
  reinstall latest release, fully restart Codex, and start a new task.
- data directory not writable: report exact path and permissions. Do not use `sudo` or recursively chmod home.

Finish with observed versions, service URL, data path, and exact remaining repair command.
