package series

import (
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/tychoish/fun/assert"
	"github.com/tychoish/fun/ft"
	"github.com/tychoish/fun/testt"
	"github.com/tychoish/grip"
)

func TestIntegration(t *testing.T) {
	t.Run("EndToEnd", func(t *testing.T) {
		grip.Infoln("starting", t.Name())
		ctx := testt.Context(t)
		coll, err := NewCollector(
			ctx,
			CollectorConfOutputGraphite(),
			CollectorConfBuffer(9001),
			CollectorConfAppendBackends(
				LoggerBackend(grip.Sender()),
				ft.Must(SocketBackend(
					CollectorBackendSocketConfMessageWorkers(4),
					CollectorBackendSocketConfDialer(net.Dialer{
						Timeout:   2 * time.Second,
						KeepAlive: time.Minute,
					}),
					CollectorBackendSocketConfNetowrkTCP(),
					CollectorBackendSocketConfAddress("localhost:2003"),
					CollectorBackendSocketConfMinDialRetryDelay(100*time.Millisecond),
					CollectorBackendSocketConfIdleConns(6),
					CollectorBackendSocketConfMaxDialRetryDelay(time.Second),
					CollectorBackendSocketConfMessageErrorHandling(CollectorBackendSocketErrorPanic),
					CollectorBackendSocketConfDialErrorHandling(CollectorBackendSocketErrorPanic)),
				),
			))
		assert.NotError(t, err)
		assert.True(t, coll != nil)

		for i := int64(0); i < 1000; i++ {
			coll.Push(Counter("grip_counter").Label("one", "hundred").Label("itermod", fmt.Sprint(i%10)).Add(100))
			coll.Push(Gauge("grip_gauge").Label("one", "random").Label("itermod", fmt.Sprint(i%10)).Add(rand.Int63n((i + 1) * 100)))
		}
		time.Sleep(2500 * time.Millisecond)
		assert.NotError(t, coll.Close())
	})
}
