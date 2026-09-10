package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	events "github.com/yamlstar/yaml-events-binary-protocol"
)

func testEvents(data []byte) (string, error) {
	var out strings.Builder
	escape := strings.NewReplacer("\\", "\\\\", "\x00", "\\0", "\a", "\\a", "\b", "\\b", "\t", "\\t", "\n", "\\n", "\v", "\\v", "\f", "\\f", "\r", "\\r", "\x1b", "\\e", "\u0085", "\\N", "\u00a0", "\\_", "\u2028", "\\L", "\u2029", "\\P")
	err := events.Decode(data, func(e events.Event) error {
		line := ""
		switch e.Kind {
		case events.StreamStart:
			line = "+STR"
		case events.StreamEnd:
			line = "-STR"
		case events.DocumentStart:
			line = "+DOC"
			if e.Flags&events.Explicit != 0 {
				line += " ---"
			}
		case events.DocumentEnd:
			line = "-DOC"
			if e.Flags&events.Explicit != 0 {
				line += " ..."
			}
		case events.MappingStart:
			line = "+MAP"
			if e.Flags&events.Flow != 0 {
				line += " {}"
			}
		case events.MappingEnd:
			line = "-MAP"
		case events.SequenceStart:
			line = "+SEQ"
			if e.Flags&events.Flow != 0 {
				line += " []"
			}
		case events.SequenceEnd:
			line = "-SEQ"
		case events.Scalar:
			line = "=VAL"
		case events.Alias:
			line = "=ALI *" + e.Strings[events.Name]
		}
		if e.Flags&events.Present(events.Anchor) != 0 {
			line += " &" + e.Strings[events.Anchor]
		}
		if e.Flags&events.Present(events.Tag) != 0 {
			line += " <" + e.Strings[events.Tag] + ">"
		}
		if e.Kind == events.Scalar {
			style := map[string]string{"": ":", "single": "'", "double": "\"", "literal": "|", "folded": ">"}[e.Strings[events.Style]]
			line += " " + style + escape.Replace(e.Strings[events.Value])
		}
		out.WriteString(line + "\n")
		return nil
	})
	return out.String(), err
}

func TestConformance(t *testing.T) {
	root := os.Getenv("YAMLSTAR_TEST_SUITE_DIR")
	if root == "" {
		t.Skip("set YAMLSTAR_TEST_SUITE_DIR to the pinned data checkout")
	}
	known := map[string]string{}
	baseline, err := os.ReadFile("test/expected-events.txt")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(baseline), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		known[parts[0]] = parts[1]
	}
	count := 0
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "in.yaml" {
			return nil
		}
		dir := filepath.Dir(path)
		id, _ := filepath.Rel(root, dir)
		count++
		input, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, invalid := os.Stat(filepath.Join(dir, "error"))
		data, parseErr := parseBinary(string(input), "{}")
		outcome := ""
		if invalid == nil {
			if parseErr == nil {
				outcome = "accepted-invalid"
			}
		} else if parseErr != nil {
			outcome = "rejected-valid"
		} else {
			expected, err := os.ReadFile(filepath.Join(dir, "test.event"))
			if err != nil {
				return err
			}
			actual, err := testEvents(data)
			if err != nil {
				return err
			}
			if string(expected) != actual {
				outcome = "event-mismatch"
				if os.Getenv("YAMLSTAR_SHOW_EVENT_DIFF") != "" {
					t.Logf("%s expected:\n%sactual:\n%s", id, expected, actual)
				}
			}
		}
		if outcome != known[id] {
			t.Errorf("%s %s (expected %q; parse error: %v)", id, outcome, known[id], parseErr)
		}
		delete(known, id)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("no fixtures")
	}
	if len(known) != 0 {
		t.Fatal("stale cases", known)
	}
	t.Log(fmt.Sprintf("checked %d YAML test-suite cases", count))
}
