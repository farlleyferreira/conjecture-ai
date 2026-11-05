package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/user"
	"time"

	"github.com/farlleyferreira/conjecture-ai/internal/cjtr"
	"github.com/farlleyferreira/conjecture-ai/internal/version"
	"github.com/farlleyferreira/conjecture-ai/internal/yamlstrict"
)

const signUsage = "usage: conjecture sign [--user <name>] [--confirmed] <file.cjtr>"

// runSign signs a .cjtr file in place (see cjtr.Sign): it refuses when
// the file has an unresolved "???" marker, and otherwise replaces any
// existing "signed" block with a new one recording the given (or
// current OS) user, this binary's own version as "tool", today's date,
// and whether --confirmed was given.
func runSign(args []string, stdout, stderr io.Writer) int {
	var userName, path string
	var confirmed bool

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--user":
			i++
			if i >= len(args) {
				_, _ = fmt.Fprintln(stderr, signUsage)
				return 1
			}
			userName = args[i]
		case "--confirmed":
			confirmed = true
		default:
			if path != "" {
				_, _ = fmt.Fprintln(stderr, signUsage)
				return 1
			}
			path = args[i]
		}
	}
	if path == "" {
		_, _ = fmt.Fprintln(stderr, signUsage)
		return 1
	}

	if userName == "" {
		u, err := user.Current()
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "conjecture: --user not given and the OS user could not be determined")
			return 1
		}
		userName = u.Username
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	root, err := yamlstrict.Load(bytes.NewReader(raw))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	tool := "conjecture-cli " + version.String()
	date := time.Now().UTC().Format("2006-01-02")

	signed, err := cjtr.Sign(raw, root, userName, tool, date, confirmed)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	if err := os.WriteFile(path, signed, 0o644); err != nil {
		_, _ = fmt.Fprintf(stderr, "conjecture: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "signed %s (user %s, confirmed %v)\n", path, userName, confirmed)
	return 0
}
