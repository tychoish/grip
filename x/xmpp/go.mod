module github.com/tychoish/grip/x/xmpp

go 1.24

toolchain go1.24.0

require (
	github.com/mattn/go-xmpp v0.0.0-20220513082406-1411b9cc8b9a
	github.com/tychoish/grip v0.3.9-0.20250425134421-fd099d1c46f4
)

require github.com/tychoish/fun v0.12.0 // indirect

replace github.com/tychoish/grip => ../../
