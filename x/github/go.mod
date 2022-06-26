module github.com/tychoish/grip/x/github

go 1.18

replace github.com/tychoish/grip => ./../..

require (
	github.com/google/go-github v17.0.0+incompatible
	github.com/stretchr/testify v1.7.5
	github.com/tychoish/grip v0.0.0-00010101000000-000000000000
	golang.org/x/oauth2 v0.0.0-20220622183110-fd043fe589d2
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/protobuf v1.4.2 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/tychoish/emt v0.0.0-20220306153139-74b97c67f209 // indirect
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd // indirect
	google.golang.org/appengine v1.6.6 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
