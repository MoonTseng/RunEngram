---
name: runengram-doctor
description: Diagnose a local RunEngram installation, service, CLI identity, storage, and Codex plugin version. Use when RunEngram does not open, taskline cannot connect, or a teammate needs an installation health check.
---

# RunEngram Doctor

Run read-only checks first:

```bash
runengram status
taskline version
taskline status
test -w "${RUNENGRAM_HOME:-$HOME/.local/share/runengram}/data"
curl -fsS http://127.0.0.1:8787/healthz
```

Interpret failures:

- `runengram: command not found`: run `runengram-setup`.
- server stopped: run `runengram start`.
- health check fails while PID is alive: inspect `~/.local/state/runengram/server.log`, then `runengram restart`.
- CLI reports unregistered: follow `taskline status`; register only when it reports `registered=false`.
- invalid identity or token: repair existing local config. Never silently replace identity.
- data directory not writable: report exact path and permissions. Do not use `sudo` or recursively chmod home.

Finish with observed versions, service URL, data path, and exact remaining repair command.
