package main

/*
#include <stdint.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"unsafe"
)

//export yamlstar_plugin_v1_parse_binary
func yamlstar_plugin_v1_parse_binary(
	input *C.uint8_t, inputLength C.size_t,
	options *C.uint8_t, optionsLength C.size_t,
	output **C.uint8_t, outputLength *C.size_t,
) (status C.int32_t) {
	if output == nil || outputLength == nil {
		return 2
	}
	*output = nil
	*outputLength = 0
	defer func() {
		if value := recover(); value != nil {
			_ = writeOutput(errorEDN("internal", fmt.Errorf("%v", value)), output, outputLength)
			status = 2
		}
	}()
	maxLength := uint64((1<<31)/4 - 1)
	if (input == nil && inputLength != 0) ||
		(options == nil && optionsLength != 0) ||
		uint64(inputLength) > maxLength || uint64(optionsLength) > maxLength {
		_ = writeOutput(errorEDN("abi", fmt.Errorf("invalid input buffer")), output, outputLength)
		return 2
	}
	// Length-aware copies avoid C.int truncation and never retain caller memory.
	inputText := string(unsafe.Slice((*byte)(unsafe.Pointer(input)), int(inputLength)))
	optionsText := string(unsafe.Slice((*byte)(unsafe.Pointer(options)), int(optionsLength)))
	result, err := parseBinary(inputText, optionsText)
	if err != nil {
		if _, internal := err.(internalError); internal {
			_ = writeOutput(errorEDN("internal", err), output, outputLength)
			return 2
		}
		_ = writeOutput(errorEDN("parse", err), output, outputLength)
		return 1
	}
	return writeBytes(result, output, outputLength)
}
