package main

import "testing"

const changelog = `# Changelog

## [Unreleased]

### Added

- Something not yet released.

## [1.1.0] - 2026-11-02

### Added

- Releases, as [compatibility](docs/compatibility.md#deprecation) describes, and the [examples](examples/).
- A link that is already absolute: [Go](https://go.dev), and one to a heading: [below](#fixed).

### Fixed

- A fix.

## [1.0.0] - 2026-11-01

### Added

- The first stable release.

## [0.9.0] - 2026-10-01

## [0.8.0] - 2026-09-01

- Released without a link reference.

[Unreleased]: https://github.com/typhonhilldev/adder/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/typhonhilldev/adder/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/typhonhilldev/adder/compare/v0.6.0...v1.0.0
`

func TestNotes(t *testing.T) {
	tests := []struct {
		name    string
		tag     string
		want    string
		wantErr bool
	}{
		{
			name: "a section with relative links",
			tag:  "v1.1.0",
			want: `### Added

- Releases, as [compatibility](https://github.com/typhonhilldev/adder/blob/v1.1.0/docs/compatibility.md#deprecation) describes, and the [examples](https://github.com/typhonhilldev/adder/blob/v1.1.0/examples/).
- A link that is already absolute: [Go](https://go.dev), and one to a heading: [below](#fixed).

### Fixed

- A fix.

**Full changelog:** https://github.com/typhonhilldev/adder/compare/v1.0.0...v1.1.0
`,
		},
		{
			name: "the section before the link references",
			tag:  "v1.0.0",
			want: "### Added\n\n- The first stable release.\n\n**Full changelog:** https://github.com/typhonhilldev/adder/compare/v0.6.0...v1.0.0\n",
		},
		{name: "no section for the version", tag: "v1.2.0", wantErr: true},
		{name: "an empty section", tag: "v0.9.0", wantErr: true},
		{name: "a version without a link reference", tag: "v0.8.0", wantErr: true},
		{name: "not a version", tag: "latest", wantErr: true},
		{name: "a prefix of another version", tag: "v1.1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := notes(changelog, tt.tag)
			if (err != nil) != tt.wantErr {
				t.Fatalf("notes() error = %v, want an error: %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("notes() =\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}
