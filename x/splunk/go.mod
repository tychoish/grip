module github.com/tychoish/grip/x/splunk

go 1.18

replace github.com/tychoish/grip => ./../..

require (
	github.com/fuyufjh/splunk-hec-go v0.4.0
	github.com/tychoish/grip v0.0.0-00010101000000-000000000000
)

require (
	github.com/google/uuid v1.0.0 // indirect
	github.com/stretchr/testify v1.7.5 // indirect
	github.com/tychoish/emt v0.0.0-20220306153139-74b97c67f209 // indirect
)
