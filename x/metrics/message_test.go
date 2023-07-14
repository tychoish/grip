package metrics

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/tychoish/grip/message"
)

func TestDataCollecterComposerConstructors(t *testing.T) {
	t.Parallel()

	const testMsg = "hello"
	// map objects to output (prefix)

	t.Run("Single", func(t *testing.T) {
		t.Parallel()
		for _, test := range []struct {
			Name       string
			Msg        message.Composer
			Expected   string
			ShouldSkip bool
		}{
			{
				Name: "ProcessInfoCurrentProc",
				Msg:  MakeProcessInfo(int32(os.Getpid()), testMsg),
			},
			{
				Name:     "NewSystemInfo",
				Msg:      MakeSystemInfo(testMsg),
				Expected: testMsg,
			},

			{
				Name:     "MakeSystemInfo",
				Msg:      MakeSystemInfo(testMsg),
				Expected: testMsg,
			},
			{
				Name:       "CollectProcInfoPidOne",
				Msg:        CollectProcessInfo(int32(1)),
				ShouldSkip: runtime.GOOS == "windows",
			},
			{
				Name: "CollectProcInfoSelf",
				Msg:  CollectProcessInfoSelf(),
			},
			{
				Name: "CollectSystemInfo",
				Msg:  CollectSystemInfo(),
			},
			{
				Name: "CollectBasicGoStats",
				Msg:  CollectBasicGoStats(),
			},
			{
				Name: "CollectGoStatsDeltas",
				Msg:  CollectGoStatsDeltas(),
			},
			{
				Name: "CollectGoStatsRates",
				Msg:  CollectGoStatsRates(),
			},
			{
				Name: "CollectGoStatsTotals",
				Msg:  CollectGoStatsTotals(),
			},
			{
				Name:     "MakeGoStatsDelta",
				Msg:      MakeGoStatsDeltas(testMsg),
				Expected: testMsg,
			},
			{
				Name:     "MakeGoStatsRates",
				Msg:      MakeGoStatsRates(testMsg),
				Expected: testMsg,
			},
			{
				Name:     "MakeGoStatsTotals",
				Msg:      MakeGoStatsTotals(testMsg),
				Expected: testMsg,
			},
			{
				Name:     "NewGoStatsDeltas",
				Msg:      MakeGoStatsDeltas(testMsg),
				Expected: testMsg,
			},
			{
				Name:     "NewGoStatsRates",
				Msg:      MakeGoStatsRates(testMsg),
				Expected: testMsg,
			},
			{
				Name:     "NewGoStatsTotals",
				Msg:      MakeGoStatsTotals(testMsg),
				Expected: testMsg,
			},
		} {
			if test.ShouldSkip {
				continue
			}
			t.Run(test.Name, func(t *testing.T) {
				t.Parallel()
				if test.Msg == nil {
					t.Fatal("message should not be nil")
				}
				if test.Msg.Raw() == nil {
					t.Fatal("message must not be nil in raw form")
				}

				if _, ok := test.Msg.(message.Composer); !ok {
					t.Errorf("%T should implement message.Composer, but doesn't", test.Msg)
				}
				if !test.Msg.Loggable() {
					t.Error("should be true")
				}
				if !strings.HasPrefix(test.Msg.String(), test.Expected) {
					t.Errorf("%T: %s", test.Msg, test.Msg)
				}
			})
		}
	})

	t.Run("Multi", func(t *testing.T) {
		t.Parallel()

		for _, test := range []struct {
			Name       string
			Group      []message.Composer
			ShouldSkip bool
		}{
			{
				Name:  "SelfWithChildren",
				Group: CollectProcessInfoSelfWithChildren(),
			},
			{
				Name:       "PidOneWithChildren",
				Group:      CollectProcessInfoWithChildren(int32(1)),
				ShouldSkip: runtime.GOOS == "windows",
			},
			{
				Name:  "AllProcesses",
				Group: CollectAllProcesses(),
			},
		} {
			if test.ShouldSkip {
				continue
			}
			t.Run(test.Name, func(t *testing.T) {
				t.Parallel()

				if len(test.Group) == 0 {
					t.Fatalf("test group is empty and should not")
				}
				for _, msg := range test.Group {
					if msg == nil {
						t.Fatal("msg not be nill")
					}
					if _, ok := msg.(message.Composer); !ok {
						t.Errorf("%T should implement message.Composer, but doesn't", msg)
					}
					if msg.String() == "" {
						t.Fatal("message must not be empty")
					}
					if !msg.Loggable() {
						t.Error("should be true")
					}
				}
			})

		}
	})
}

func TestProcessTreeDoesNotHaveDuplicates(t *testing.T) {
	t.Parallel()

	procs := CollectProcessInfoWithChildren(1)
	seen := make(map[int32]struct{})

	for _, p := range procs {
		pinfo, ok := p.(*ProcessInfo)
		if !ok {
			t.Error("should be true")
		}
		seen[pinfo.Payload.Pid] = struct{}{}
	}

	if len(procs) != len(seen) {
		t.Error("elements should be equal")
	}
}
