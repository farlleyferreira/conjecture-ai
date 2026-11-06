package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

var fakeDockerPath string

// TestMain compiles testdata/fakedocker once, into a temp directory
// shared by every test in this package, so Run's own subprocess
// orchestration (argument building, output capture, exit codes,
// timeout + kill) can be exercised without a real Docker daemon - not
// installed in the environment this was written in.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "fakedocker")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	fakeDockerPath = filepath.Join(dir, "fakedocker")
	if runtime.GOOS == "windows" {
		fakeDockerPath += ".exe"
	}

	buildCmd := exec.Command(goBinary(), "build", "-o", fakeDockerPath, "./testdata/fakedocker")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "building fakedocker: %v\n%s", err, out)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// goBinary locates the Go toolchain by GOROOT rather than relying on
// "go" being on PATH, since this test process's own environment does
// not guarantee that (the same reason this project's own build
// commands elsewhere use a full path to go.exe).
func goBinary() string {
	name := "go"
	if runtime.GOOS == "windows" {
		name = "go.exe"
	}
	return filepath.Join(runtime.GOROOT(), "bin", name)
}

func TestBuildDockerArgs_NetworkNoneAndMountShape(t *testing.T) {
	args := buildDockerArgs(Options{Image: "python:3.12-slim", Dir: "/tmp/work", Command: []string{"python3", "test.py"}}, "my-container")

	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--network none") {
		t.Errorf("buildDockerArgs = %v, want it to contain \"--network none\"", args)
	}
	if !strings.Contains(joined, "-v /tmp/work:/workspace") {
		t.Errorf("buildDockerArgs = %v, want it to mount Dir at /workspace", args)
	}
	if !strings.Contains(joined, "-w /workspace") {
		t.Errorf("buildDockerArgs = %v, want -w /workspace", args)
	}
	if !strings.Contains(joined, "--name my-container") {
		t.Errorf("buildDockerArgs = %v, want --name my-container", args)
	}
	if args[len(args)-2] != "python3" || args[len(args)-1] != "test.py" {
		t.Errorf("buildDockerArgs ends with %v, want the command last", args[len(args)-2:])
	}
}

func TestRun_SuccessfulCommand(t *testing.T) {
	s := &Sandbox{DockerBinary: fakeDockerPath}
	result, err := s.Run(context.Background(), Options{Image: "x", Dir: t.TempDir(), Command: []string{"echo", "hi"}})
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if result.TimedOut {
		t.Error("TimedOut = true, want false")
	}
	if !strings.Contains(string(result.Stdout), "hello from sandbox") {
		t.Errorf("Stdout = %q, want it to contain fakedocker's own output", result.Stdout)
	}
}

func TestRun_NonZeroExitCodeCaptured(t *testing.T) {
	s := &Sandbox{DockerBinary: fakeDockerPath}
	result, err := s.Run(context.Background(), Options{Image: "x", Dir: t.TempDir(), Command: []string{"FAIL"}})
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if result.ExitCode != 7 {
		t.Errorf("ExitCode = %d, want 7", result.ExitCode)
	}
	if !strings.Contains(string(result.Stderr), "boom") {
		t.Errorf("Stderr = %q, want it to contain fakedocker's own error output", result.Stderr)
	}
}

func TestRun_TimeoutKillsContainerAndReportsTimedOut(t *testing.T) {
	markerPath := filepath.Join(t.TempDir(), "kill-marker")
	t.Setenv("FAKEDOCKER_KILL_MARKER", markerPath)

	s := &Sandbox{DockerBinary: fakeDockerPath}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	result, err := s.Run(ctx, Options{Image: "x", Dir: t.TempDir(), Command: []string{"SLEEP_FOREVER"}})
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if !result.TimedOut {
		t.Error("TimedOut = false, want true")
	}

	if _, statErr := os.Stat(markerPath); statErr != nil {
		t.Errorf("Run did not call \"docker kill\" on timeout (no marker file written): %v", statErr)
	}
}

func TestRun_DockerBinaryNotFoundIsError(t *testing.T) {
	s := &Sandbox{DockerBinary: filepath.Join(t.TempDir(), "nonexistent-docker-binary")}
	_, err := s.Run(context.Background(), Options{Image: "x", Dir: t.TempDir(), Command: []string{"echo", "hi"}})
	if err == nil {
		t.Fatal("Run with a nonexistent docker binary = nil error, want one")
	}
}
