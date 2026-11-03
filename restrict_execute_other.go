//go:build !(linux && amd64)

package nono

// RestrictExecute limits which files the current process and all its
// children may execute to paths. It is available on linux/amd64; on other
// platforms it returns an error wrapping ErrUnsupportedPlatform until their
// bundled libraries are rebuilt with nono_sandbox_restrict_execute.
func RestrictExecute(paths []string) error {
	return staticError(ErrCodeUnsupportedPlatform, "execute restriction is not built for this platform yet")
}
