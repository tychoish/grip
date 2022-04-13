package metrics

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

func TestDataCollecterComposerConstructors(t *testing.T) {
	const testMsg = "hello"
	// map objects to output (prefix)

	t.Run("Single", func(t *testing.T) {
		for _, test := range []struct {
			Name       string
			Msg        message.Composer
			Expected   string
			ShouldSkip bool
		}{
			{
				Name: "ProcessInfoCurrentProc",
				Msg:  NewProcessInfo(level.Error, int32(os.Getpid()), testMsg),
			},
			{
				Name:     "NewSystemInfo",
				Msg:      NewSystemInfo(level.Error, testMsg),
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
				Msg:      NewGoStatsDeltas(level.Error, testMsg),
				Expected: testMsg,
			},
			{
				Name:     "NewGoStatsRates",
				Msg:      NewGoStatsRates(level.Error, testMsg),
				Expected: testMsg,
			},
			{
				Name:     "NewGoStatsTotals",
				Msg:      NewGoStatsTotals(level.Error, testMsg),
				Expected: testMsg,
			},
		} {
			if test.ShouldSkip {
				continue
			}
			t.Run(test.Name, func(t *testing.T) {
				assert.NotNil(t, test.Msg)
				assert.NotNil(t, test.Msg.Raw())
				assert.Implements(t, (*message.Composer)(nil), test.Msg)
				assert.True(t, test.Msg.Loggable())
				assert.True(t, strings.HasPrefix(test.Msg.String(), test.Expected), "%T: %s", test.Msg, test.Msg)
			})
		}
	})

	t.Run("Multi", func(t *testing.T) {
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
				require.True(t, len(test.Group) >= 1)
				for _, msg := range test.Group {
					assert.NotNil(t, msg)
					assert.Implements(t, (*message.Composer)(nil), msg)
					assert.NotEqual(t, "", msg.String())
					assert.True(t, msg.Loggable())
				}
			})

		}
	})
}

func TestProcessTreeDoesNotHaveDuplicates(t *testing.T) {
	assert := assert.New(t) // nolint

	procs := CollectProcessInfoWithChildren(1)
	seen := make(map[int32]struct{})

	for _, p := range procs {
		pinfo, ok := p.(*ProcessInfo)
		assert.True(ok)
		seen[pinfo.Pid] = struct{}{}
	}

	assert.Equal(len(seen), len(procs))
}
