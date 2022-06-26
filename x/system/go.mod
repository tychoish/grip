module github.com/tychoish/grip/x/system

go 1.18

replace github.com/tychoish/grip => ./../..

require (
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf
	github.com/tychoish/grip v0.0.0-00010101000000-000000000000
)

require github.com/tychoish/emt v0.0.0-20220306153139-74b97c67f209 // indirect
