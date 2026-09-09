//go:build !windows

package diagnostics

import "runtime"

func osVersion() string {
	return runtime.GOOS
}
