module github.com/tychoish/grip/x/system

go 1.24

toolchain go1.24.0

require (
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf
	github.com/tychoish/grip v0.4.0
)

require github.com/tychoish/fun v0.12.0

// replace github.com/tychoish/grip => ../../
