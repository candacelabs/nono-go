//go:build linux && amd64

package nono_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	nono "github.com/nolabs-ai/nono-go"
)

// TestRestrictExecuteSubprocess applies a read-everything sandbox in a child
// process, restricts execute to the directories holding the `true` binary
// and its loader, then checks that `true` still runs while a byte-identical
// copy of it in a temporary directory does not.
func TestRestrictExecuteSubprocess(t *testing.T) {
	if os.Getenv("NONO_SKIP_APPLY_SANDBOX_TEST") == "1" {
		t.Skip("sandbox apply test disabled for this run")
	}
	if os.Getenv("NONO_RESTRICT_EXECUTE_CHILD") == "1" {
		runRestrictExecuteChild(t)
		return
	}

	t.Parallel()
	if !nono.IsSupported() {
		t.Skip("sandboxing is not supported on this platform")
	}

	allowed, err := exec.LookPath("true")
	if err != nil {
		t.Skipf("no true binary on PATH: %v", err)
	}
	// `true` may be a multi-call binary choosing what to do from its name,
	// so it runs by its own path and the copy keeps the name.
	content, err := os.ReadFile(allowed)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", allowed, err)
	}
	denied := filepath.Join(t.TempDir(), "true")
	if err := os.WriteFile(denied, content, 0o755); err != nil {
		t.Fatalf("WriteFile copy: %v", err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestRestrictExecuteSubprocess$", "-test.v")
	cmd.Env = append(os.Environ(),
		"NONO_RESTRICT_EXECUTE_CHILD=1",
		"NONO_EXEC_ALLOWED="+allowed,
		"NONO_EXEC_DENIED="+denied,
	)
	output, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == exitNoExecuteRestriction && os.Getenv("NONO_REQUIRE_APPLY") != "1" {
		t.Skipf("kernel lacks Landlock execute restriction (ABI v3+): %s", output)
	}
	if err != nil {
		t.Fatalf("child failed: %v\n%s", err, output)
	}
}

// exitNoExecuteRestriction is the child's exit status when the kernel cannot
// restrict execute, so the parent skips instead of failing.
const exitNoExecuteRestriction = 77

func runRestrictExecuteChild(t *testing.T) {
	allowed := os.Getenv("NONO_EXEC_ALLOWED")
	denied := os.Getenv("NONO_EXEC_DENIED")

	caps := nono.New()
	defer caps.Close()
	if err := caps.AllowPath("/", nono.AccessRead); err != nil {
		t.Fatalf("AllowPath /: %v", err)
	}
	// os/exec opens /dev/null read-write for a child's unused stdio.
	if err := caps.AllowFile(os.DevNull, nono.AccessReadWrite); err != nil {
		t.Fatalf("AllowFile %s: %v", os.DevNull, err)
	}
	if err := nono.Apply(caps); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// The binary's own directory and wherever its dynamic loader may live.
	var executable []string
	resolved, err := filepath.EvalSymlinks(allowed)
	if err != nil {
		t.Fatalf("EvalSymlinks %s: %v", allowed, err)
	}
	for _, dir := range []string{filepath.Dir(allowed), filepath.Dir(resolved), "/nix/store", "/lib", "/lib64", "/usr/lib", "/usr/lib64"} {
		if _, err := os.Stat(dir); err == nil {
			executable = append(executable, dir)
		}
	}
	if err := nono.RestrictExecute(executable); err != nil {
		if errors.Is(err, nono.ErrSandboxInit) {
			os.Exit(exitNoExecuteRestriction)
		}
		t.Fatalf("RestrictExecute: %v", err)
	}

	if err := exec.Command(allowed).Run(); err != nil {
		t.Fatalf("running %s inside the execute list: %v", allowed, err)
	}
	err = exec.Command(denied).Run()
	if err == nil {
		t.Fatalf("running %s outside the execute list succeeded", denied)
	}
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("running %s outside the execute list: want permission denied, got %v", denied, err)
	}
}
