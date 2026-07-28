# Dependency Graph Delete Edge Implementation Plan

Task: 7fcef117-a0aa-4131-a9e5-b41441b78865

## Goal

Clicking a dependency edge in the Dependency graph should select that relationship, highlight the related dependency chain, render the selected edge with a distinct color, and show a delete icon button near the edge. Clicking the delete button removes that dependency.

## Scope

- Reuse the existing dependency delete API through `useDeleteDependency(project.id)`.
- Keep task node click behavior unchanged: clicking a task still opens `TaskEditor`.
- Keep pane click as the universal selection clear action.
- Do not change graph layout, dependency creation, server handlers, CLI commands, or schema.

## Design

- Track `selectedEdgeId` separately from `selectedTaskId`.
- Selecting a task clears selected edge state.
- Selecting an edge clears selected task state and highlights the relationship chain around the edge.
- Compute edge-chain related task ids as the union of the source and target task relationship chains.
- Use a custom React Flow edge to render:
  - the smooth step path,
  - a selected-edge color,
  - the existing arrow marker,
  - a delete icon button at the edge label position when selected.
- Delete button calls `useDeleteDependency` with `{ taskId: target, dependsOn: source }`.

## Files

- Modify `web/src/components/GraphView.tsx`
  - Import `BaseEdge`, `EdgeLabelRenderer`, `getSmoothStepPath`, `Trash2`, and `useDeleteDependency`.
  - Add selected edge state and custom edge data.
  - Add edge click and delete behavior.
  - Add helper for edge-chain related ids.
- Modify `web/src/components/GraphView.test.tsx`
  - Extend the React Flow mock to expose edge click and custom edge labels.
  - Add a failing test for edge selection, related-chain highlighting, selected-edge color, and delete button.
  - Add a pane-clear assertion for edge selection.

## Test Plan

- RED:
  - Add a GraphView test for selecting edge `b->c`, expecting node `a/b/c` highlighted, unrelated node dimmed, selected edge colored, and delete button visible.
  - Expect failure before implementation because edge selection/delete UI does not exist.
- GREEN:
  - Implement selected edge state, helper, custom edge rendering, and delete mutation.
  - Rerun the focused GraphView test.
- Full local:
  - `cd web && mise exec -- pnpm lint && mise exec -- pnpm test -- --run && mise exec -- pnpm build`
  - `cd cli && mise exec -- go test ./...`
  - `cd server && mise exec -- go test ./...`
  - `git diff --check`
- Browser smoke:
  - Start rebuilt taskline on `*:8787`.
  - Create a temporary project with two dependent tasks.
  - Open Dependency graph, select the edge, click delete, and verify the target task no longer lists the dependency.

## Acceptance Criteria

- Clicking an edge selects it and highlights the related dependency chain.
- The selected edge uses a distinct color from normal highlighted edges.
- Only the selected edge shows a delete icon button.
- Clicking delete removes the dependency through the existing API path.
- Pane click clears edge selection and hides the delete button.
