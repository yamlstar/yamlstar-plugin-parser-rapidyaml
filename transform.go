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

//export yamlstar_plugin_v2_transform
func yamlstar_plugin_v2_transform(
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
			_ = writeOutput(fmt.Sprint(value), output, outputLength)
			status = 2
		}
	}()
	maxLength := uint64((1<<31)/4 - 1)
	if (input == nil && inputLength != 0) ||
		(options == nil && optionsLength != 0) ||
		uint64(inputLength) > maxLength || uint64(optionsLength) > maxLength {
		_ = writeOutput("invalid input buffer", output, outputLength)
		return 2
	}
	// Length-aware copies avoid C.int truncation and never retain caller memory.
	inputText := string(unsafe.Slice((*byte)(unsafe.Pointer(input)), int(inputLength)))
	optionsText := string(unsafe.Slice((*byte)(unsafe.Pointer(options)), int(optionsLength)))
	result, err := parseEDN(inputText, optionsText)
	if err != nil {
		if _, internal := err.(internalError); internal {
			_ = writeOutput(err.Error(), output, outputLength)
			return 2
		}
		_ = writeOutput(err.Error(), output, outputLength)
		return 1
	}
	return writeBytes(result, output, outputLength)
}
