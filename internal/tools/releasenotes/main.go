// Command releasenotes prints the release notes for a tag: the changelog's section for its version, with
// links that work on the release page. It fails when the section is missing, so no release ships without one.
//
//	go run ./internal/tools/releasenotes v1.1.0 < CHANGELOG.md
package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// repo is where relative links in the changelog point, at the tag being released.
const repo = "https://github.com/typhonhilldev/adder"

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: releasenotes <tag> < CHANGELOG.md")
		os.Exit(2)
	}
	changelog, err := io.ReadAll(os.Stdin)
	if err == nil {
		var out string
		if out, err = notes(string(changelog), os.Args[1]); err == nil {
			_, err = io.WriteString(os.Stdout, out)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "releasenotes:", err)
		os.Exit(1)
	}
}

var (
	semver = regexp.MustCompile(`^v(\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?)$`)
	// relativeLink matches a Markdown link whose target is a path in the repository.
	relativeLink = regexp.MustCompile(`\]\(([^)#:\s][^):\s]*)\)`)
)

// notes returns the release notes for tag from the changelog: its section, without the heading, then a link to
// the full changelog from the section's link reference.
func notes(changelog, tag string) (string, error) {
	m := semver.FindStringSubmatch(tag)
	if m == nil {
		return "", fmt.Errorf("tag %q is not a version such as v1.2.3", tag)
	}
	version := m[1]

	var section []string
	in := false
	for line := range strings.Lines(changelog) {
		switch {
		case strings.HasPrefix(line, "## ["+version+"]"):
			in = true
			continue
		case in && (strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "[")):
			in = false
		}
		if in {
			section = append(section, line)
		}
	}
	body := strings.TrimSpace(strings.Join(section, ""))
	if body == "" {
		return "", fmt.Errorf("CHANGELOG.md has no section for %s, or it is empty", version)
	}

	compare := ""
	for line := range strings.Lines(changelog) {
		if url, ok := strings.CutPrefix(strings.TrimSpace(line), "["+version+"]: "); ok {
			compare = url
		}
	}
	if compare == "" {
		return "", fmt.Errorf("CHANGELOG.md has no link reference for [%s] at its end", version)
	}

	body = relativeLink.ReplaceAllString(body, "]("+repo+"/blob/"+tag+"/$1)")
	return body + "\n\n**Full changelog:** " + compare + "\n", nil
}
