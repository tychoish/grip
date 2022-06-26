package metrics

import (
	"os"
	"os/exec"
	"testing"

	"github.com/shirou/gopsutil/process"
)

func TestChildren(t *testing.T) {
	myPid := int32(os.Getpid())
	p, err := process.NewProcess(myPid)
	if p == nil {
		t.Fatal("pid info should not be nil")
	}
	if err != nil {
		t.Error(err)
	}
	cmd := exec.Command("sleep", "1")
	if err := cmd.Start(); err != nil {
		t.Error(err)
	}

	c, err := p.Children()
	if c == nil {
		t.Fatal("child information should not be nil")
	}
	if err != nil {
		t.Error(err)
	}
	if len(c) != 1 {
		t.Error("elements should be equal")
	}
	for _, process := range c {
		if myPid == process.Pid {
			t.Error("pids should not be equal")
		}
	}
}
