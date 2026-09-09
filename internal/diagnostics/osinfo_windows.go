//go:build windows

package diagnostics

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// osVersion returns a human-readable Windows version, e.g.
// "Windows 11 Pro 23H2 (build 22631.4169)".
func osVersion() string {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return "Windows (unknown build)"
	}
	defer k.Close()

	product, _, err := k.GetStringValue("ProductName")
	if err != nil || product == "" {
		product = "Windows"
	}
	display, _, err := k.GetStringValue("DisplayVersion")
	if err != nil {
		display = ""
	}
	build, _, err := k.GetStringValue("CurrentBuildNumber")
	if err != nil {
		build = ""
	}
	ubr, _, err := k.GetIntegerValue("UBR")
	if err != nil {
		ubr = 0
	}

	// The registry keeps "Windows 10 ..." as ProductName on Windows 11 too.
	if buildNum, convErr := strconv.Atoi(build); convErr == nil && buildNum >= 22000 {
		product = strings.Replace(product, "Windows 10", "Windows 11", 1)
	}

	version := product
	if display != "" {
		version += " " + display
	}
	if build != "" {
		if ubr > 0 {
			version += fmt.Sprintf(" (build %s.%d)", build, ubr)
		} else {
			version += " (build " + build + ")"
		}
	}
	return version
}
