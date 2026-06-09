package antibot

// The compiled regex artifacts are embedded into the CLI binary. They are generated
// from signatures/ by cmd/gen (package gen) and are NOT committed — a clean checkout
// must run `go run ./cmd/gen` before building this package. Go's stdlib regexp IS RE2
// (linear-time, no backreferences/lookaround), so nothing else is bundled at runtime.

import _ "embed"

//go:embed antibot.re2.txt
var embeddedRegex string

//go:embed antibot-challenge.re2.txt
var embeddedChallengeRegex string
