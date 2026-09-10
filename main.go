package main

/*
#include <stdint.h>
#include <stdlib.h>
*/
import "C"

import (
	_ "embed"
	"encoding/json"
	"unsafe"
)

//go:embed plugin.edn
var manifest string

func writeOutput(text string, output **C.uint8_t, length *C.size_t) C.int32_t {
	return writeBytes([]byte(text), output, length)
}

func writeBytes(data []byte, output **C.uint8_t, length *C.size_t) C.int32_t {
	if output == nil || length == nil {
		return 2
	}
	var pointer unsafe.Pointer
	if len(data) > 0 {
		pointer = C.CBytes(data)
	}
	*output = (*C.uint8_t)(pointer)
	*length = C.size_t(len(data))
	return 0
}

func quoteEDN(value string) string {
	text, _ := json.Marshal(value)
	return string(text)
}

func errorEDN(kind string, err error) string {
	return "{:error {:type " + quoteEDN(kind) +
		" :message " + quoteEDN(err.Error()) + " :data {}}}"
}

//export yamlstar_plugin_v1_abi
func yamlstar_plugin_v1_abi() C.uint64_t {
	return 1
}

//export yamlstar_plugin_v1_manifest
func yamlstar_plugin_v1_manifest(
	output **C.uint8_t,
	length *C.size_t,
) C.int32_t {
	return writeOutput(manifest, output, length)
}

//export yamlstar_plugin_v1_free
func yamlstar_plugin_v1_free(output *C.uint8_t) {
	C.free(unsafe.Pointer(output))
}

func main() {}
