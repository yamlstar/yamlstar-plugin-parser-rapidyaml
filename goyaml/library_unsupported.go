//go:build !darwin && !freebsd && !linux

package goyaml

import (
	"fmt"
	"runtime"
)

func loadLibrary() (*library, error) {
	return nil, fmt.Errorf("rapidyaml shared libraries are unsupported on %s",
		runtime.GOOS)
}

func openLibrary(_ string) (*library, error) { return loadLibrary() }
