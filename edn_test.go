package main

import (
	"strings"
	"testing"
)

func TestEDNTransform(t *testing.T) {
	data, err := parseEDN("%YAML 1.2\n---\na: &x [\"a\\0b\\n\\t\\\"\\\\\", 'λ']\nb: *x\n...\n", "{}")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`{:event "stream_start"}`, `:version "1.2"`, `:explicit true`,
		`:flow false`, `:flow true`, `:anchor "x"`, `:style "double"`,
		`:value "a\u0000b\n\t\"\\"`, `:value "λ"`, `:name "x"`,
	} {
		if !strings.Contains(string(data), want) {
			t.Errorf("missing %s in %s", want, data)
		}
	}
	if _, err := parseEDN("[unterminated", "{}"); err == nil {
		t.Fatal("accepted malformed YAML")
	}
	if _, err := parseEDN("ok", "{:unknown true}"); err == nil {
		t.Fatal("accepted unknown options")
	}
}
