# Integrations

adder works alongside the tools a repository already uses. Each snippet below runs `go tool adder`, the version `go.mod` pins, so every machine and CI job runs the same release.

## go generate

One directive anywhere in the module regenerates every output, since adder finds `adder.yaml` from any directory:

```go
//go:generate go tool adder derive
```

Put it in a single file, such as `doc.go` at the root, since `go generate ./...` runs each directive it finds.

## pre-commit

With [pre-commit](https://pre-commit.com), a local hook runs the adder `go.mod` pins, including a repository's [own build](extending.md#your-own-build) with its packs:

```yaml
# .pre-commit-config.yaml
repos:
  - repo: local
    hooks:
      - id: adder-check
        name: adder check
        entry: go tool adder check
        language: system
        pass_filenames: false
        always_run: true
```

Use `adder derive` as the entry instead to update the derived files on commit. pre-commit then fails the commit when one changed, so that it can be added.

A repository that does not declare adder as a tool can use the hooks adder publishes, which build adder from source at the revision given:

```yaml
repos:
  - repo: https://github.com/typhonhilldev/adder
    rev: v1.0.0
    hooks:
      - id: adder-check   # or adder-derive
```

The hooks run on every commit, whatever changed, since a change anywhere can affect a scan. In a [monorepo](commands.md#monorepos) without an `adder.yaml` at its root, give them `args: [./...]`.

## GitHub Actions

adder's action runs `adder check` and shows each problem, warning, and out-of-date output as an annotation on the pull request:

```yaml
# .github/workflows/ci.yml
jobs:
  adder:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with: {go-version-file: go.mod}
      - uses: typhonhilldev/adder@v1.2.0
```

It runs the adder `go.mod` declares as a tool, including a repository's [own build](extending.md#your-own-build) with its packs, which needs adder v1.2.0 or later. In a repository that does not declare it, the action downloads the release it is referenced by and checks it against `checksums.txt` and its attestation before running it; `version` picks another, from v1.2.0 on.

| Input | Default | Description |
| --- | --- | --- |
| `command` | `check` | The command to run, such as `derive`. |
| `args` | | Arguments after the command, such as `./... --diff`. |
| `version` | the action's release | The release to download when `go.mod` does not declare adder as a tool. |
| `working-directory` | `.` | Where to run adder. |

Reference the action by its full version, or a commit, as GitHub [recommends](https://docs.github.com/actions/reference/security/secure-use#using-third-party-actions). adder's tags never move, so there is no `@v1`: let Dependabot's `github-actions` updates propose each new release instead.

Without the action, run `adder check --format github` in a step. It fails the job when a derived file is out of date, as `--diff` shows, and annotates the pull request:

```yaml
      - run: go tool adder check --diff --format github
```

`go tool adder check ./... --format github` does the same in a [monorepo](commands.md#monorepos).

## GitLab CI

adder's job for GitLab runs `adder check` and shows problems, warnings, and out-of-date outputs in the merge request's [Code Quality](https://docs.gitlab.com/ci/testing/code_quality/) report. Include it from a release, with its integrity hash, which GitLab 17.9 and later check before running it:

```yaml
# .gitlab-ci.yml
include:
  - remote: https://raw.githubusercontent.com/typhonhilldev/adder/v1.2.0/ci/gitlab/adder.yml
    integrity: sha256-Eky4uwW1BeR2qmkJ0d61rnHxm4RjOHqSZ1jhwPx6DuA=
    inputs:
      version: 1.2.0
```

The job runs in adder's image for `version`, which has Go too. It runs the adder `go.mod` declares as a tool, including a repository's [own build](extending.md#your-own-build), and otherwise the image's. Its other inputs are `command`, default `check`, `args`, such as `./... --diff`, `stage`, default `test`, and `working-directory`. The job is called `adder`, and can be extended like any other.

Without the job, `--format gitlab` writes the same report from any job, and the usual output to the job's log:

```yaml
# .gitlab-ci.yml
adder:
  image: golang:1.27
  script:
    - go tool adder check --diff --format gitlab > gl-code-quality-report.json
  artifacts:
    when: always
    reports:
      codequality: gl-code-quality-report.json
```

A repository that does not declare adder as a tool can use the release's image, which has Go too, for scans with `typeCheck`. Pin the version, as `go.mod` would, and clear the entrypoint, which runs `adder`:

```yaml
adder:
  image:
    name: ghcr.io/typhonhilldev/adder:1.2.0
    entrypoint: [""]
  script:
    - adder check --diff --format gitlab > gl-code-quality-report.json
  artifacts:
    when: always
    reports:
      codequality: gl-code-quality-report.json
```

The image runs anywhere else the same way, such as `docker run --rm -v "$PWD:/src" ghcr.io/typhonhilldev/adder:1.2.0 check`. A repository whose scans set `typeCheck` needs its modules downloaded first, with `go mod download`, since adder never downloads them.

## Coding agents

See [coding agents](agents.md) for how agents discover and run generators, and an `AGENTS.md` section to copy into a repository.
