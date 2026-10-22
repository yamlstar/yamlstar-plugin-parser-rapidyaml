package main

import (
	"bytes"
	"strings"
	"testing"
)

func checkNativeParity(t testing.TB, input, options string) {
	t.Helper()
	want, wantErr := parseEDN(input, options)
	got, gotErr := parseNativeEDN(input, options)
	if (wantErr == nil) != (gotErr == nil) {
		t.Fatalf("native/oracle status differs for %q: native %v, oracle %v",
			input, gotErr, wantErr)
	}
	if wantErr == nil && !bytes.Equal(got, want) {
		t.Fatalf("native/oracle events differ for %q:\nnative %s\noracle %s",
			input, got, want)
	}
	if wantErr != nil {
		_, wantInternal := wantErr.(internalError)
		_, gotInternal := gotErr.(internalError)
		if wantInternal != gotInternal {
			t.Fatalf("native/oracle error classification differs: %v, %v",
				gotErr, wantErr)
		}
	}
}

func TestNativeParity(t *testing.T) {
	for _, input := range []string{
		"", "# comment\n", "---\n...\n---\na\n",
		"a: &a [!!str 42, 'λ🙂', *a]\n",
		"%YAML 1.2\n%TAG !e! tag:example.com,2026:\n--- !e!items [1]\n...\n",
		"!<tag:example.com,a%20b> value", "!local value", "!!str 42",
		"!<tag:example.com,%ff> value", "!<tag:example.com,%zz> value",
		"literal: |\n  line\nfolded: >-\n  more\n  words\n",
		`"a\0b\n\t\b\f\r\"\\<&>\L\P\u0085"`,
		"a: b\rc: d\r", "a: b\r\nc: d\r\n", "a: \"\"\n",
		"[unterminated", "\xff", "\xed\xa0\x80", "\xf4\x90\x80\x80",
		"\xc0\x80", "\xe2\x82", "\xf0\x9f\x99", "[untermi\rated",
		`"` + strings.Repeat("\\L", 10000) + `"`,
		`"` + strings.Repeat("x", 240*1024) + `"`,
		"[" + strings.Repeat("[],", 10000) + "]",
	} {
		checkNativeParity(t, input, "{}")
	}
	for _, options := range []string{"", " \t{ , }\n", "{:unknown true}", "\xff"} {
		checkNativeParity(t, "answer: 42", options)
	}
}
