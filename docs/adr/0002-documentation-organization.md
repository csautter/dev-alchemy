# ADR 0002: Documentation Organization Contract

## Status

Accepted

## Context

The repository now has multiple documentation layers with different jobs:

- the root [README.md](./../../README.md) is the first-stop
  introduction for new users
- the `docs/` guides hold workflow detail, troubleshooting, and deeper command
  behavior
- component or platform directories can carry their own focused `README.md`
  files when the detail belongs next to the implementation or assets

Without an explicit rule, new user-facing details tend to accumulate in the
root README until it becomes long, repetitive, and harder to scan. That makes
onboarding worse and increases the chance that detailed command flags drift
between multiple copies of the same instructions.

## Decision

Documentation should be organized by depth and ownership:

- The root README stays short and high level.
- Detailed command behavior belongs in the closest matching guide in `docs/`
  or in a focused sub-`README.md` that lives with the relevant workflow.
- High-level docs should link to deeper docs instead of duplicating the same
  operational detail.

## Rules

When adding or changing user-facing functionality, document it in the smallest
set of places that keeps the information discoverable:

1. Update the root [README.md](./../../README.md) only when the
   change affects the high-level introduction, quick-start path, support
   matrix, or docs map.
2. Put detailed CLI flags, troubleshooting notes, workflow caveats, and
   platform-specific examples in a matching file under `docs/` whenever one
   already exists for that topic.
3. If the detail is tightly coupled to one implementation area, prefer the
   nearest deeper `README.md` in that subtree.
4. When the root README mentions a capability but intentionally omits detail,
   add a short pointer to the deeper guide instead of expanding the root doc.
5. Avoid duplicating long command examples across multiple files unless each
   copy serves a distinct audience; when duplication is necessary, keep one
   file as the primary home and align the others to it.
6. If no suitable deeper doc exists, add one in `docs/` or create a focused
   sub-README near the relevant workflow rather than turning the root README
   into the default storage location.

## Consequences

- New users get a faster project overview from the root README.
- Returning users can still find setup, debugging, and platform-specific detail
  in stable deeper locations.
- Reviewers and coding agents have a clearer rule for where documentation
  changes belong when new CLI flags or workflows are introduced.
