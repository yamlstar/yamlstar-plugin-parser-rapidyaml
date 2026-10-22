package main

import (
	"reflect"
	"strings"
	"sync"
	"testing"

	events "github.com/yamlstar/yaml-events-binary-protocol"
)

func decoded(t *testing.T, input string) []events.Event {
	t.Helper()
	data, err := parseBinary(input, "{}")
	if err != nil {
		t.Fatal(err)
	}
	var out []events.Event
	if err = events.Decode(data, func(e events.Event) error { out = append(out, e); return nil }); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestEvents(t *testing.T) {
	out := decoded(t, "%YAML 1.2\n%TAG !e! tag:example.com,2026:\n---\na: &x !e!thing [\"\\L\", 'b']\nc: *x\n...\n")
	var names []string
	for _, e := range out {
		names = append(names, events.Names[e.Kind])
	}
	want := []string{"stream_start", "document_start", "mapping_start", "scalar", "sequence_start", "scalar", "scalar", "sequence_end", "scalar", "alias", "mapping_end", "document_end", "stream_end"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("%v", names)
	}
	if out[1].Strings[events.VersionDirective] != "1.2" || out[1].Flags&events.Explicit == 0 {
		t.Fatalf("document: %+v", out[1])
	}
	if out[4].Strings[events.Tag] != "tag:example.com,2026:thing" || out[4].Strings[events.Anchor] != "x" || out[4].Flags&events.Flow == 0 {
		t.Fatalf("sequence: %+v", out[4])
	}
	if out[5].Strings[events.Value] != "\u2028" || out[5].Strings[events.Style] != "double" {
		t.Fatalf("scalar: %+v", out[5])
	}
	if out[9].Strings[events.Name] != "x" || out[11].Flags&events.Explicit == 0 {
		t.Fatal(out)
	}
}

func TestScalarStylesAndTags(t *testing.T) {
	for _, tc := range []struct{ input, value, style, tag string }{
		{"plain", "plain", "", ""}, {"'a'", "a", "single", ""},
		{"\"a\"", "a", "double", ""}, {"|\n  a\n", "a\n", "literal", ""},
		{">\n  a\n", "a\n", "folded", ""}, {"!!str 42", "42", "", "tag:yaml.org,2002:str"},
		{"!local x", "x", "", "!local"}, {"!<tag:example.com,a%20b> x", "x", "", "tag:example.com,a b"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			out := decoded(t, tc.input)
			e := out[2]
			if e.Strings[events.Value] != tc.value || e.Strings[events.Style] != tc.style || e.Strings[events.Tag] != tc.tag {
				t.Fatalf("%+v", e)
			}
		})
	}
}

func TestEmptyAndMultipleDocuments(t *testing.T) {
	for _, input := range []string{"", "# comment\n", "---\n...\n---\na\n", "--- !!str x\n--- !local y\n"} {
		decoded(t, input)
	}
}

func TestErrorsAndConcurrency(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				if _, err := parseBinary("[unterminated", "{}"); err == nil {
					t.Error("accepted invalid YAML")
				}
				if _, err := parseBinary("a: b", "{:unknown true}"); err == nil {
					t.Error("accepted options")
				}
				if _, err := parseBinary("[a, b]", "{}"); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
}

func TestLargeAndArenaStrings(t *testing.T) {
	for _, value := range []string{strings.Repeat("x", 240*1024), strings.Repeat("\\L", 10000)} {
		out := decoded(t, "\""+value+"\"")
		if out[2].Strings[events.Value] != strings.ReplaceAll(value, "\\L", "\u2028") {
			t.Fatal("scalar changed")
		}
	}
	decoded(t, "["+strings.Repeat("[],", 10000)+"]")
}

func TestMalformedIntegerBuffer(t *testing.T) {
	for _, words := range [][]int32{{512}, {512, -1, 1}, {512, 0, 2}, {0}} {
		if _, err := encodeEvents(words, []byte("x"), nil); err == nil {
			t.Fatalf("accepted %v", words)
		}
	}
}

func FuzzParse(f *testing.F) {
	for _, s := range []string{"", "a: b", "[unterminated", "%YAML 1.2\n---\na"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 100000 {
			t.Skip()
		}
		checkNativeParity(t, input, "{}")
		data, err := parseBinary(input, "{}")
		if err == nil {
			if err = events.Decode(data, func(events.Event) error { return nil }); err != nil {
				t.Fatal(err)
			}
		}
	})
}

func BenchmarkParse(b *testing.B) {
	for name, input := range map[string]string{"mapping": strings.Repeat("a: [1, 2, 3]\n", 1000), "long-string": "\"" + strings.Repeat("x", 240*1024) + "\""} {
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(input)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := parseBinary(input, "{}"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestLineBreakNormalization(t *testing.T) {
	want := decoded(t, "a: b\nc: d\n")
	for _, input := range []string{"a: b\rc: d\r", "a: b\r\nc: d\r\n"} {
		if got := decoded(t, input); !reflect.DeepEqual(got, want) {
			t.Fatal(got)
		}
	}
}
