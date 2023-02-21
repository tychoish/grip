//go:build !linux
// +build !linux

package system

// MakeDefaultSystem returns a native log sender on all platforms
// other than linux.
func MakeDefault() (Sender, error) { return MakeNative(), nil }
