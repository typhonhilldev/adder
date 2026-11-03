# AGENTS.md

Guidance for coding agents working on adder itself. For using adder in another repository, see [docs/agents.md](docs/agents.md).

adder is a Go code generator: `adder.yaml` declares scans of the repository, templates render derived files from them, and generators scaffold new files. Read these before changing anything they cover:

- [CONTRIBUTING.md](CONTRIBUTING.md): building, testing, fixtures, schemas, and releasing. Its rules apply to agents as written.
- [docs/design.md](docs/design.md): every decision and its reasons. Check the relevant decision before changing behavior, and follow it, or propose changing it.
- [docs/compatibility.md](docs/compatibility.md): what v1 promises.

## Done means green

A change is done when the checks in [Getting started](CONTRIBUTING.md#getting-started) pass, and `GOOS=windows golangci-lint run` too. CI runs the tests on Linux, Windows, and macOS with both supported Go releases, so paths go through `filepath`, and scripts match positions with either separator.

## The v1 surface is permanent

Commands, flags, diagnostic codes, exit codes, JSON fields, `adder.yaml` keys, template functions, and the generated-code header are frozen for v1. Additions are allowed and become permanent the moment they ship; a replacement is deprecated, never removed or renamed ([deprecation](docs/compatibility.md#deprecation)). So, before writing code that adds to the surface, settle its shape with the maintainer, and record it as a design decision.

## What a feature carries

One pull request is one slice: a feature or fix with everything it needs.

- **Tests.** Table-driven, with a `testscript` section in `cmd/adder/testdata/script` for anything a command prints or how it exits. A test of a fix fails without it.
- **A design decision** for anything that adds to the surface: a new `### N. Title (planned, launch)` in `docs/design.md`, or `(planned, vX.Y)` after the launch, saying what it does, then `**Why:**`, then a `Considered | Why not` table of the alternatives.
- **Reference docs**: the flag in `docs/commands.md`, a JSON field in its table there, a code in the codes table of design decision 4, a key in `docs/configuration.md` and the schema. `adder mcp` passes each tool fixed arguments, so the decision says whether a new flag becomes a tool argument.
- **`CHANGELOG.md`** under `[Unreleased]`, in the sections of Keep a Changelog, written for users: what they can now do, not how it was built.
- **`ROADMAP.md`**: link the row to its decision. The row stays 🔜 until the release that ships it marks it ✅.

## Style

- Code reads like the code around it, in naming, idiom, and comment density, with comments as [Code](CONTRIBUTING.md#code) describes. Inline a helper that would have one caller.
- Prose, in docs and comments alike, uses commas, parentheses, or separate sentences instead of em dashes, and states what adder does in the present tense.

## Git and pull requests

- Commit messages are a short imperative subject and, when useful, a body saying why. No AI attribution trailers.
- Push, open a pull request, or comment on GitHub only once the maintainer says so. The pull request template's checklist is the bar.
- The maintainer merges. Wait for every check, Windows and macOS included, before calling a pull request ready.

## Releases

The maintainer tags; an agent prepares. A release is a pull request that moves `[Unreleased]` into a version section, as [Releasing](CONTRIBUTING.md#releasing) lists, then the maintainer tags its merge commit and the release workflow publishes everything. Tags are immutable: a mistake becomes a new patch release. Leave tags and GitHub releases to the maintainer and the workflow. Once a release is out, run the checks in [After a release](CONTRIBUTING.md#after-a-release).
