// Command gen compiles signatures/ into the embedded RE2 artifacts
// (antibot/antibot.re2.txt and antibot/antibot-challenge.re2.txt). It has no
// //go:embed in its import graph, so it builds and runs on a clean checkout — where
// those generated files do not yet exist — to produce them. Run from the repo root:
//
//	go run ./cmd/gen
//
// Flags mirror the old `antibot compile`: --dir, --out, --challenge-out.
package main

import (
	"fmt"
	"os"

	"github.com/albinstman/antibot-print/gen"
)

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	dir := "signatures"
	out := "antibot/antibot.re2.txt"
	chOut := "antibot/antibot-challenge.re2.txt"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			i++
			if i < len(args) {
				dir = args[i]
			}
		case "--out":
			i++
			if i < len(args) {
				out = args[i]
			}
		case "--challenge-out":
			i++
			if i < len(args) {
				chOut = args[i]
			}
		default:
			fmt.Fprintf(os.Stderr, "gen: unknown argument %q\n", args[i])
			return 2
		}
	}
	pattern, err := gen.CompileSignatures(dir, out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "gen: wrote %s (%d bytes)\n", out, len(pattern))
	chPattern, err := gen.CompileChallengeSignatures(dir, chOut)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "gen: wrote %s (%d bytes)\n", chOut, len(chPattern))
	return 0
}
