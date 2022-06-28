module github.com/tychoish/grip/x/slack

go 1.18

replace github.com/tychoish/grip => ./../..

require (
	github.com/bluele/slack v0.0.0-20180528010058-b4b4d354a079
	github.com/tychoish/grip v0.0.0-00010101000000-000000000000
)

require github.com/tychoish/emt v0.0.0-20220306153139-74b97c67f209 // indirect
