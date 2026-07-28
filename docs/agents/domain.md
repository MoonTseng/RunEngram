# Domain Docs

This repository uses a single-context domain documentation layout.

## Before exploring

Read these files when they exist:

- `CONTEXT.md` at the repository root.
- Architecture decision records under `docs/adr/`.

Missing files are not errors. Create domain documentation lazily when concepts
or decisions become stable.

## Vocabulary

Use terms defined in `CONTEXT.md` in task titles, specifications, plans, tests,
and code. If a required concept is missing, verify whether it is new domain
language or an existing concept under another name.

## Architecture decisions

If proposed work conflicts with an existing ADR, surface the conflict
explicitly. Do not silently replace the existing decision.
