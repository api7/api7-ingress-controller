package version

import (
	"bytes"
	"fmt"
	"runtime"
)

var (
	// The following fields are populated at build time using -ldflags -X.
	_buildVersion     = "unknown"
	_buildGitRevision = "unknown"
	_buildOS          = "unknown"

	_buildGoVersion = runtime.Version()
	_runningOS      = runtime.GOOS + "/" + runtime.GOARCH
)

// Short produces a single-line version info with format:
// <version>-<git revision>-<go version>
func Short() string {
	return fmt.Sprintf("%s-%s-%s", _buildVersion, _buildGitRevision, _buildGoVersion)
}

// Long produces a verbose version info with format:
// Version: xxx
// Git SHA: xxx
// GO Version: xxx
// Running OS/Arch: xxx/xxx
// Building OS/Arch: xxx/xxx
func Long() string {
	buf := bytes.NewBuffer(nil)
	fmt.Fprintln(buf, "Version:", _buildVersion)
	fmt.Fprintln(buf, "Git SHA:", _buildGitRevision)
	fmt.Fprintln(buf, "Go Version:", _buildGoVersion)
	fmt.Fprintln(buf, "Building OS/Arch:", _buildOS)
	fmt.Fprintln(buf, "Running OS/Arch:", _runningOS)
	return buf.String()
}
