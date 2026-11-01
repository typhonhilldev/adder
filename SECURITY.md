# Security policy

## Supported versions

Security fixes go into the latest v1 release, as a new patch or minor release. Within v1, upgrading never breaks a working setup ([compatibility](docs/compatibility.md)), so upgrade with `go get -tool typhonhill.dev/adder/cmd/adder@latest`, or the same way you installed adder.

## What counts as a vulnerability

adder promises that running it on a repository is safe, even an untrusted one. Reports that break these guarantees are vulnerabilities:

- Writing or reading files outside the repository root or the configuration file's directory.
- Executing code from the repository or its configuration. Scans with `typeCheck: true` run the Go toolchain, offline and with cgo off, which compiles code but never runs it.
- Making network requests.

Packs are Go code that a repository builds into its own adder, so they run as part of that build and are trusted like any other dependency. A problem in a release itself, such as a binary that does not match its signature, is a vulnerability too.

Bugs in generated output, crashes, and wrong diagnostics are ordinary bugs; please open an issue for those.

## Reporting a vulnerability

Please do not open a public issue. Report it privately through [GitHub's private vulnerability reporting](https://github.com/typhonhilldev/adder/security/advisories/new), and include:

- What you ran, including the output of `adder --version`.
- A configuration and repository layout that reproduce the problem.
- What you expected to happen and what happened instead.

Reports are handled on a best-effort basis, with an acknowledgement within one week. Once a fix is released, the advisory is published with credit to the reporter, unless they prefer otherwise.

## Verifying a release

Each release from v1.1.0 is built by the [release workflow](.github/workflows/release.yml) from its tag, and reproducibly, so rebuilding a tag gives the same archives. Its archives have GitHub attestations of where they were built, `checksums.txt` is signed with [cosign](https://docs.sigstore.dev/cosign/), and each archive has an SPDX SBOM. To check a download:

```sh
gh attestation verify adder_1.1.0_linux_amd64.tar.gz --repo typhonhilldev/adder

cosign verify-blob checksums.txt --bundle checksums.txt.sigstore.json \
  --certificate-identity-regexp '^https://github\.com/typhonhilldev/adder/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
sha256sum --ignore-missing -c checksums.txt
```

The container image is signed the same way:

```sh
cosign verify ghcr.io/typhonhilldev/adder:1.1.0 \
  --certificate-identity-regexp '^https://github\.com/typhonhilldev/adder/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

Installing from source with `go get -tool` or `go install` is checked by Go's checksum database instead.
