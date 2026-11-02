package cli

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mcpSession starts adder mcp in root, with the mcp command's own args, through Main, and returns a client
// connected to it.
func mcpSession(t *testing.T, root string, args ...string) *mcp.ClientSession {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	clientReads, serverWrites := io.Pipe()
	serverReads, clientWrites := io.Pipe()
	done := make(chan int)
	go func() {
		done <- Main(ctx, "v0.0.0-test", nil, append([]string{"--root", root, "mcp"}, args...), serverReads, serverWrites, io.Discard)
		// A server that stops early, such as on a usage mistake, fails the client instead of leaving it waiting.
		serverWrites.Close()
		serverReads.Close()
	}()
	client := mcp.NewClient(&mcp.Implementation{Name: "test"}, nil)
	session, err := client.Connect(ctx, &mcp.IOTransport{Reader: clientReads, Writer: clientWrites}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		session.Close()
		clientWrites.Close()
		cancel()
		<-done
	})
	return session
}

// call calls a tool and returns its result's structured content decoded, with whether the result is an error.
func call(t *testing.T, session *mcp.ClientSession, name string, args any) (map[string]any, bool) {
	t.Helper()
	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s) error = %v", name, err)
	}
	var out map[string]any
	if res.StructuredContent != nil {
		data, _ := json.Marshal(res.StructuredContent)
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("structured content is not an object: %s", data)
		}
	}
	return out, res.IsError
}

func toolNames(t *testing.T, session *mcp.ClientSession) []string {
	t.Helper()
	res, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	slices.Sort(names)
	return names
}

func driversRepo(t *testing.T) string {
	t.Helper()
	root := copyDir(t, filepath.Join("..", "scaffold", "testdata", "drivers", "repo"))
	if code, _, errOut := run(t.Context(), "--root", root, "derive"); code != ExitOK {
		t.Fatal(errOut)
	}
	return root
}

func TestMCPTools(t *testing.T) {
	readOnly := []string{"check", "describe", "explain", "list", "preview", "test"}
	tests := []struct {
		name string
		args []string // the mcp command's
		want []string
	}{
		{name: "read-only", want: readOnly},
		{name: "with --write", args: []string{"--write"}, want: slices.Sorted(slices.Values(append(slices.Clone(readOnly), "derive", "new-driver")))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := driversRepo(t)
			session := mcpSession(t, root, tt.args...)
			if got := toolNames(t, session); !slices.Equal(got, tt.want) {
				t.Errorf("tools = %v, want %v", got, tt.want)
			}
			if !strings.Contains(session.InitializeResult().Instructions, "DO NOT EDIT") {
				t.Errorf("instructions = %q", session.InitializeResult().Instructions)
			}
		})
	}
}

func TestMCPCalls(t *testing.T) {
	gcs := filepath.Join("drivers", "gcs", "gcs.go")
	tests := []struct {
		name  string
		write bool
		setup func(t *testing.T, root string)
		tool  string
		args  any
		// want lists fields of the result and the JSON each must contain.
		want      map[string]string
		wantError bool
		wantFile  string // a file that exists afterwards, relative to the root
		wantNo    string // a file that does not
	}{
		{name: "list", tool: "list", want: map[string]string{"generators": `"name":"driver"`}},
		{name: "describe", tool: "describe", args: map[string]any{"generator": "driver"}, want: map[string]string{"inputSchema": `"required":["name"]`}},
		{name: "describe an unknown generator", tool: "describe", args: map[string]any{"generator": "nope"}, want: map[string]string{"problems": `"code":"unknown-generator"`}, wantError: true},
		{name: "explain", tool: "explain", args: map[string]any{"path": "drivers/drivers.gen.go"}, want: map[string]string{"generated": `true`}},
		{name: "check, current", tool: "check", want: map[string]string{"changes": `"status":"unchanged"`}},
		{
			name: "check, out of date, is not an error",
			setup: func(t *testing.T, root string) {
				writeFile(t, filepath.Join(root, "drivers", "blob", "blob.go"), "package blob\n\nfunc New() {}\n")
			},
			tool: "check", args: map[string]any{"diff": true},
			want: map[string]string{"changes": `"diff":`},
		},
		{name: "test", tool: "test", want: map[string]string{"tests": `[]`}},
		{
			name: "preview writes nothing",
			tool: "preview", args: map[string]any{"generator": "driver", "inputs": map[string]any{"name": "gcs"}, "diff": true},
			want:   map[string]string{"dryRun": `true`, "changes": `"status":"created"`},
			wantNo: gcs,
		},
		{name: "preview without a required input", tool: "preview", args: map[string]any{"generator": "driver"}, want: map[string]string{"problems": `"code":"missing-input"`}, wantError: true},
		{name: "preview of a flag", tool: "preview", args: map[string]any{"generator": "--json"}, wantError: true},
		{name: "unknown arguments", tool: "check", args: map[string]any{"nope": true}, wantError: true},
		{name: "a generator's tool", write: true, tool: "new-driver", args: map[string]any{"name": "gcs"}, want: map[string]string{"generator": `"driver"`}, wantFile: gcs},
		{
			name:  "derive",
			write: true,
			setup: func(t *testing.T, root string) {
				writeFile(t, filepath.Join(root, "drivers", "blob", "blob.go"), "package blob\n\nfunc New() {}\n")
			},
			tool: "derive",
			want: map[string]string{"changes": `"status":"updated"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := driversRepo(t)
			if tt.setup != nil {
				tt.setup(t, root)
			}
			var args []string
			if tt.write {
				args = append(args, "--write")
			}
			got, isError := call(t, mcpSession(t, root, args...), tt.tool, tt.args)
			if isError != tt.wantError {
				t.Errorf("error result = %v, want %v: %v", isError, tt.wantError, got)
			}
			for field, want := range tt.want {
				data, _ := json.Marshal(got[field])
				if !strings.Contains(string(data), want) {
					t.Errorf("%s = %s, want it to contain %s", field, data, want)
				}
			}
			if tt.wantFile != "" && !exists(t, filepath.Join(root, tt.wantFile)) {
				t.Errorf("%s was not created", tt.wantFile)
			}
			if tt.wantNo != "" && exists(t, filepath.Join(root, tt.wantNo)) {
				t.Errorf("%s was created", tt.wantNo)
			}
		})
	}
}

func TestMCPFollowsTheConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		change func(t *testing.T, root string)
		want   []string // the generator tools afterwards
	}{
		{
			name: "a generator added",
			change: func(t *testing.T, root string) {
				appendFile(t, filepath.Join(root, "adder.yaml"), "  doc:\n    files: [{template: templates/new/README.md.tmpl, path: doc.md}]\n")
			},
			want: []string{"new-doc", "new-driver"},
		},
		{
			name:   "the configuration gone",
			change: func(t *testing.T, root string) { os.Remove(filepath.Join(root, "adder.yaml")) },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := driversRepo(t)
			session := mcpSession(t, root, "--write")
			toolNames(t, session)
			tt.change(t, root)
			var got []string
			for _, name := range toolNames(t, session) {
				if strings.HasPrefix(name, "new-") {
					got = append(got, name)
				}
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("generator tools = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMCPWithoutConfiguration checks that the server starts in a directory without adder.yaml and reports that
// through its tools. It has a single case, so it is not a table.
func TestMCPWithoutConfiguration(t *testing.T) {
	session := mcpSession(t, t.TempDir())
	got, isError := call(t, session, "list", nil)
	data, _ := json.Marshal(got["problems"])
	if !isError || !strings.Contains(string(data), `"code":"io"`) {
		t.Errorf("list = %v, error %v, want an io problem", got, isError)
	}
}

func appendFile(t *testing.T, path, content string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
}
