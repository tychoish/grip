module github.com/tychoish/grip/x/sumologic

go 1.24

toolchain go1.24.0

replace github.com/nutmegdevelopment/sumologic => github.com/tychoish/sumologic v1.0.0

require (
	github.com/nutmegdevelopment/sumologic v0.0.0-00010101000000-000000000000
	github.com/tychoish/grip v0.3.9-0.20250425134421-fd099d1c46f4
)

require (
	github.com/stretchr/testify v1.8.1 // indirect
	github.com/tychoish/fun v0.12.0 // indirect
)

replace github.com/tychoish/grip => ../../
