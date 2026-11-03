//go:build linux && amd64

package nono

/*
#include <stdlib.h>
#include "internal/clib/nono.h"
*/
import "C"

import (
	"runtime"
	"unsafe"
)

// RestrictExecute limits which files the current process and all its
// children may execute to paths.
//
// It stacks a second Landlock layer that handles only the execute right: a
// file outside paths can no longer be executed, even where Apply granted read
// access (read access includes execute). Read and write grants are unaffected.
// Like Apply, it is irreversible. Call it after Apply.
//
// It requires Landlock ABI v3 or later. With no paths, nothing may be
// executed.
func RestrictExecute(paths []string) error {
	for _, path := range paths {
		if err := checkNUL(path); err != nil {
			return err
		}
	}
	array := C.malloc(C.size_t(len(paths)+1) * C.size_t(unsafe.Sizeof(uintptr(0))))
	defer C.free(array)
	entries := unsafe.Slice((**C.char)(array), len(paths)+1)
	for index, path := range paths {
		entries[index] = C.CString(path)
	}
	defer func() {
		for index := range paths {
			C.free(unsafe.Pointer(entries[index]))
		}
	}()
	// LockOSThread ensures nono_last_error() in mapError reads the
	// thread-local error set by nono_sandbox_restrict_execute on this thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	code := C.nono_sandbox_restrict_execute((**C.char)(array), C.size_t(len(paths)))
	if !isOK(C.int(code)) {
		return mapError(C.int(code))
	}
	return nil
}
