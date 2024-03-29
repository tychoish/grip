// +go:build linux

package system

import (
	"github.com/coreos/go-systemd/journal"
	"github.com/tychoish/grip/send"
)

// MakeDefaultSystem constructs a default logger that pushes to
// systemd on platforms where that's available and standard output
// otherwise.
func MakeDefault() send.Sender {
	if journal.Enabled() {
		return MakeSystemdSender()
	}

	return send.MakeStdError()
}
