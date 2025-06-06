module github.com/tychoish/grip/x/zerolog

go 1.24

toolchain go1.24.0

require (
	github.com/rs/zerolog v1.27.0
	github.com/tychoish/fun v0.12.0
	github.com/tychoish/grip v0.3.9-0.20250425134421-fd099d1c46f4
)

require (
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	golang.org/x/sys v0.31.0 // indirect
)

replace github.com/tychoish/grip => ../../
