# Commands

| Command | Description |
| --- | --- |
| `adder init` | Write a starter `adder.yaml`, never overwriting one. |
| `adder derive` | Run every scan and rewrite the derived files that changed. |
| `adder check` | Report derived files that are out of date, writing nothing. Exits 3 when any are. |
| `adder new <generator> [input=value...]` | Create files with a generator, then update the derived files. Without a generator, lists them. |
| `adder test [generator...]` | Run generator tests against their golden files. `--update` rewrites them. |
| `adder list` | List the generators and derived outputs. |
| `adder describe <generator>` | Show a generator's inputs, with the values its scans allow now, and the files it creates. |
| `adder explain <file>` | Show which output and template produced a file, and whether it is up to date. |
| `adder mcp` | Serve these commands to coding agents as MCP tools. `--write` adds the ones that write. See [coding agents](agents.md#mcp). |
| `adder completion <shell>` | Print a completion script for bash, zsh, fish, or powershell. |

adder finds `adder.yaml` in the current directory or the nearest parent with one, the way `go` finds `go.mod`, so commands work from anywhere in the repository.

## Flags

| Flag | Commands | Description |
| --- | --- | --- |
| `--root <dir>` | all | Repository root, holding `adder.yaml`. |
| `-c`, `--config <file>` | all | Configuration file. The root is its directory, unless `--root` says otherwise. |
| `--format <format>` | all | How to report: `text`, the default, `json` for one JSON object ([JSON output](#json-output)), or `github` or `gitlab` for CI ([CI formats](#ci-formats)). |
| `--json` | all | Short for `--format json`. |
| `-q`, `--quiet` | all | Print only problems, and diffs asked for with `--diff`. |
| `-v`, `--verbose` | all | Also print unchanged files, and for `derive` and `check`, how many items each scan found. |
| `-n`, `--dry-run` | `derive`, `new` | Show what would change without writing anything. |
| `--diff` | `derive`, `check`, `new` | Also show each change as a unified diff. Writes nothing. |
| `--out <dir>` | `derive` | Write outputs under another directory instead of the root. |
| `--inputs <file>` | `new` | Read inputs from a JSON object, or `-` for standard input. Arguments override it. |
| `--update` | `test` | Rewrite the golden files with what the generators create now. |

`adder new <generator>` takes its inputs as arguments, `name=report`, or a bool input's bare name, `withTest`, for true. Arguments override `--inputs`. `adder new <generator> --help` lists the inputs.

Completion covers generator names, their inputs, input values from `values` and `valuesFrom`, and generated files for `adder explain`. In a terminal, statuses and problem codes are in color, unless `NO_COLOR` is set or `TERM` is `dumb`.

## Monorepos

A repository can hold several `adder.yaml` files, such as one per service. Each is independent, with its own directory as its root. Scan wildcards, and the search for stale files, stop at directories with their own `adder.yaml` or `go.mod`.

`derive`, `check`, `list`, and `test` take directory patterns, as `go` does:

```console
$ adder check ./...
would be updated    services/search/handlers/names.gen.go
adder: 1 derived file is out of date; run 'adder derive' to update it
```

`./...` runs every `adder.yaml` at or below the current directory, skipping `testdata`, `vendor`, and directories starting with `.` or `_`, and `./services/search` runs one. Paths are shown from the current directory, a problem in one configuration does not stop the others, and the exit status is the most serious. `adder test ./... driver` runs the `driver` tests of every configuration that has the generator. See [`examples/monorepo`](../examples/monorepo).

## Problems and exit codes

adder reports every problem it finds, not only the first, each with its position, a stable code, and often a hint:

```text
config/defaults.env:2: want KEY=value [scanner]
	hint: comment lines start with #
```

| Exit code | Meaning |
| --- | --- |
| `0` | Success. |
| `1` | adder ran and found a problem. |
| `2` | adder was invoked incorrectly, such as with an unknown flag or an invalid input. |
| `3` | `adder check` found derived files that are out of date. |

Warnings, such as the use of something [deprecated](compatibility.md#deprecation), are printed the same way with `warning:` before them, and never change the exit code:

```text
adder.yaml:3:3: warning: generator "driver" is deprecated: use adder new store [deprecated]
```

The codes are listed in [design 4](design.md#4-structured-diagnostics).

## JSON output

With `--format json`, every command that runs prints exactly one JSON object on standard output and nothing on standard error, whether it succeeds or fails, with the same exit codes. Help, shell completion, and `adder new` without a generator print text as usual; `adder list --format json` lists the generators. Scripts and coding agents can drive adder without parsing text:

```console
$ adder new command name=report withTest --dry-run --format json
{
  "generator": "command",
  "dryRun": true,
  "changes": [
    {"path": "commands/report/report.go", "status": "created"},
    {"path": "commands/report/report_test.go", "status": "created"}
  ],
  "message": "Write the command in commands/report/report.go.",
  "warnings": [],
  "problems": []
}
```

| Command | Fields |
| --- | --- |
| `derive` | `dryRun`, `changes`, `problems` |
| `check` | `changes`, `problems` |
| `new` | `generator`, `dryRun`, `changes`, `message`, `problems` |
| `init` | `changes`, `problems` |
| `list` | `generators`, each with `name`, `description`, `pos`, and when they apply `pack`, `use`, and `deprecated`; `outputs`; `problems` |
| `describe` | `name`, `description`, `inputs`, `files`, `message`, `derive`, `deprecated` when it applies, `inputSchema`, `problems` |
| `explain` | `path`, `generated`, `source`, `output`, `scans`, `status`, `problems` |
| `test` | `tests`, each with `generator`, `name`, `passed`, and `changes`, then `problems` |

- `changes` lists every file in the plan, with its `path` and a `status`: `unchanged`, `created`, `updated`, `skipped`, or `removed`. With `--diff`, a change that writes or removes a file has a `diff`.
- `warnings` is always present, and empty when there are none. Warnings have the same fields as problems, and report something that did not stop the command, such as the use of something [deprecated](compatibility.md#deprecation), with the code `deprecated`.
- `problems` is always present, and empty on success. Each has a `code` and a `message`, and when known, a `pos` with `file`, `line`, and `column`, and `pack` for a file in a pack, and a `hint`.
- Every object ends with `warnings` and `problems`, after the fields in the table.
- With directory patterns, the object has `configs`, one entry per configuration with its `config` path and the command's fields, and `problems` for anything that concerns no one configuration.
- `describe`'s `inputSchema` is a JSON Schema (draft 2020-12) for the object `--inputs` accepts, the shape MCP uses for tool inputs.

A coding agent can find and run any generator this way: `adder list --format json`, then `adder describe <generator> --format json`, then `adder new <generator> --inputs - --dry-run --format json` to preview, and the same without `--dry-run` to write.

`--json` is short for `--format json`, and is planned for removal in v2.

## CI formats

`--format github` and `--format gitlab` report problems, warnings, and the outputs `adder check` finds out of date where the CI system shows them, with paths relative to the repository root (`GITHUB_WORKSPACE` or `CI_PROJECT_DIR`). Exit codes are the same as with any other format.

- `github` prints the usual text output, plus a [workflow command](https://docs.github.com/actions/reference/workflows-and-actions/workflow-commands) for each finding, which GitHub shows as an annotation on the pull request: `::error file=drivers/drivers.gen.go,title=adder%3A out-of-date::drivers/drivers.gen.go is out of date; run adder derive`. Problems are errors and warnings are warnings, titled with their code.
- `gitlab` prints only a [Code Quality](https://docs.gitlab.com/ci/testing/code_quality/) report on standard output, and the text output on standard error. Problems are `major`, warnings `minor`, and each entry has a fingerprint so GitLab can follow it across runs.

An out-of-date output is titled `out-of-date`, a name of these formats rather than a diagnostic code. [Integrations](integrations.md) shows both in CI jobs.
