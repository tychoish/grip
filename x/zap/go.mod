module github.com/tychoish/grip/x/zap

go 1.24

toolchain go1.24.0

require (
	github.com/tychoish/fun v0.12.0
	github.com/tychoish/grip v0.4.0
	go.uber.org/zap v1.24.0
)

require (
	github.com/pkg/errors v0.9.1 // indirect
	github.com/stretchr/testify v1.8.1 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
)

replace github.com/tychoish/grip => ../../
