package metrics

import (
	"os"
	"os/exec"
	"testing"

	"github.com/shirou/gopsutil/process"
	"github.com/stretchr/testify/assert"
)

func TestChildren(t *testing.T) {
	assert := assert.New(t)
	myPid := int32(os.Getpid())
	p, err := process.NewProcess(myPid)
	assert.NotNil(p)
	if err != nil {
		t.Error(err)
	}
	cmd := exec.Command("sleep", "1")
	if err := cmd.Start(); err != nil {
		t.Error(err)
	}

	c, err := p.Children()
	assert.NotNil(c)
	if err != nil {
		t.Error(err)
	}
	if len(c) != 1 {
		t.Error("elements should be equal")
	}
	for _, process := range c {
		assert.NotEqual(myPid, process.Pid)
	}
}
