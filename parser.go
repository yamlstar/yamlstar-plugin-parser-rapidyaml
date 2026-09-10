package main

/*
#cgo CXXFLAGS: -std=c++11 -fvisibility=hidden -DC4_EXCEPTIONS_ENABLED -DC4_ERROR_THROWS_EXCEPTION -I${SRCDIR}/.cache/rapidyaml/src -I${SRCDIR}/.cache/rapidyaml/src_extra -I${SRCDIR}/.cache/rapidyaml/ext/c4core.src
#cgo linux LDFLAGS: -lstdc++
#cgo darwin LDFLAGS: -lc++
#include <stdlib.h>
#include "rymlshim.h"
*/
import "C"

import (
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"
	"unsafe"

	events "github.com/yamlstar/yaml-events-binary-protocol"
)

type internalError struct{ error }

func parseBinary(input, options string) ([]byte, error) {
	// No parser-specific options are supported in v1.
	compact := strings.Map(func(r rune) rune {
		if strings.ContainsRune(" \t\r\n,", r) {
			return -1
		}
		return r
	}, options)
	if compact != "" && compact != "{}" {
		return nil, fmt.Errorf("rapidyaml does not support parser options")
	}
	if !utf8.ValidString(input) {
		return nil, fmt.Errorf("input is not UTF-8")
	}
	// YAML normalizes CRLF and bare CR line endings before scalar processing.
	if strings.ContainsRune(input, '\r') {
		input = strings.ReplaceAll(strings.ReplaceAll(input, "\r\n", "\n"), "\r", "\n")
	}
	var ptr unsafe.Pointer
	if len(input) > 0 {
		ptr = C.CBytes([]byte(input))
		defer C.free(ptr)
	}
	result := C.yamlstar_ryml_parse_events((*C.char)(ptr), C.size_t(len(input)))
	defer C.yamlstar_ryml_free_result(&result)
	if result.ok < 0 {
		return nil, internalError{fmt.Errorf("%s", C.GoString(result.error))}
	}
	if result.ok == 0 {
		if result.error != nil {
			return nil, fmt.Errorf("%s", C.GoString(result.error))
		}
		return nil, fmt.Errorf("rapidyaml internal allocation failure")
	}
	words := unsafe.Slice((*int32)(unsafe.Pointer(result.events)), int(result.events_len))
	source := unsafe.Slice((*byte)(unsafe.Pointer(result.source)), int(result.source_len))
	arena := unsafe.Slice((*byte)(unsafe.Pointer(result.arena)), int(result.arena_len))
	data, err := encodeEvents(words, source, arena)
	if err != nil {
		return nil, internalError{err}
	}
	return data, nil
}

func encodeEvents(words []int32, source, arena []byte) ([]byte, error) {
	var builder events.Builder
	var anchor, tag, version, handle string
	var hasAnchor, hasTag bool
	tags := map[string]string{"!!": "tag:yaml.org,2002:", "!": "!"}
	for i := 0; i < len(words); i++ {
		bits := words[i]
		text := ""
		if bits&C.RYML_WSTR != 0 {
			if i+2 >= len(words) {
				return nil, fmt.Errorf("truncated rapidyaml event")
			}
			offset, length := int(words[i+1]), int(words[i+2])
			region := source
			if bits&C.RYML_AREN != 0 {
				region = arena
			}
			if offset < 0 || length < 0 || offset > len(region) || length > len(region)-offset {
				return nil, fmt.Errorf("rapidyaml string offset out of bounds")
			}
			text = string(region[offset : offset+length])
			i += 2
		}
		var e events.Event
		set := func(field int, value string) { e.Flags |= events.Present(field); e.Strings[field] = value }
		all := func(mask int32) bool { return bits&mask == mask }
		switch {
		case all(C.RYML_BSTR):
			e.Kind = events.StreamStart
		case all(C.RYML_ESTR):
			e.Kind = events.StreamEnd
		case all(C.RYML_BDOC):
			e.Kind = events.DocumentStart
			if version != "" {
				set(events.VersionDirective, version)
				version = ""
			}
			if bits&C.RYML_EXPL != 0 {
				e.Flags |= events.HasExplicit | events.Explicit
			}
		case all(C.RYML_EDOC):
			e.Kind = events.DocumentEnd
			if bits&C.RYML_EXPL != 0 {
				e.Flags |= events.HasExplicit | events.Explicit
			}
			tags = map[string]string{"!!": "tag:yaml.org,2002:", "!": "!"}
		case all(C.RYML_BMAP):
			e.Kind = events.MappingStart
		case all(C.RYML_EMAP):
			e.Kind = events.MappingEnd
		case all(C.RYML_BSEQ):
			e.Kind = events.SequenceStart
		case all(C.RYML_ESEQ):
			e.Kind = events.SequenceEnd
		case bits&C.RYML_YAML != 0:
			version = text
			continue
		case bits&C.RYML_TAGH != 0:
			handle = text
			continue
		case bits&C.RYML_TAGP != 0:
			tags[handle] = text
			handle = ""
			continue
		case bits&C.RYML_ANCH != 0:
			anchor = text
			hasAnchor = true
			continue
		case bits&C.RYML_TAG != 0:
			var err error
			tag, err = resolveTag(text, tags)
			if err != nil {
				return nil, err
			}
			hasTag = true
			continue
		case bits&C.RYML_ALIA != 0:
			e.Kind = events.Alias
			set(events.Name, text)
		case bits&C.RYML_SCLR != 0:
			e.Kind = events.Scalar
			set(events.Value, text)
			switch {
			case bits&C.RYML_SQUO != 0:
				set(events.Style, "single")
			case bits&C.RYML_DQUO != 0:
				set(events.Style, "double")
			case bits&C.RYML_LITL != 0:
				set(events.Style, "literal")
			case bits&C.RYML_FOLD != 0:
				set(events.Style, "folded")
			}
		default:
			return nil, fmt.Errorf("unknown rapidyaml event: %x", bits)
		}
		if e.Kind == events.MappingStart || e.Kind == events.SequenceStart {
			e.Flags |= events.HasFlow
			if bits&C.RYML_FLOW != 0 {
				e.Flags |= events.Flow
			}
		}
		if e.Kind == events.Scalar || e.Kind == events.MappingStart || e.Kind == events.SequenceStart {
			if hasAnchor {
				set(events.Anchor, anchor)
				hasAnchor = false
			}
			if hasTag {
				set(events.Tag, tag)
				hasTag = false
			}
		}
		if err := builder.Add(e); err != nil {
			return nil, err
		}
	}
	if hasAnchor || hasTag || version != "" || handle != "" {
		return nil, fmt.Errorf("incomplete rapidyaml metadata")
	}
	return builder.Bytes()
}

func resolveTag(tag string, handles map[string]string) (string, error) {
	if strings.HasPrefix(tag, "!<") && strings.HasSuffix(tag, ">") {
		tag = tag[2 : len(tag)-1]
	} else if strings.HasPrefix(tag, "<") && strings.HasSuffix(tag, ">") {
		tag = tag[1 : len(tag)-1]
	} else if strings.HasPrefix(tag, "!") {
		handle, suffix := "!", tag[1:]
		if end := strings.IndexByte(suffix, '!'); end >= 0 {
			handle, suffix = tag[:end+2], suffix[end+1:]
		}
		prefix, ok := handles[handle]
		if !ok {
			return "", fmt.Errorf("undefined tag handle %q", handle)
		}
		tag = prefix + suffix
	}
	result, err := url.PathUnescape(tag)
	if err != nil {
		return "", fmt.Errorf("invalid tag: %w", err)
	}
	return result, nil
}
