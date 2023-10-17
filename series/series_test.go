package series

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/tychoish/fun/assert"
	"github.com/tychoish/grip"
)

func TestIntegration(t *testing.T) {
	t.Run("EndToEnd", func(t *testing.T) {
		t.Run("TwoOutputs", func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
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

			for i := int64(0); i < 100; i++ {
				coll.Push(Counter("grip_counter").
					Label("case", t.Name()).
					Label("origin", "static").
					Label("itermod", fmt.Sprint(i%10)).Add(100))
				coll.Push(Gauge("grip_gauge").
					Label("case", t.Name()).
					Label("origin", "random").
					Label("itermod", fmt.Sprint(i%10)).Set(rand.Int63n((i + 1) * 100)))
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

			for i := int64(0); i < 100; i++ {
				coll.Push(Counter("grip_counter").
					Label("case", t.Name()).
					Label("origin", "static").
					Label("itermod", fmt.Sprint(i%10)).Add(100))
				coll.Push(Gauge("grip_gauge").
					Label("origin", "random").
					Label("case", t.Name()).
					Label("itermod", fmt.Sprint(i%10)).Set(rand.Int63n((i + 1) * 100)))
				time.Sleep(time.Millisecond)
			}
			time.Sleep(time.Second)
			err = coll.Close()
			assert.NotError(t, err)
		})

	})
}
