module github.com/tychoish/grip/x/twitter

go 1.18

replace github.com/tychoish/grip => ./../..

require (
	github.com/dghubble/go-twitter v0.0.0-20220626024101-68c0170dc641
	github.com/dghubble/oauth1 v0.7.1
	github.com/tychoish/grip v0.0.0-00010101000000-000000000000
)

require (
	github.com/cenkalti/backoff/v4 v4.1.3 // indirect
	github.com/dghubble/sling v1.4.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/tychoish/emt v0.0.0-20220306153139-74b97c67f209 // indirect
)
