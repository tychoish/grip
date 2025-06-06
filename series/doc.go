// Package series provides tools for collecting and aggregating
// timeseries events as part of the logging infrastructure.
//
// The series "system" includes a few basic types and concepts: an
// "event" which is a single data point, a Metric which is a single
// series of datapoints, and a Collector which is responsible for
// tracking and publishing metrics.
//
// In general, as a developer, to use grip/series for your metrics:
// you configure a series.Collector, and embed it in your grip sending
// pipeline, and then embed metric events in your message.
//
// The x/metrics package contains message types that use
// github.com/shirou/gopsutil to collect and generate structred
// logging messages with metrics information. These tools also
// integrate with the `tychoish/birch` bson library and it's
// `birch/x/ftdc` timeseries compression format. Additionally, `bson`
// formatted output renders for metric events are also provided here.
//
// WARNING: This implementation is alpha quality at the moment. Pull
// requests welcome.
package series

import (
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/message"
)

// TODO:
//  - testing
//  - documentation
//  - simple system metric collector (without gopsutil?)

func example() { //nolint:unused
	grip.Info(WithMetrics(message.Fields{"op": "test"},
		Gauge("new_op").Label("key", "value").Inc(),
		Histogram("new_op").Label("key", "value").Inc(),
	))
	// extractMetrics(fn.Futurize(func() message.Fields { return message.Fields{} }), metricMessageWithComposer)
}
