module github.com/tychoish/grip/x/github

go 1.24

require (
	github.com/google/go-github v17.0.0+incompatible
	github.com/tychoish/grip v0.4.1
	golang.org/x/oauth2 v0.27.0
)

require (
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/tychoish/fun v0.14.0 // indirect
)

replace github.com/tychoish/grip => ../../
