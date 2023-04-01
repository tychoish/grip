/*
Package level defines a Priority type and some conversion methods for a 7-tiered
logging level schema, which mirror syslog and system's logging levels.

Levels range from Emergency (0) to Debug (7), and the special type
Priority and associated constants provide access to these values.
*/
package level

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Priority is an integer that tracks log levels. Use with one of the
// defined constants.
type Priority uint8

// Constants defined for easy access to
const (
	Emergency Priority = 250
	Alert     Priority = 225
	Critical  Priority = 200
	Error     Priority = 175
	Warning   Priority = 150
	Notice    Priority = 125
	Info      Priority = 100
	Debug     Priority = 50
	Trace     Priority = 25
	Invalid   Priority = 0
)

// String implements the Stringer interface and makes it possible to
// print human-readable string identifier for a log level.
func (p Priority) String() string {
	switch p {
	case Emergency:
		return "emergency"
	case Alert:
		return "alert"
	case Critical:
		return "critical"
	case Error:
		return "error"
	case Warning:
		return "warning"
	case Notice:
		return "notice"
	case Info:
		return "info"
	case Debug:
		return "debug"
	case Trace:
		return "trace"
	case Invalid:
		return "invalid"
	default:
		return fmt.Sprintf("level.Priority<%d>", uint8(p))
	}
}

// FromString takes a string, (case insensitive, leading and trailing space removed, )
func FromString(l string) Priority {
	l = strings.TrimSpace(strings.ToLower(l))
	switch l {
	case "emergency":
		return Emergency
	case "alert":
		return Alert
	case "critical":
		return Critical
	case "error":
		return Error
	case "warning":
		return Warning
	case "notice":
		return Notice
	case "info":
		return Info
	case "debug":
		return Debug
	case "trace":
		return Trace
	case "invalid":
		return Invalid
	default:
		if strings.HasPrefix(l, "level.priority<") && strings.HasSuffix(l, ">") {
			l = l[15 : len(l)-1]
		}

		out, err := strconv.Atoi(l)
		if err != nil {
			return Invalid
		}
		if out > math.MaxUint8 {
			return Invalid
		}

		return Priority(out)
	}
}
