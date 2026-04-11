module github.com/tychoish/grip/x/benchmarks

go 1.24

require (
	github.com/rs/zerolog v1.27.0
	github.com/tychoish/grip v0.4.8
	github.com/tychoish/grip/x/zap v0.0.0
	github.com/tychoish/grip/x/zerolog v0.0.0
	go.uber.org/zap v1.27.1
)

require (
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/tychoish/fun v0.14.9 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
)

replace (
	github.com/tychoish/grip => ../../
	github.com/tychoish/grip/x/zap => ../zap
	github.com/tychoish/grip/x/zerolog => ../zerolog
)
