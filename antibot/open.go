// --open support: extract the final response body from a raw HTTP chain and open
// it in the user's default browser, so the page antibot fetched can be eyeballed.
package antibot

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// extractFinalBody returns the body of the last HTTP response in a raw chain — the
// page a browser would actually render. fetch concatenates each redirect hop as
// `status line / headers / blank line / body`, so the final body is whatever
// follows the last status line's header separator. If raw carries no HTTP status
// line (a bare body piped on stdin), the whole input is treated as the body.
func extractFinalBody(raw []byte) []byte {
	start := lastStatusLine(raw)
	if start < 0 {
		return raw // bare body, no headers
	}
	rest := raw[start:]
	for _, sep := range [][]byte{[]byte("\r\n\r\n"), []byte("\n\n")} {
		if i := bytes.Index(rest, sep); i >= 0 {
			// fetch appends a trailing CRLF after each hop's body; drop it.
			return bytes.TrimSuffix(rest[i+len(sep):], []byte("\r\n"))
		}
	}
	return rest // headers with no separator/body
}

// lastStatusLine returns the byte offset of the last line beginning with "HTTP/",
// or -1 if there is none.
func lastStatusLine(raw []byte) int {
	prefix := []byte("HTTP/")
	last := -1
	for i := 0; i+len(prefix) <= len(raw); i++ {
		if (i == 0 || raw[i-1] == '\n') && bytes.HasPrefix(raw[i:], prefix) {
			last = i
		}
	}
	return last
}

// openInBrowser writes body to a temp .html file and hands it to the OS opener.
func openInBrowser(body []byte) error {
	f, err := os.CreateTemp("", "antibot-*.html")
	if err != nil {
		return err
	}
	if _, err := f.Write(body); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return openPath(f.Name())
}

// openPath launches the platform's default handler for path without blocking.
func openPath(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	default: // linux, *bsd, …
		cmd = exec.Command("xdg-open", path)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s: %w", cmd.Path, err)
	}
	return nil
}
