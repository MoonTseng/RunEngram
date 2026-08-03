---
name: runengram-setup
description: Install or upgrade the complete local RunEngram runtime for Codex, including server, web UI, CLI, user-local data directories, and service controls. Use when a user asks to install, set up, upgrade, or start RunEngram.
---

# RunEngram Setup

Install complete local runtime from checksum-verified GitHub release artifacts.
The installer downloads the runtime matching plugin version by default, so a
new Skill never silently drives an older CLI.

## Procedure

1. Resolve this skill's plugin root. Installer lives at `../../scripts/install-runengram.sh` relative to this `SKILL.md`.
2. Tell user installer writes only to:
   - `~/.local/share/runengram`
   - `~/.local/state/runengram`
   - `~/.local/bin/runengram`
   - `~/.local/bin/runengram-service`
   - `~/.local/bin/taskline` (compatibility symlink)
3. Run installer. It resolves the matching plugin version automatically:

```bash
bash <plugin-root>/scripts/install-runengram.sh
```

Use an explicit override only for diagnosis or rollback:

```bash
RUNENGRAM_VERSION=v0.7.0 bash <plugin-root>/scripts/install-runengram.sh
```

4. Run:

```bash
runengram-service status
runengram version
runengram status
```

5. If healthy, run `runengram-service open`.
6. Tell user to open new Codex task after plugin upgrade so updated skills load.

## Safety

- Never use `sudo`.
- Never overwrite non-symlink files in `~/.local/bin`.
- Never kill process found only by name or broad port pattern.
- Stop on checksum mismatch.
- Keep server bound to `127.0.0.1` by default.
- Do not print API tokens or GitHub credentials.
