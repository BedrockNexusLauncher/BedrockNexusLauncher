// Package diagnostics builds a sanitized, user-reviewable diagnostics report
// for GitHub bug reports. The report is only returned to the UI; the app
// itself never transmits it anywhere.
package diagnostics

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/apppath"
	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/update"
)

const (
	logTailLines = 80
	logMaxChars  = 4000
)

// Report returns a Markdown diagnostics block: app version, OS, architecture
// and a redacted tail of the startup log.
func Report() string {
	var b strings.Builder
	fmt.Fprintf(&b, "- App version: %s\n", update.GetAppVersion())
	fmt.Fprintf(&b, "- OS: %s\n", osVersion())
	fmt.Fprintf(&b, "- Architecture: %s\n", runtime.GOARCH)

	if tail := logTail(); tail != "" {
		b.WriteString("\n**Startup log (tail):**\n\n```\n")
		b.WriteString(tail)
		b.WriteString("\n```\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func logTail() string {
	data, err := os.ReadFile(apppath.StartupLogPath())
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) > logTailLines {
		lines = lines[len(lines)-logTailLines:]
	}
	tail := strings.TrimSpace(redactPaths(strings.Join(lines, "\n")))
	if len(tail) > logMaxChars {
		tail = tail[len(tail)-logMaxChars:]
		if i := strings.IndexByte(tail, '\n'); i >= 0 {
			tail = tail[i+1:]
		}
	}
	return tail
}

// redactPaths hides the local user name by replacing the home directory with
// %USERPROFILE% everywhere it appears in the log text.
func redactPaths(s string) string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		s = strings.ReplaceAll(s, home, "%USERPROFILE%")
	}
	return s
}
