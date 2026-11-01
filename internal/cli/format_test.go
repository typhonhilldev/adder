package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"typhonhill.dev/adder/internal/diag"
)

func TestWriteGitHub(t *testing.T) {
	tests := []struct {
		name string
		list []finding
		want string
	}{
		{
			name: "a problem with a position",
			list: []finding{{code: "go-syntax", file: "api/routes.go", line: 3, column: 2, message: "expected 'IDENT'"}},
			want: "::error file=api/routes.go,line=3,col=2,title=adder%3A go-syntax::expected 'IDENT'\n",
		},
		{
			name: "a warning with a hint",
			list: []finding{{warning: true, code: "deprecated", file: "adder.yaml", line: 7, message: "old is deprecated", hint: "use new"}},
			want: "::warning file=adder.yaml,line=7,title=adder%3A deprecated::old is deprecated%0Ahint: use new\n",
		},
		{
			name: "without a file, the position is left out",
			list: []finding{{code: "usage", line: 4, message: "unknown flag"}},
			want: "::error title=adder%3A usage::unknown flag\n",
		},
		{
			name: "escaping",
			list: []finding{{code: "invalid-value", file: "a,b:c%.yaml", message: "100% wrong\r\nreally"}},
			want: "::error file=a%2Cb%3Ac%25.yaml,title=adder%3A invalid-value::100%25 wrong%0D%0Areally\n",
		},
		{name: "nothing", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writeGitHub(&buf, tt.list)
			if got := buf.String(); got != tt.want {
				t.Errorf("writeGitHub() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

func TestWriteGitLab(t *testing.T) {
	type entry struct {
		Description string `json:"description"`
		CheckName   string `json:"check_name"`
		Severity    string `json:"severity"`
		Location    struct {
			Path  string `json:"path"`
			Lines struct {
				Begin int `json:"begin"`
			} `json:"lines"`
		} `json:"location"`
	}
	at := func(description, check, severity, path string, line int) entry {
		e := entry{Description: description, CheckName: check, Severity: severity}
		e.Location.Path, e.Location.Lines.Begin = path, line
		return e
	}
	tests := []struct {
		name string
		list []finding
		want []entry
	}{
		{
			name: "a problem and a warning",
			list: []finding{
				{code: "go-syntax", file: "api/routes.go", line: 3, column: 2, message: "expected 'IDENT'"},
				{warning: true, code: "deprecated", file: "adder.yaml", line: 7, message: "old is deprecated", hint: "use new"},
			},
			want: []entry{
				at("expected 'IDENT' [go-syntax]", "go-syntax", "major", "api/routes.go", 3),
				at("old is deprecated [deprecated]; hint: use new", "deprecated", "minor", "adder.yaml", 7),
			},
		},
		{
			name: "without a file or line, on the configuration's first line",
			list: []finding{{code: outOfDate, message: "x.gen.go is out of date; run adder derive"}},
			want: []entry{at("x.gen.go is out of date; run adder derive [out-of-date]", outOfDate, "major", "services/api/adder.yaml", 1)},
		},
		{name: "nothing", want: []entry{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writeGitLab(&buf, tt.list, "services/api/adder.yaml")
			var got []entry
			if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
				t.Fatalf("%v\n%s", err, buf.String())
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("writeGitLab() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestGitLabFingerprints(t *testing.T) {
	fingerprint := func(f finding) string {
		var buf bytes.Buffer
		writeGitLab(&buf, []finding{f}, "adder.yaml")
		var got []struct{ Fingerprint string }
		if err := json.Unmarshal(buf.Bytes(), &got); err != nil || len(got) != 1 {
			t.Fatalf("%v\n%s", err, buf.String())
		}
		return got[0].Fingerprint
	}
	base := finding{code: "go-syntax", file: "a.go", line: 3, column: 2, message: "bad"}
	tests := []struct {
		name  string
		other finding
		same  bool
	}{
		{name: "the same finding", other: base, same: true},
		{name: "another column", other: finding{code: "go-syntax", file: "a.go", line: 3, column: 9, message: "bad"}, same: true},
		{name: "another line", other: finding{code: "go-syntax", file: "a.go", line: 4, column: 2, message: "bad"}},
		{name: "another file", other: finding{code: "go-syntax", file: "b.go", line: 3, column: 2, message: "bad"}},
		{name: "another code", other: finding{code: "invalid-output", file: "a.go", line: 3, column: 2, message: "bad"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if same := fingerprint(base) == fingerprint(tt.other); same != tt.same {
				t.Errorf("same fingerprint = %v, want %v", same, tt.same)
			}
		})
	}
}

func TestFindings(t *testing.T) {
	base := t.TempDir()
	t.Chdir(base)
	tests := []struct {
		name     string
		rep      report
		warnings diag.List
		err      error
		check    bool
		root     string
		want     []finding
	}{
		{
			name:     "warnings, then problems, at their positions",
			rep:      &planReport{Changes: []changeReport{}},
			warnings: diag.List{{Code: diag.Deprecated, Pos: diag.Pos{File: "adder.yaml", Line: 2}, Message: "old"}},
			err:      &diag.Diagnostic{Code: diag.GoSyntax, Pos: diag.Pos{File: filepath.Join("a", "x.go"), Line: 3, Column: 4}, Message: "bad", Hint: "fix it"},
			want: []finding{
				{warning: true, code: "deprecated", file: "adder.yaml", line: 2, message: "old"},
				{code: "go-syntax", file: "a/x.go", line: 3, column: 4, message: "bad", hint: "fix it"},
			},
		},
		{
			name: "a position in a pack has no file in the repository",
			rep:  &planReport{Changes: []changeReport{}},
			err:  &diag.Diagnostic{Code: diag.TemplateExec, Pos: diag.Pos{Pack: "store", File: "t.tmpl", Line: 1, Column: 4}, Message: "bad"},
			want: []finding{{code: "template-exec", message: "pack store: t.tmpl:1:4: bad"}},
		},
		{
			name:  "out-of-date outputs of check, relative to the configuration's root",
			rep:   &planReport{Changes: []changeReport{{Path: "x/a.gen.go", Status: 2}, {Path: "x/b.gen.go"}}},
			err:   &staleError{count: 1},
			check: true,
			root:  "svc",
			want:  []finding{{code: outOfDate, file: "svc/x/a.gen.go", message: "x/a.gen.go is out of date; run adder derive"}},
		},
		{
			name: "with several configurations, each one's out-of-date outputs and problems",
			rep: &multiReport{Configs: []json.RawMessage{
				json.RawMessage(`{"config":"svc/adder.yaml","changes":[{"path":"a.gen.go","status":"removed"}],"warnings":[],"problems":[]}`),
				json.RawMessage(`{"config":"web/adder.yaml","changes":[],"warnings":[],"problems":[{"code":"unknown-field","message":"bad","pos":{"file":"web/adder.yaml","line":3}}]}`),
			}},
			check: true,
			want: []finding{
				{code: outOfDate, file: "svc/a.gen.go", message: "a.gen.go is out of date; run adder derive"},
				{code: "unknown-field", file: "web/adder.yaml", line: 3, message: "bad"},
			},
		},
		{
			name: "changes are not findings for other commands",
			rep:  &planReport{Changes: []changeReport{{Path: "x/a.gen.go", Status: 1}}},
			root: "svc",
		},
		{
			name: "usage mistakes",
			err:  &usageError{errors.New("unknown flag: --x"), "adder check"},
			want: []finding{{code: "usage", message: "unknown flag: --x", hint: "run 'adder check --help' for usage"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findings(tt.rep, tt.warnings, tt.err, tt.check, tt.root, base)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findings() =\n%+v\nwant\n%+v", got, tt.want)
			}
		})
	}
}
