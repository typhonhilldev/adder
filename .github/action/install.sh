#!/usr/bin/env bash
# Finds the adder the GitHub Action runs: the one go.mod declares as a tool, which leaves the path output empty,
# or else a release, downloaded and verified, whose binary's path it outputs.
set -euo pipefail

if [ -f go.mod ] && command -v go > /dev/null && go tool -n adder > /dev/null 2>&1; then
  echo "Running the adder that go.mod declares as a tool: $(go tool adder --version)"
  exit 0
fi

version="${INPUT_VERSION:-}"
if [ -z "$version" ]; then
  if [[ "${ACTION_REF:-}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    version="$ACTION_REF"
  elif [ -n "${ACTION_REF:-}" ]; then
    # Referenced by a commit: the release tagged at it, if any.
    version=$(gh api "repos/typhonhilldev/adder/tags?per_page=100" --jq ".[] | select(.commit.sha == \"$ACTION_REF\") | .name" | head -n 1)
  fi
fi
if [ -z "$version" ]; then
  echo "::error title=adder::Cannot tell which adder release to run. Set the action's version input, such as v1.2.0, reference the action by a release, or declare adder as a tool in go.mod."
  exit 1
fi

case "${RUNNER_OS:-}" in
  Linux) os=linux ext=tar.gz bin=adder ;;
  macOS) os=darwin ext=tar.gz bin=adder ;;
  Windows) os=windows ext=zip bin=adder.exe ;;
  *) echo "::error title=adder::No adder release for the runner's operating system, ${RUNNER_OS:-unknown}."; exit 1 ;;
esac
case "${RUNNER_ARCH:-}" in
  X64) arch=amd64 ;;
  ARM64) arch=arm64 ;;
  *) echo "::error title=adder::No adder release for the runner's architecture, ${RUNNER_ARCH:-unknown}."; exit 1 ;;
esac

asset="adder_${version#v}_${os}_${arch}.${ext}"
dir="${RUNNER_TEMP:-/tmp}/adder-$version"
mkdir -p "$dir"
cd "$dir"
gh release download "$version" --repo typhonhilldev/adder --pattern "$asset" --pattern checksums.txt --clobber
# sha256sum on Linux and in Git Bash on Windows, shasum on macOS.
if command -v sha256sum > /dev/null; then sum=(sha256sum); else sum=(shasum -a 256); fi
grep " $asset\$" checksums.txt | "${sum[@]}" -c -
if ! gh attestation verify "$asset" --repo typhonhilldev/adder > /dev/null; then
  echo "::error title=adder::$asset has no valid attestation from adder's release workflow. Releases before v1.1.1 have none; use a later one."
  exit 1
fi
if [ "$ext" = zip ]; then 7z x -y "$asset" "$bin" > /dev/null; else tar -xzf "$asset" "$bin"; fi
echo "Running adder $version, downloaded and verified: $("./$bin" --version)"
echo "path=$dir/$bin" >> "$GITHUB_OUTPUT"
