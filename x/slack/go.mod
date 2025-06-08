module github.com/tychoish/grip/x/slack

go 1.24

toolchain go1.24.0

require (
	github.com/bluele/slack v0.0.0-20180528010058-b4b4d354a079
	github.com/tychoish/grip v0.4.0
)

require github.com/tychoish/fun v0.12.0 // indirect

replace github.com/tychoish/grip => ../../
