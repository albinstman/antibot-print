// Command antibot names the antibot/WAF/CAPTCHA vendor(s) protecting a site. It is a
// thin entrypoint: all logic lives in package antibot. version is stamped at release
// time with -ldflags "-X main.version=...".
package main

import (
	"os"

	"github.com/albinstman/antibot-print/antibot"
)

var version = "dev"

func main() {
	os.Exit(antibot.Run(version, os.Args[1:]))
}
