package adder

import (
	"crypto/sha256"
	"encoding/base64"
	"os"
	"regexp"
	"testing"
)

// TestGitLabJobIntegrity checks that the integrity hash the documentation gives for the GitLab CI job matches the
// job, so that a change to the job cannot ship with a hash GitLab would reject. There is one job and one place
// that documents it, so it is not a table.
func TestGitLabJobIntegrity(t *testing.T) {
	job, err := os.ReadFile("ci/gitlab/adder.yml")
	if err != nil {
		t.Fatal(err)
	}
	docs, err := os.ReadFile("docs/integrations.md")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)/ci/gitlab/adder\.yml\n\s+integrity: (sha256-\S+)$`).FindSubmatch(docs)
	if m == nil {
		t.Fatal("docs/integrations.md has no integrity hash after the job's include")
	}
	sum := sha256.Sum256(job)
	if want := "sha256-" + base64.StdEncoding.EncodeToString(sum[:]); string(m[1]) != want {
		t.Errorf("docs/integrations.md gives the integrity %s, but ci/gitlab/adder.yml is %s", m[1], want)
	}
}
