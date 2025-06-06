module github.com/tychoish/grip/x/twitter

go 1.24

toolchain go1.24.0

require (
	github.com/dghubble/go-twitter v0.0.0-20220626024101-68c0170dc641
	github.com/dghubble/oauth1 v0.7.1
	github.com/tychoish/grip v0.3.9-0.20250425134421-fd099d1c46f4
)

require (
	github.com/cenkalti/backoff/v4 v4.1.3 // indirect
	github.com/dghubble/sling v1.4.0 // indirect
	github.com/google/go-cmp v0.5.7 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/stretchr/testify v1.8.1 // indirect
	github.com/tychoish/fun v0.12.0 // indirect
)

replace github.com/tychoish/grip => ../../
