// Package langtarget lets "conjecture check" run an implementation
// straight from a project's own source tree, by language and entry
// point, instead of from a container image the user built and an
// adapter the user wrote.
//
// Before this package, checking a rule against real code meant
// writing three things by hand for every rule: a program that reads
// conjecture_input.json and writes conjecture_output.json, a
// Dockerfile for it, and an image build. Here conjecture writes the
// adapter itself, from a fixed template per language, and runs it in
// that language's stock image with the project mounted read-only -
// so the one thing left for the project to supply is the entry
// function.
//
// The entry function has the same shape in every language: it takes
// the rule's inputs as a map of name to text and returns the outputs
// as a map of name to text. Names are the rule's own variable names
// without the "input."/"output." prefix; values are the exact text
// the rule uses ("30.00", "economy-flex"). Text on both sides is
// deliberate: a decimal handed over as a binary float would already
// have lost the digits a money rule is about.
//
// Honesty note on what this does and does not remove. The adapter is
// no longer something the implementing agent writes, so it is no
// longer a place to make a failing implementation look conformant.
// The entry function still is: it is project code, and it decides
// what the rule's variable names mean in terms of the real program.
// It is ordinary, reviewable, version-controlled code in the
// project's own language - but nothing here can check that it is a
// faithful binding rather than a second implementation of the rule.
package langtarget

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/farlleyferreira/conjecture-ai/internal/sandbox"
)

//go:embed adapters
var adapters embed.FS

// The paths a prepared target appears at inside the container.
const (
	srcMount     = "/src"
	adapterMount = "/adapter"
)

// Target names the implementation to check.
type Target struct {
	// Lang is one of [Languages], or an alias [Canonical] accepts.
	Lang string
	// Entry names the function to call; its form is per language, see
	// [EntryForm].
	Entry string
	// SrcDir is the project directory mounted read-only at /src.
	SrcDir string
	// Image overrides the language's stock image - for a project whose
	// dependencies are not in its own tree and so need an image that
	// already has them installed.
	Image string
}

// Prepared is a target ready to run: what to hand the sandbox for
// every case.
type Prepared struct {
	Image   string
	Command []string
	Mounts  []sandbox.Mount
	Env     []string
	// Cleanup removes the generated adapter directory.
	Cleanup func()
}

type language struct {
	image     string
	entryForm string
	// files maps an embedded adapter file to the name it is written
	// under in the adapter directory.
	files map[string]string
	env   []string
	build []string
	// run is the per-case command; the entry is appended to it unless
	// noEntryArg is set.
	run        []string
	noEntryArg bool
}

var languages = map[string]language{
	"python": {
		image:     "python:3-slim",
		entryForm: "package.module:function",
		files:     map[string]string{"conjecture_adapter.py": "conjecture_adapter.py"},
		env:       []string{"PYTHONDONTWRITEBYTECODE=1"},
		run:       []string{"python3", adapterMount + "/conjecture_adapter.py"},
	},
	"javascript": {
		image:     "node:24-slim",
		entryForm: "path/to/file.js:exportedFunction",
		files:     map[string]string{"conjecture_adapter.mjs": "conjecture_adapter.mjs"},
		run:       []string{"node", adapterMount + "/conjecture_adapter.mjs"},
	},
	"typescript": {
		image:     "node:24-slim",
		entryForm: "path/to/file.ts:exportedFunction",
		files:     map[string]string{"conjecture_adapter.mjs": "conjecture_adapter.mjs"},
		run: []string{
			"node", "--experimental-transform-types", "--disable-warning=ExperimentalWarning",
			adapterMount + "/conjecture_adapter.mjs",
		},
	},
	"ruby": {
		image:     "ruby:3-slim",
		entryForm: "path/to/file.rb:Module::Class.method or path/to/file.rb:method",
		files:     map[string]string{"conjecture_adapter.rb": "conjecture_adapter.rb"},
		run:       []string{"ruby", adapterMount + "/conjecture_adapter.rb"},
	},
	"go": {
		image:     "golang:1",
		entryForm: "./package/dir:Function",
		files:     map[string]string{"main.go.tmpl": "main.go"},
		env: []string{
			"GOCACHE=" + adapterMount + "/.gocache",
			"GOFLAGS=-buildvcs=false",
			"GOPROXY=off",
			"GOSUMDB=off",
			"GOTOOLCHAIN=local",
			"CGO_ENABLED=0",
		},
		build: []string{
			"sh", "-c",
			"cd " + srcMount + " && go build -overlay " + adapterMount + "/overlay.json -o " +
				adapterMount + "/conjecture_adapter ./" + goOverlayPackage,
		},
		run:        []string{adapterMount + "/conjecture_adapter"},
		noEntryArg: true,
	},
	"java": {
		image:     "eclipse-temurin:21-jdk",
		entryForm: "com.example.Class:staticMethod",
		files: map[string]string{
			"ConjectureAdapter.java": "ConjectureAdapter.java",
			"build_java.sh":          "build_java.sh",
		},
		build: []string{"sh", adapterMount + "/build_java.sh"},
		run: []string{
			"sh", "-c",
			`exec java -cp "$(cat ` + adapterMount + `/classpath)" ConjectureAdapter "$0"`,
		},
	},
	"csharp": {
		image:     "mcr.microsoft.com/dotnet/sdk:10.0",
		entryForm: "path/to/Assembly.dll:Namespace.Class:StaticMethod",
		files: map[string]string{
			"ConjectureAdapter.csproj": "ConjectureAdapter.csproj",
			"Program.cs":               "Program.cs",
		},
		env: []string{
			"DOTNET_CLI_HOME=" + adapterMount + "/.dotnet",
			"DOTNET_CLI_TELEMETRY_OPTOUT=1",
			"DOTNET_NOLOGO=1",
			"NUGET_PACKAGES=" + adapterMount + "/.nuget",
		},
		build: []string{
			"dotnet", "build", adapterMount + "/ConjectureAdapter.csproj",
			"-c", "Release", "-o", adapterMount + "/out", "-v", "quiet",
		},
		run: []string{"dotnet", adapterMount + "/out/ConjectureAdapter.dll"},
	},
}

// goOverlayPackage is the directory, relative to the module root, the
// generated Go adapter is overlaid into (see [goFiles]).
const goOverlayPackage = "conjecture_adapter_main"

var aliases = map[string]string{
	"py":     "python",
	"js":     "javascript",
	"node":   "javascript",
	"ts":     "typescript",
	"golang": "go",
	"rb":     "ruby",
	"cs":     "csharp",
	"c#":     "csharp",
	"dotnet": "csharp",
}

// Languages lists every supported language name, sorted.
func Languages() []string {
	names := make([]string, 0, len(languages))
	for name := range languages {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Canonical resolves a language name or alias ("py", "ts", "c#") to
// its name in [Languages].
func Canonical(lang string) (string, bool) {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if canonical, ok := aliases[lang]; ok {
		lang = canonical
	}
	_, ok := languages[lang]
	return lang, ok
}

// EntryForm describes what --entry looks like for lang.
func EntryForm(lang string) string {
	canonical, ok := Canonical(lang)
	if !ok {
		return ""
	}
	return languages[canonical].entryForm
}

// Prepare writes the adapter for t into a fresh directory, runs the
// language's build step once when it has one (inside the sandbox, so
// with no network, exactly like every case that follows), and returns
// what to run for each case.
//
// A failed build is returned as an error carrying the build's own
// output: a project that does not compile is not a conformance
// verdict about any rule.
func Prepare(ctx context.Context, sb *sandbox.Sandbox, t Target) (Prepared, error) {
	name, ok := Canonical(t.Lang)
	if !ok {
		return Prepared{}, fmt.Errorf("langtarget: unsupported language %q (supported: %s)", t.Lang, strings.Join(Languages(), ", "))
	}
	lang := languages[name]
	if strings.TrimSpace(t.Entry) == "" {
		return Prepared{}, fmt.Errorf("langtarget: --entry is required (for %s: %s)", name, lang.entryForm)
	}

	srcDir, err := filepath.Abs(t.SrcDir)
	if err != nil {
		return Prepared{}, fmt.Errorf("langtarget: %w", err)
	}
	if info, err := os.Stat(srcDir); err != nil || !info.IsDir() {
		return Prepared{}, fmt.Errorf("langtarget: source directory %s does not exist", srcDir)
	}

	adapterDir, err := os.MkdirTemp("", "conjecture-adapter")
	if err != nil {
		return Prepared{}, fmt.Errorf("langtarget: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(adapterDir) }

	files, err := adapterFiles(name, lang, t.Entry, srcDir)
	if err != nil {
		cleanup()
		return Prepared{}, err
	}
	for fileName, content := range files {
		if err := os.WriteFile(filepath.Join(adapterDir, fileName), content, 0o644); err != nil {
			cleanup()
			return Prepared{}, fmt.Errorf("langtarget: writing adapter: %w", err)
		}
	}

	image := lang.image
	if t.Image != "" {
		image = t.Image
	}
	prepared := Prepared{
		Image:   image,
		Command: append([]string{}, lang.run...),
		Mounts: []sandbox.Mount{
			{Host: srcDir, Container: srcMount, ReadOnly: true},
			{Host: adapterDir, Container: adapterMount},
		},
		Env:     append([]string{}, lang.env...),
		Cleanup: cleanup,
	}
	if !lang.noEntryArg {
		prepared.Command = append(prepared.Command, t.Entry)
	}
	if name == "python" {
		prepared.Env = append(prepared.Env, "PYTHONPATH="+srcMount)
	}
	if name == "go" {
		if cache := hostGoModCache(); cache != "" {
			prepared.Mounts = append(prepared.Mounts, sandbox.Mount{Host: cache, Container: "/go/pkg/mod", ReadOnly: true})
		}
	}

	if len(lang.build) > 0 {
		builder := &sandbox.Sandbox{DockerBinary: sb.DockerBinary, Mounts: prepared.Mounts, Env: prepared.Env}
		result, err := builder.Run(ctx, sandbox.Options{Image: image, Dir: adapterDir, Command: lang.build})
		if err != nil {
			cleanup()
			return Prepared{}, fmt.Errorf("langtarget: building the %s adapter: %w", name, err)
		}
		if result.TimedOut {
			cleanup()
			return Prepared{}, fmt.Errorf("langtarget: building the %s adapter timed out", name)
		}
		if result.ExitCode != 0 {
			cleanup()
			return Prepared{}, fmt.Errorf("langtarget: building the %s adapter failed (exit %d):\n%s%s",
				name, result.ExitCode, result.Stdout, result.Stderr)
		}
	}
	return prepared, nil
}

// adapterFiles returns the adapter directory's content for one target:
// the language's embedded templates, with line endings normalized to
// LF (a shell script written with CRLF does not run), plus whatever a
// language has to compute from the target itself.
func adapterFiles(name string, lang language, entry, srcDir string) (map[string][]byte, error) {
	out := make(map[string][]byte, len(lang.files)+1)
	for embedded, written := range lang.files {
		content, err := adapters.ReadFile(path.Join("adapters", embedded))
		if err != nil {
			return nil, fmt.Errorf("langtarget: %w", err)
		}
		out[written] = []byte(strings.ReplaceAll(string(content), "\r\n", "\n"))
	}
	if name == "go" {
		if err := goFiles(out, entry, srcDir); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// goFiles fills in the Go adapter. Go has no way to call a function
// chosen at run time, so the adapter's main.go has to import the
// entry's package by name; and it has to be compiled as part of the
// project's own module, or it could not import anything under an
// internal/ directory - which is where most application code lives.
// "go build -overlay" does exactly that without writing into the
// project: it makes main.go appear at <module root>/conjecture_adapter_main
// for this one build only.
func goFiles(files map[string][]byte, entry, srcDir string) error {
	dir, fn, ok := strings.Cut(entry, ":")
	if !ok || dir == "" || fn == "" {
		return fmt.Errorf("langtarget: go entry must be ./package/dir:Function, got %q", entry)
	}
	if !isGoIdentifier(fn) {
		return fmt.Errorf("langtarget: go entry function %q is not an exported Go identifier", fn)
	}
	module, err := goModulePath(filepath.Join(srcDir, "go.mod"))
	if err != nil {
		return err
	}
	rel := path.Clean(filepath.ToSlash(dir))
	if path.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, "../") {
		return fmt.Errorf("langtarget: go entry package %q must be a directory inside the module", dir)
	}
	importPath := module
	if rel != "." {
		importPath = module + "/" + rel
	}

	main := string(files["main.go"])
	main = strings.ReplaceAll(main, "{{IMPORT_PATH}}", importPath)
	main = strings.ReplaceAll(main, "{{FUNC}}", fn)
	files["main.go"] = []byte(main)

	overlay, err := json.Marshal(map[string]map[string]string{
		"Replace": {srcMount + "/" + goOverlayPackage + "/main.go": adapterMount + "/main.go"},
	})
	if err != nil {
		return fmt.Errorf("langtarget: %w", err)
	}
	files["overlay.json"] = overlay
	return nil
}

func isGoIdentifier(s string) bool {
	for i, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
		case i > 0 && (r == '_' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')):
		default:
			return false
		}
	}
	return s != ""
}

// goModulePath reads the module path out of a go.mod file.
func goModulePath(goMod string) (string, error) {
	data, err := os.ReadFile(goMod)
	if err != nil {
		return "", fmt.Errorf("langtarget: a go target needs go.mod at the source root: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "module" {
			return strings.Trim(fields[1], `"`), nil
		}
	}
	return "", errors.New("langtarget: go.mod has no module line")
}

// hostGoModCache finds this machine's Go module cache, so a Go target
// with dependencies can build with no network: GOMODCACHE, then
// GOPATH/pkg/mod, then ~/go/pkg/mod. It returns "" when none exists -
// a dependency-free or vendored module builds without one.
func hostGoModCache() string {
	candidates := []string{os.Getenv("GOMODCACHE")}
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		candidates = append(candidates, filepath.Join(filepath.SplitList(gopath)[0], "pkg", "mod"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, "go", "pkg", "mod"))
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}
	return ""
}
