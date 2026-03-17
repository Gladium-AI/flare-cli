package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestVersionOutput(t *testing.T) {
	buf, err := executeCmd(t, "version")
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	if !strings.Contains(out, "flare-cli") {
		t.Errorf("version output should contain 'flare-cli', got: %s", out)
	}
	if !strings.Contains(out, "Version") {
		t.Errorf("version output should contain 'Version', got: %s", out)
	}
}

func TestVersionJSON(t *testing.T) {
	buf, err := executeCmd(t, "version", "--json")
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}

	for _, key := range []string{"version", "commit", "built", "go", "os/arch"} {
		if _, ok := result[key]; !ok {
			t.Errorf("missing key %q in JSON output", key)
		}
	}
}
