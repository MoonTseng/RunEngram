---
name: runengram-setup
description: Install or upgrade the complete local RunEngram runtime for Codex, including server, web UI, CLI, user-local data directories, and service controls. Use when a user asks to install, set up, upgrade, or start RunEngram.
---

# RunEngram Setup

Install complete local runtime from checksum-verified GitHub release artifacts.

## Procedure

1. Resolve this skill's plugin root. Installer lives at `../../scripts/install-runengram.sh` relative to this `SKILL.md`.
2. Tell user installer writes only to:
   - `~/.local/share/runengram`
   - `~/.local/state/runengram`
   - `~/.local/bin/taskline`
   - `~/.local/bin/runengram`
3. Run installer:

```bash
bash <plugin-root>/scripts/install-runengram.sh
```

To install specific version:

```bash
RUNENGRAM_VERSION=v0.1.2 bash <plugin-root>/scripts/install-runengram.sh
```

4. Run:

```bash
runengram status
taskline status
```

5. If healthy, run `runengram open`.
6. Tell user to open new Codex task after plugin upgrade so updated skills load.

## Safety

- Never use `sudo`.
- Never overwrite non-symlink files in `~/.local/bin`.
- Never kill process found only by name or broad port pattern.
- Stop on checksum mismatch.
- Keep server bound to `127.0.0.1` by default.
- Do not print API tokens or GitHub credentials.
