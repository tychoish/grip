module github.com/tychoish/grip/x/splunk

go 1.24

toolchain go1.24.0

require (
	github.com/fuyufjh/splunk-hec-go v0.4.0
	github.com/tychoish/grip v0.3.9-0.20250425134421-fd099d1c46f4
)

require (
	github.com/google/uuid v1.0.0 // indirect
	github.com/stretchr/testify v1.8.1 // indirect
	github.com/tychoish/fun v0.12.0 // indirect
)

replace github.com/tychoish/grip => ../../
