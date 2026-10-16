package main

import (
	events "github.com/yamlstar/yaml-events-binary-protocol"
	"strings"
)

// parseEDN adapts the validated event stream to the ABI v2 text contract.
func parseEDN(input, options string) ([]byte, error) {
	packet, err := parseBinary(input, options)
	if err != nil {
		return nil, err
	}
	var out strings.Builder
	out.WriteByte('[')
	fields := [...]string{"value", "style", "anchor", "tag", "name", "version"}
	err = events.Decode(packet, func(e events.Event) error {
		out.WriteString("{:event ")
		out.WriteString(quoteEDN(events.Names[e.Kind]))
		for i, name := range fields {
			if e.Flags&events.Present(i) != 0 {
				out.WriteString(" :" + name + " ")
				out.WriteString(quoteEDN(e.Strings[i]))
			}
		}
		for _, flag := range []struct {
			name           string
			present, value uint32
		}{
			{"flow", events.HasFlow, events.Flow},
			{"explicit", events.HasExplicit, events.Explicit},
		} {
			if e.Flags&flag.present != 0 {
				out.WriteString(" :" + flag.name + " ")
				if e.Flags&flag.value != 0 {
					out.WriteString("true")
				} else {
					out.WriteString("false")
				}
			}
		}
		out.WriteString("} ")
		return nil
	})
	if err != nil {
		return nil, internalError{err}
	}
	out.WriteByte(']')
	return []byte(out.String()), nil
}
