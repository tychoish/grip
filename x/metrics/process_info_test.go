package metrics

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/shirou/gopsutil/process"
	"github.com/tychoish/fun/testt"
)

func TestChildren(t *testing.T) {
	t.Parallel()
	myPid := int32(os.Getpid())
	p, err := process.NewProcess(myPid)
	if p == nil {
		t.Fatal("pid info should not be nil")
	}
	if err != nil {
		t.Error(err)
	}

	ctx := testt.Context(t)
	cmd := exec.CommandContext(ctx, "sleep", "2")
	if err = cmd.Start(); err != nil {
		t.Error(err)
	}
	time.Sleep(100 * time.Millisecond)
	c, err := p.Children()
	if c == nil {
		t.Error("child information should not be nil")
	}
	if err != nil {
		t.Error(err)
	}
	if len(c) < 1 {
		t.Error("expected at least one child process")
	}
	for _, process := range c {
		if myPid == process.Pid {
			t.Error("pids should not be equal")
		}
	}
}
