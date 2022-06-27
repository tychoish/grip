module github.com/tychoish/grip/x/xmpp

go 1.18

replace github.com/tychoish/grip => ./../..

require (
	github.com/mattn/go-xmpp v0.0.0-20220513082406-1411b9cc8b9a
	github.com/tychoish/grip v0.0.0-00010101000000-000000000000
)

require github.com/tychoish/emt v0.0.0-20220306153139-74b97c67f209 // indirect
