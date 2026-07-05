# ADR 0004: Project Rename To Sailwright

## Status

Accepted

## Context

The project has used multiple related names:

- `Dev Alchemy`
- `dev alchemy`
- `dev-alchemy`
- `alchemy`

The project name is now `Sailwright`, with the canonical machine-readable form
`sailwright`.

Renaming everything in one change would touch documentation, CLI behavior,
package names, import paths, artifact metadata, registry media types,
configuration keys, test fixtures, and persisted state. Some of those changes
are straightforward user-facing cleanup. Others are compatibility-sensitive and
could require migration logic, release coordination, or broad refactoring.

Without a rename policy, future changes can keep reintroducing old names in
newly touched lines, and user-facing surfaces can drift between `alchemy` and
`sailwright`.

## Decision

The repository will move to `Sailwright` as the project name and `sailwright` as
the canonical CLI command name.

Every change that touches a line containing an old project name must rename
that line to the new naming when the refactor is reasonably local and low risk.
This includes nearby identifiers, comments, tests, examples, and documentation
when they can be updated without changing unrelated behavior.

Complex renames that force broad package moves, compatibility migrations, or
large cross-cutting refactors do not have to be completed immediately. They
should be left as explicit follow-up work instead of being mixed into unrelated
changes.

User-facing surfaces must be migrated immediately when touched. This includes:

- root and nested `README.md` files
- guides and other Markdown documentation
- CLI command names, help text, examples, and error messages
- release notes and changelog entries for new changes
- user-facing configuration examples
- scripts or setup instructions that users copy and run

Old repository URLs must be replaced with, or mapped to, the new Sailwright
release URL:

- Old: https://github.com/csautter/dev-alchemy
- New: https://github.com/csautter/sailwright

The CLI command name is `sailwright`. A short alias, `sail`, is supported only
as an opt-in convenience. Users may configure it with a shell alias such as
`alias sail=sailwright`, or install a `sail` symlink that points to the
`sailwright` executable. The project must not require `sail` to exist.

## Rename Rules

When editing existing code or documentation:

1. Replace old user-facing names with `Sailwright` in prose.
2. Replace old command examples with `sailwright`.
3. Use `sailwright` for new file names, slugs, binary names, package-local
   identifiers, and generated examples when the rename is local.
4. Keep `sail` only for documentation that explains the optional alias or
   symlink.
5. Do not introduce new `alchemy`, `dev alchemy`, `dev-alchemy`, or
   `dev_alchemy` names unless preserving backward compatibility or historical
   context.
6. If an old name must remain for compatibility, add or preserve a short comment
   explaining why it still exists.

## Deferred Renames

The following rename categories may be deferred until dedicated migration work:

- Go module path changes and external import paths.
- Public package names or exported API names that would break downstream users.
- Persisted local data directories, cache keys, or state file formats.
- OCI media types, annotations, registry paths, or artifact contracts that are
  already covered by compatibility ADRs.
- Test fixture directories or golden data where renaming would obscure the
  behavior under test.
- Build, release, or deployment paths that require coordinated infrastructure
  changes.

Deferred items should not block small user-facing fixes. However, once a
deferred area is intentionally refactored, the new names should use
`sailwright` unless a compatibility boundary requires the old name.

## Consequences

- New users see one project and CLI name: `Sailwright` and `sailwright`.
- Existing compatibility-sensitive names can remain temporarily without turning
  every small change into a broad migration.
- Reviewers and coding agents have a concrete rule: rename old names in touched
  lines when it is easy, prioritize user-facing surfaces, and defer only the
  parts that need deliberate migration work.
