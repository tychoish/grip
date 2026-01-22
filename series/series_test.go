package series

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/tychoish/fun/assert"
	"github.com/tychoish/fun/assert/check"
	"github.com/tychoish/fun/fn"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/fun/testt"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

func TestIntegration(t *testing.T) {
	t.Run("EndToEnd", func(t *testing.T) {
		t.Run("TwoOutputs", func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			sb, err := SocketBackend(CollectorBackendSocketConfWithRenderer(MakeGraphiteRenderer()),
				CollectorBackendSocketConfMessageWorkers(4),
				CollectorBackendSocketConfDialWorkers(6),
				CollectorBackendSocketConfDialer(net.Dialer{
					Timeout:   2 * time.Second,
					KeepAlive: time.Minute,
				}),
				CollectorBackendSocketConfNetowrkTCP(),
				CollectorBackendSocketConfAddress("localhost:2003"),
				CollectorBackendSocketConfMinDialRetryDelay(100*time.Millisecond),
				CollectorBackendSocketConfIdleConns(6),
				CollectorBackendSocketConfMaxDialRetryDelay(time.Second),
				CollectorBackendSocketConfMessageErrorHandling(CollectorBackendSocketErrorAbort),
				CollectorBackendSocketConfDialErrorHandling(CollectorBackendSocketErrorAbort))
			if err != nil {
				t.Fatal(err)
			}

			coll, err := NewCollector(
				ctx,
				CollectorConfBuffer(9001),
				CollectorConfAppendBackends(
					LoggerBackend(grip.Sender(), MakeJSONRenderer()),
					sb,
				))
			assert.NotError(t, err)
			assert.True(t, coll != nil)

			counter := Counter("grip_counter").Label("type", "incrementing")
			gauge := Gauge("grip_gauge").Label("type", "variable")

			for i := range int64(128) {
				coll.Push(counter.Add(i))
				coll.Push(gauge.Set(rand.Int64N(128)))
				time.Sleep(time.Millisecond)
			}

			time.Sleep(time.Second)
			err = coll.Close()
			assert.NotError(t, err)
		})
		t.Run("LoggingOnly", func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			coll, err := NewCollector(
				ctx,
				CollectorConfBuffer(9001),
				CollectorConfAppendBackends(
					LoggerBackend(grip.Sender(), MakeJSONRenderer()),
				))
			assert.NotError(t, err)
			assert.True(t, coll != nil)

			counter := Counter("grip_counter").Label("case", t.Name()).Label("origin", "static")
			gauge := Gauge("grip_gauge").Label("case", t.Name()).Label("origin", "random")

			for i := range int64(128) {
				coll.Push(counter.Add(i))
				coll.Push(gauge.Set(rand.Int64N(128)))
				time.Sleep(time.Millisecond)
			}

			time.Sleep(time.Second)
			err = coll.Close()
			assert.NotError(t, err)
		})
	})
	t.Run("Backends", func(t *testing.T) {
		t.Run("File", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			dir := t.TempDir()

			fbConf := &CollectorBackendFileConf{
				Directory:      dir,
				FilePrefix:     "metrics",
				Extension:      ".jsonl",
				CounterPadding: 3,
				Megabytes:      1,
				Renderer:       MakeJSONRenderer(),
			}

			fileBackend, err := FileBackend(CollectorBackendFileConfSet(fbConf))
			assert.NotError(t, err)

			var (
				mu       sync.Mutex
				captured []string
			)
			captureFn := fnx.MakeHandler(func(msg string) error {
				mu.Lock()
				defer mu.Unlock()

				captured = append(captured, msg)
				return nil
			}).Lock()

			passBackend := PassthroughBackend(MakeJSONRenderer(), captureFn)

			coll, err := NewCollector(ctx,
				CollectorConfBuffer(1),
				CollectorConfAppendBackends(passBackend, fileBackend),
			)
			assert.NotError(t, err)

			memSender, err := send.NewInMemorySender("integrationFile", level.Info, 256)
			assert.NotError(t, err)
			wrappedSender := Sender(memSender, coll)

			const iterations = 128
			for i := 0; i < iterations; i++ {
				coll.Push(Gauge("integration_file_gauge").Set(int64(rand.Int64N(int64(max(1, i))))))
				wrappedSender.Send(message.MakeString(fmt.Sprintf("hello-log-%d", i)))
				time.Sleep(time.Millisecond)
			}

			time.Sleep(100 * time.Millisecond)
			assert.NotError(t, memSender.Flush(ctx))
			assert.NotError(t, coll.Close())

			// Poll until at least one metrics file appears or the test context is done.
			var files []string

			for idx := 0; idx > -1; idx++ {
				files, err = filepath.Glob(filepath.Join(dir, fmt.Sprint(fbConf.FilePrefix, "*")))

				assert.NotError(t, err)
				if len(files) > 0 {
					break
				}

				select {
				case <-ctx.Done():
					t.Fatalf("timed out waiting for metrics files to be written")
				case <-time.After(100 * time.Millisecond):
				}
			}

			type metricLine struct {
				Metric string `json:"metric"`
				Value  int64  `json:"value"`
			}

			var metricsFromFile []metricLine
			for _, fp := range files {
				b, err := os.ReadFile(fp)
				assert.NotError(t, err)

				dec := json.NewDecoder(bytes.NewReader(b))
				for {
					var ml metricLine
					if err := dec.Decode(&ml); err != nil {
						if errors.Is(err, io.EOF) {
							break
						}
						testt.Log(t, string(b))
						t.Fatalf("failed decoding %q: %v", fp, err)
					}
					metricsFromFile = append(metricsFromFile, ml)
				}
			}

			testt.Log(t, metricsFromFile)
			mu.Lock()
			defer mu.Unlock()
			testt.Log(t, captured)
			check.Equal(t, iterations, len(metricsFromFile))
			check.Equal(t, len(captured), len(metricsFromFile))
		})
		t.Run("Socket", func(t *testing.T) {
			t.Run("Graphite", func(t *testing.T) {
				inst := startVictoriaMetrics(t)
				if inst == nil {
					t.Fatal("startVictoria returned nil instance")
				}

				captureCh := make(chan string, 128)
				captureFn := fnx.MakeHandler(func(s string) error {
					select {
					case captureCh <- s:
					default:
					}
					return nil
				})

				sb, err := GraphiteBackend("127.0.0.1:2003")
				assert.NotError(t, err)

				passBackend := PassthroughBackend(MakeGraphiteRenderer(), captureFn)

				ctx := t.Context()

				coll, err := NewCollector(ctx,
					CollectorConfBuffer(512),
					CollectorConfAppendBackends(passBackend, sb),
				)
				assert.NotError(t, err)

				metricName := "integration_graphite_metric"
				gauge := Gauge(metricName).Label("test", t.Name())
				source := fn.MakeFuture(func() int64 { return rand.Int64N(128) }).Lock()

				for i := int64(0); i < 128; i++ {
					coll.Push(gauge.Add(source()))
					time.Sleep(time.Millisecond)
				}

				time.Sleep(10 * time.Millisecond)
				check.NotError(t, coll.Close())

				queryCtx, qCancel := context.WithTimeout(t.Context(), 15*time.Second)
				defer qCancel()

			RETRY:
				for {
					has, err := victoriaHasMetric(queryCtx, t, metricName)
					check.NotError(t, err)
					if has {
						break
					}

					select {
					case <-queryCtx.Done():
						t.Errorf("timed out waiting for graphite metric %q to be ingested", metricName)
						break RETRY
					case <-time.After(100 * time.Millisecond):
					}
				}

				// close(captureCh)
				testt.Log(t, "capture chan", len(captureCh))
				check.True(t, len(captureCh) >= 16)
				testt.Log(t, "capture chan", len(captureCh))
			})
			t.Run("Statsd", func(t *testing.T) {
				inst := startVictoriaMetrics(t)
				if inst == nil {
					t.Fatal("startVictoria returned nil instance")
				}
				// StatsD requires port 8125 UDP to be available with proper victoria-metrics configuration.
				// Pre-existing victoria-metrics instances typically dont have StatsD support.
				if inst.external {
					t.Skip("StatsD test requires Docker container; pre-existing victoria-metrics instance doesnt support StatsD protocol")
				}


				capCh := make(chan string, 128)
				handler := fnx.MakeHandler(func(s string) error {
					select {
					case capCh <- s:
					default:
					}
					return nil
				})

				sb, err := StatsdBackend(
					"127.0.0.1:8125",
					CollectorBackendSocketConfMinMessageRetryDelay(10*time.Millisecond),
				)
				assert.NotError(t, err)

				passBackend := PassthroughBackend(MakeStatsdRenderer(), handler)

				ctx := t.Context()

				coll, err := NewCollector(ctx,
					CollectorConfBuffer(512),
					CollectorConfAppendBackends(passBackend, sb),
				)
				assert.NotError(t, err)

				metricName := "integration_statsd_metric"
				gauge := Gauge(metricName).Label("test", t.Name())
				for i := int64(0); i < 128; i++ {
					coll.Push(gauge.Add(rand.Int64N(max(1, i*i))))
					time.Sleep(time.Millisecond)
				}
				time.Sleep(10 * time.Millisecond)

				assert.NotError(t, coll.Close())

				queryCtx, qCancel := context.WithTimeout(t.Context(), 5*time.Second)
				defer qCancel()

				for {
					has, err := victoriaHasMetric(queryCtx, t, metricName)
					testt.Log(t, err)
					assert.NotError(t, err)
					if has {
						break
					}

					select {
					case <-queryCtx.Done():
						t.Fatalf("timed out waiting for statsd metric %q to be ingested", metricName)
					case <-time.After(250 * time.Millisecond):
					}
				}

				close(capCh)
				assert.True(t, len(capCh) >= 5)
			})
		})
	})
}
