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

// goBinary locates the go toolchain via PATH, not runtime.GOROOT()
// (deprecated since Go 1.24 - its own value reflects where this test
// binary was compiled, not necessarily a real toolchain install): go
// test itself only ever runs via the go tool, so "go" being
// unresolvable here would mean a broken environment, not something
// worth a fallback for.
func goBinary() string {
	path, err := exec.LookPath("go")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return path
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

// networkProbeScript is a self-contained Python script that attempts
// a real outbound TCP connection to a real external host (8.8.8.8,
// a well-known, highly-available public DNS resolver, port 53), and
// exits 0 if that attempt was refused/unreachable (the sandboxed
// expectation), 1 if it actually connected (network isolation did
// not hold).
const networkProbeScript = `import socket, sys
try:
    socket.create_connection(("8.8.8.8", 53), timeout=5)
    print("CONNECTED")
    sys.exit(1)
except OSError as e:
    print("BLOCKED:", e)
    sys.exit(0)
`

// TestRun_RealNetworkIsolationBlocksOutboundConnection is a real,
// manual integration test against a real Docker/Podman daemon and
// real outbound internet access from the host running it - never run
// in CI (this project's own CI has neither). conformance-cycle's own
// "Sandboxed execution with no network access" requirement - the one
// real security property this package exists to provide - was, before
// this test, only ever checked by [TestBuildDockerArgs_
// NetworkNoneAndMountShape] as a string present in the constructed
// argument list; this is the first time it is checked against an
// actual running container attempting an actual network call,
// confirming the real container runtime (not just this package's own
// argument-building code) actually enforces it.
//
// This runs the same probe script twice: once through a *Sandbox
// (which always applies --network none - there is no way to disable
// it, by design) and, as a control proving the probe itself is a
// valid test of connectivity (ruling out "this host has no internet
// at all" producing a false-blocked result), once via the same
// container runtime directly, with no network restriction. If the
// control itself fails to connect, the test reports that honestly
// rather than treating it as isolation working.
//
// Set CONJECTURE_SANDBOX_MANUAL_TEST=1, with a real "docker" (or a
// "docker"-named shim to a real Podman, as this project's own README/
// WSL setup uses) reachable on PATH, a pullable "python:3-slim"
// image, and real outbound internet access, then run this test alone.
func TestRun_RealNetworkIsolationBlocksOutboundConnection(t *testing.T) {
	if os.Getenv("CONJECTURE_SANDBOX_MANUAL_TEST") == "" {
		t.Skip("manual only - set CONJECTURE_SANDBOX_MANUAL_TEST=1 with a real docker/podman daemon and real outbound internet access from this host")
	}

	controlDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(controlDir, "probe.py"), []byte(networkProbeScript), 0o644); err != nil {
		t.Fatalf("writing control probe script returned unexpected error: %v", err)
	}
	controlOut, controlErr := exec.Command("docker", "run", "--rm",
		"-v", controlDir+":/workspace", "-w", "/workspace", "python:3-slim",
		"python3", "probe.py",
	).CombinedOutput()
	if controlErr == nil {
		t.Fatalf("control probe (no network restriction) reported BLOCKED when it should have CONNECTED - this host/image has no real outbound internet access, so this test cannot say anything about real isolation:\n%s", controlOut)
	}
	if !strings.Contains(string(controlOut), "CONNECTED") {
		t.Fatalf("control probe did not report CONNECTED - cannot trust this test's own methodology:\n%s", controlOut)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "probe.py"), []byte(networkProbeScript), 0o644); err != nil {
		t.Fatalf("writing sandboxed probe script returned unexpected error: %v", err)
	}
	s := &Sandbox{}
	result, err := s.Run(context.Background(), Options{Image: "python:3-slim", Dir: dir, Command: []string{"python3", "probe.py"}})
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("sandboxed probe exit code = %d (stdout: %q, stderr: %q), want 0 (BLOCKED) - the control probe proved this host can reach 8.8.8.8:53 with no restriction, so a sandboxed CONNECTED means --network none did not isolate this real container",
			result.ExitCode, result.Stdout, result.Stderr)
	}
	if !strings.Contains(string(result.Stdout), "BLOCKED") {
		t.Errorf("sandboxed probe stdout = %q, want it to report BLOCKED", result.Stdout)
	}
}

func TestRun_DockerBinaryNotFoundIsError(t *testing.T) {
	s := &Sandbox{DockerBinary: filepath.Join(t.TempDir(), "nonexistent-docker-binary")}
	_, err := s.Run(context.Background(), Options{Image: "x", Dir: t.TempDir(), Command: []string{"echo", "hi"}})
	if err == nil {
		t.Fatal("Run with a nonexistent docker binary = nil error, want one")
	}
}

func TestResolveRuntime(t *testing.T) {
	onPath := func(present ...string) func(string) (string, error) {
		return func(name string) (string, error) {
			for _, p := range present {
				if p == name {
					return "/usr/bin/" + name, nil
				}
			}
			return "", exec.ErrNotFound
		}
	}
	cases := []struct {
		name     string
		override string
		present  []string
		want     string
	}{
		{"docker wins when both are installed", "", []string{"docker", "podman"}, "docker"},
		{"podman is used when docker is absent", "", []string{"podman"}, "podman"},
		{"docker is named when neither is installed", "", nil, "docker"},
		{"the override wins over detection", "nerdctl", []string{"docker", "podman"}, "nerdctl"},
	}
	for _, c := range cases {
		if got := resolveRuntime(c.override, onPath(c.present...)); got != c.want {
			t.Errorf("%s: resolveRuntime = %q, want %q", c.name, got, c.want)
		}
	}
}
