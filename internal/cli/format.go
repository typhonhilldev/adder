package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"typhonhill.dev/adder/internal/diag"
	"typhonhill.dev/adder/internal/plan"
)

// The values of --format.
const (
	formatText   = "text"
	formatJSON   = "json"
	formatGitHub = "github"
	formatGitLab = "gitlab"
)

var formats = []string{formatText, formatJSON, formatGitHub, formatGitLab}

// outOfDate names the findings for outputs that adder check found out of date. It belongs to the CI formats,
// not to the diagnostic codes, since every other format reports these outputs as changes.
const outOfDate = "out-of-date"

// finding is one problem, warning, or out-of-date output, for the CI formats.
type finding struct {
	warning bool
	code    string
	file    string // slash-separated, relative to the repository root; empty without a file in the repository
	line    int
	column  int
	message string
	hint    string
}

// ciReport is the part of a JSON report the CI formats read, for one configuration or for several.
type ciReport struct {
	Config  string `json:"config"`
	Changes []struct {
		Path   string `json:"path"`
		Status string `json:"status"`
	} `json:"changes"`
	Warnings diag.List  `json:"warnings"`
	Problems diag.List  `json:"problems"`
	Configs  []ciReport `json:"configs"`
}

// findings returns what a run found, read from the report --json would print, in order. Out-of-date outputs are
// included for adder check. Paths are made relative to base, the repository root, from the current directory
// for positions and from root, the configuration's directory, for changes.
func findings(rep report, warnings diag.List, err error, check bool, root, base string) []finding {
	var buf bytes.Buffer
	writeReport(&buf, rep, warnings, err)
	var r ciReport
	if json.Unmarshal(buf.Bytes(), &r) != nil {
		return nil
	}
	var out []finding
	var add func(r ciReport, root string)
	add = func(r ciReport, root string) {
		for _, d := range r.Warnings {
			out = append(out, diagFinding(d, true, base))
		}
		for _, d := range r.Problems {
			out = append(out, diagFinding(d, false, base))
		}
		if check {
			for _, c := range r.Changes {
				if c.Status != plan.Unchanged.String() {
					out = append(out, finding{code: outOfDate, file: relTo(base, filepath.Join(root, filepath.FromSlash(c.Path))), message: c.Path + " is out of date; run adder derive"})
				}
			}
		}
		for _, c := range r.Configs {
			add(c, filepath.Dir(c.Config))
		}
	}
	add(r, root)
	return out
}

func diagFinding(d *diag.Diagnostic, warning bool, base string) finding {
	f := finding{warning: warning, code: string(d.Code), line: d.Pos.Line, column: d.Pos.Column, message: d.Message}
	switch {
	case d.Pos.Pack != "":
		f.message = d.Pos.String() + ": " + d.Message // the file is in the pack, not the repository
		f.line, f.column = 0, 0
	case d.Pos.File != "":
		f.file = relTo(base, d.Pos.File)
	}
	f.hint = d.Hint
	return f
}

// relTo returns name, relative to the current directory, as a slash-separated path relative to base.
func relTo(base, name string) string {
	abs, err := filepath.Abs(name)
	if err != nil {
		return filepath.ToSlash(name)
	}
	rel, err := filepath.Rel(base, abs)
	if err != nil {
		return filepath.ToSlash(name)
	}
	return filepath.ToSlash(rel)
}

// ciBase returns the repository root that paths in the CI formats are relative to: the one the CI system names,
// or the current directory.
func ciBase(format string) string {
	name := map[string]string{formatGitHub: "GITHUB_WORKSPACE", formatGitLab: "CI_PROJECT_DIR"}[format]
	if dir := os.Getenv(name); name != "" && dir != "" {
		return dir
	}
	dir, _ := os.Getwd()
	return dir
}

// writeGitHub writes each finding as a GitHub Actions workflow command, which GitHub shows as an annotation.
func writeGitHub(w io.Writer, list []finding) {
	for _, f := range list {
		level := "error"
		if f.warning {
			level = "warning"
		}
		var props []string
		if f.file != "" {
			props = append(props, "file="+escapeProperty(f.file))
			if f.line > 0 {
				props = append(props, "line="+strconv.Itoa(f.line))
				if f.column > 0 {
					props = append(props, "col="+strconv.Itoa(f.column))
				}
			}
		}
		props = append(props, "title="+escapeProperty("adder: "+f.code))
		message := f.message
		if f.hint != "" {
			message += "\nhint: " + f.hint
		}
		fmt.Fprintf(w, "::%s %s::%s\n", level, strings.Join(props, ","), escapeData(message))
	}
}

// escapeData escapes a workflow command's message, as GitHub's toolkit does.
func escapeData(s string) string {
	return strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A").Replace(s)
}

// escapeProperty escapes a workflow command's property value, as GitHub's toolkit does.
func escapeProperty(s string) string {
	return strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A", ":", "%3A", ",", "%2C").Replace(s)
}

// codeQuality is one entry of a GitLab Code Quality report.
type codeQuality struct {
	Description string `json:"description"`
	CheckName   string `json:"check_name"`
	Fingerprint string `json:"fingerprint"`
	Severity    string `json:"severity"`
	Location    struct {
		Path  string `json:"path"`
		Lines struct {
			Begin int `json:"begin"`
		} `json:"lines"`
	} `json:"location"`
}

// writeGitLab writes the findings as a GitLab Code Quality report. A finding without a file is placed on
// fallback, the configuration file.
func writeGitLab(w io.Writer, list []finding, fallback string) {
	report := make([]codeQuality, 0, len(list))
	for _, f := range list {
		var e codeQuality
		e.Description = f.message + " [" + f.code + "]"
		if f.hint != "" {
			e.Description += "; hint: " + f.hint
		}
		e.CheckName = f.code
		e.Severity = "major"
		if f.warning {
			e.Severity = "minor"
		}
		e.Location.Path = f.file
		if e.Location.Path == "" {
			e.Location.Path = fallback
		}
		e.Location.Lines.Begin = max(f.line, 1)
		sum := sha256.Sum256([]byte(strings.Join([]string{f.code, e.Location.Path, strconv.Itoa(f.line), f.message}, "\x00")))
		e.Fingerprint = hex.EncodeToString(sum[:])
		report = append(report, e)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	_ = enc.Encode(report) // nothing useful can be done when standard output fails
}

// formatArg returns the output format args ask for, before they are parsed, so that mistakes found before the
// flags are parsed are reported in it too. An unknown format is text.
func formatArg(args []string) string {
	format := formatText
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			return format
		case arg == "--json":
			format = formatJSON
		case strings.HasPrefix(arg, "--json="):
			switch on, _ := strconv.ParseBool(strings.TrimPrefix(arg, "--json=")); {
			case on:
				format = formatJSON
			case format == formatJSON:
				format = formatText
			}
		case arg == "--format" && i+1 < len(args):
			i++
			if slices.Contains(formats, args[i]) {
				format = args[i]
			}
		case strings.HasPrefix(arg, "--format="):
			if value := strings.TrimPrefix(arg, "--format="); slices.Contains(formats, value) {
				format = value
			}
		}
	}
	return format
}

// fallbackPath returns the configuration file, relative to base, for findings without a file.
func fallbackPath(config, base string) string {
	if config == "" {
		return "adder.yaml"
	}
	return relTo(base, config)
}
