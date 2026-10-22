package main

/*
#include "include/yamlstar_plugin.h"
*/
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"
)

// parseNativeEDN exercises the C++ ABI alongside the previous Go test oracle.
// The shared library itself is built directly with the C++ compiler.
func parseNativeEDN(input, options string) ([]byte, error) {
	var source, config *C.uint8_t
	if input != "" {
		source = (*C.uint8_t)(unsafe.Pointer(unsafe.StringData(input)))
	}
	if options != "" {
		config = (*C.uint8_t)(unsafe.Pointer(unsafe.StringData(options)))
	}
	var output *C.uint8_t
	var length C.size_t
	status := C.yamlstar_plugin_v2_transform(source, C.size_t(len(input)),
		config, C.size_t(len(options)), &output, &length)
	runtime.KeepAlive(input)
	runtime.KeepAlive(options)
	defer C.yamlstar_plugin_v2_free(output)
	data := append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(output)),
		int(length))...)
	if status == 2 {
		return nil, internalError{fmt.Errorf("%s", data)}
	}
	if status != 0 {
		return nil, fmt.Errorf("%s", data)
	}
	return data, nil
}
