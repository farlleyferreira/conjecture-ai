// Package sandbox implements conformance-cycle's one sandboxing
// requirement: "All implementation code exercised during the
// conformance cycle SHALL run in a sandbox with no network access."
// It runs a command inside a Docker container started with
// "--network none" - Docker's own network namespace isolation is the
// actual security boundary here, not anything this package implements
// itself. Hand-rolling OS-level isolation (Linux namespaces, seccomp,
// Windows containers) from first principles would be the same mistake
// this project already avoided for YAML and JSON Schema (using a
// conformant external implementation of a real standard instead of
// reimplementing one), at much higher stakes: a sandbox that isn't
// quite right is a vulnerability, not a parsing bug.
//
// This package implements exactly the one sandboxing requirement the
// specification names - network isolation. It does not add filesystem
// restrictions, resource limits, or further process hardening beyond
// what "--network none" plus Docker's own default container isolation
// already provide; the specification gives no requirement or concrete
// number for any of those, so none is invented here. It also does not
// implement the conformance cycle itself (rounds, token budget,
// counterexample minimization, the three rigor levels) - only the one
// primitive that loop would run implementation code through.
package sandbox

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// Options configures one sandboxed run.
type Options struct {
	// Image is the Docker image the command runs inside. This package
	// has no default and does not guess one: the target implementation
	// code's language is not this package's concern (language-agnostic-
	// contracts), so the caller - who already knows what it is testing -
	// supplies the image.
	Image string
	// Dir is mounted into the container, read-write, as its working
	// directory - a test run may need to write an output file (e.g. a
	// JUnit XML report). The caller is responsible for Dir's path being
	// one the local Docker daemon can actually mount (on Windows, that
	// may require a specific path form Docker Desktop accepts); this
	// package passes it through unchanged.
	Dir string
	// Command is the command to run inside the container, e.g.
	// []string{"python3", "test.py"}.
	Command []string
}

// Result is one sandboxed run's captured outcome.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	// TimedOut reports whether the run was killed for exceeding its
	// context's deadline, rather than exiting on its own -
	// conformance-cycle's "a timed-out check SHALL be reported as
	// inconclusive, never as a violation" depends on a caller being
	// able to tell the two apart, which is exactly what this field is
	// for; Run deliberately returns (Result{TimedOut: true}, nil) for
	// this case, not an error, since a timeout is an expected, legal
	// outcome of running under a deadline, not a failure of the
	// sandbox itself.
	TimedOut bool
}

// Sandbox runs commands inside Docker containers with no network
// access. The zero value is ready to use against a real local Docker
// daemon (DockerBinary defaults to whatever [resolveRuntime] finds:
// "docker", or "podman" when only that is installed).
type Sandbox struct {
	// DockerBinary overrides the docker executable to invoke - a test
	// hook (a real caller has Docker on its PATH; this package does not
	// install or locate it).
	DockerBinary string
}

// RuntimeEnvVar names the environment variable that picks the
// container runtime executable explicitly (a name resolved via PATH,
// or a full path), overriding the detection in [resolveRuntime].
const RuntimeEnvVar = "CONJECTURE_CONTAINER_RUNTIME"

// resolveRuntime picks the container runtime executable to invoke
// when the caller did not set DockerBinary: override (the value of
// [RuntimeEnvVar]) when non-empty, otherwise "docker" when it is on
// PATH, otherwise "podman" when that is - Podman accepts the same
// "run"/"kill" arguments this package builds, so a Podman-only
// machine needs no "docker" shim. When neither is found it still
// returns "docker", so the error Run reports names the runtime a
// reader most likely expects to install. lookPath is exec.LookPath,
// passed in so a test can answer for a PATH it controls.
func resolveRuntime(override string, lookPath func(string) (string, error)) string {
	if override != "" {
		return override
	}
	for _, candidate := range []string{"docker", "podman"} {
		if _, err := lookPath(candidate); err == nil {
			return candidate
		}
	}
	return "docker"
}

// Run executes opts.Command inside a fresh, no-network Docker
// container, mounting opts.Dir as its working directory. If ctx
// carries a deadline and the command has not finished by then, Run
// kills the container (sending "docker kill" explicitly, since killing
// only the local "docker run" client process does not by itself stop a
// container the daemon is still running) and returns (Result{TimedOut:
// true}, nil).
func (s *Sandbox) Run(ctx context.Context, opts Options) (Result, error) {
	binary := s.DockerBinary
	if binary == "" {
		binary = resolveRuntime(os.Getenv(RuntimeEnvVar), exec.LookPath)
	}

	name := containerName()
	args := buildDockerArgs(opts, name)

	cmd := exec.CommandContext(ctx, binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		killCmd := exec.Command(binary, "kill", name)
		_ = killCmd.Run() // best-effort cleanup; the container is already being torn down by the kill signal exec.CommandContext sent
		return Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), TimedOut: true}, nil
	}

	if runErr != nil {
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			return Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), ExitCode: exitErr.ExitCode()}, nil
		}
		return Result{}, fmt.Errorf("sandbox: running %s: %w", binary, runErr)
	}

	return Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), ExitCode: 0}, nil
}

// buildDockerArgs builds the "docker run" argument list for opts,
// naming the container containerName so it can be targeted by a later
// "docker kill" on timeout. "--network none" is the one requirement
// this package exists to enforce; "--rm" cleans up the container's
// own filesystem layer once it exits on its own (a path "docker kill"
// does not need to duplicate - killing a "--rm" container still
// triggers its removal).
func buildDockerArgs(opts Options, containerName string) []string {
	args := []string{
		"run", "--rm",
		"--network", "none",
		"--name", containerName,
		"-v", opts.Dir + ":/workspace",
		"-w", "/workspace",
		opts.Image,
	}
	return append(args, opts.Command...)
}

// containerName returns a name unlikely to collide with a concurrent
// run's own container - 8 random bytes is far more entropy than this
// needs, chosen only because crypto/rand makes drawing them this
// simple.
func containerName() string {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	return "conjecture-sandbox-" + hex.EncodeToString(buf[:])
}
