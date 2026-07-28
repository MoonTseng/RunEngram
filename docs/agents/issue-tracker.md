# Issue tracker: Local Markdown

Issues, specs, and task breakdowns for this repository live as Markdown files
under `.scratch/`. This is a personal Agent workspace; no GitLab issue is
required.

## Conventions

- One effort per directory: `.scratch/<effort-slug>/`.
- The specification is `.scratch/<effort-slug>/spec.md`.
- Implementation tickets are stored one per file under
  `.scratch/<effort-slug>/issues/<NN>-<slug>.md`.
- Ticket numbers start at `01` and follow dependency order.
- Triage state is recorded as a `Status:` line near the top of each ticket.
- Comments and execution history are appended under a `## Comments` heading.
- Stable domain knowledge must move to `CONTEXT.md` or `docs/adr/`; `.scratch/`
  remains a working area.

## When a skill says "publish to the issue tracker"

Create or update a Markdown file under `.scratch/<effort-slug>/`.

## When a skill says "fetch the relevant ticket"

Read the referenced Markdown file. The task identity is its repository-relative
path.

## Wayfinding operations

The map is one file with one child file per decision ticket.

- **Map**: `.scratch/<effort>/map.md`.
- **Child ticket**:
  `.scratch/<effort>/issues/<NN>-<slug>.md`.
- **Type**: a `Type:` line containing `research`, `prototype`, `grilling`, or
  `task`.
- **State**: a `Status:` line containing `claimed` or `resolved`.
- **Blocking**: a `Blocked by: NN, NN` line near the top.
- **Frontier**: open, unblocked, unclaimed tickets sorted by number.
- **Claim**: set `Status: claimed` before starting work.
- **Resolve**: append the result under `## Answer`, set `Status: resolved`, and
  add a short context pointer to the map's decisions section.
