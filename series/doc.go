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
// pipeline, and then embed metrics in your events.
//
// The x/metrics package includes basic implementations and
// integrations with third party libraries.
package series
