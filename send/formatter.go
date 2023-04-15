package send

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/tychoish/grip/message"
)

const (
	defaultFormatTmpl = "[p=%s]: %s"
	callSiteTmpl      = "[p=%s] [%s:%d]: %s"
)

// MakeJSONFormatter returns a MessageFormatter, that returns messages
// as the string form of a JSON document built using the Raw method of
// the Composer. Returns an error if there was a problem marshalling JSON.
func MakeJSONFormatter() MessageFormatter {
	return func(m message.Composer) (string, error) {
		out, err := json.Marshal(m.Raw())
		if err != nil {
			return "", err
		}

		return string(out), nil
	}
}

// MakeDefaultFormatter returns a MessageFormatter that will produce a
// message in the following format:
//
//	[p=<level>]: <message>
//
// It can never error.
func MakeDefaultFormatter() MessageFormatter {
	return func(m message.Composer) (string, error) {
		return fmt.Sprintf(defaultFormatTmpl, m.Priority(), m.String()), nil
	}
}

// MakePlainFormatter returns a MessageFormatter that simply returns the
// string format of the log message.
func MakePlainFormatter() MessageFormatter {
	return func(m message.Composer) (string, error) {
		return m.String(), nil
	}
}

// MakeCallSiteFormatter returns a MessageFormater that formats
// messages with the following format:
//
//	[p=<levvel>] [<fileName>:<lineNumber>]: <message>
//
// It can never error.
func MakeCallSiteFormatter(depth int) MessageFormatter {
	depth++
	return func(m message.Composer) (string, error) {
		file, line := callerInfo(depth)
		return fmt.Sprintf(callSiteTmpl, m.Priority(), file, line, m), nil
	}
}

func callerInfo(depth int) (string, int) {
	// increase depth to account for callerInfo itself.
	depth++

	// get caller info.
	_, file, line, _ := runtime.Caller(depth)

	// get the directory and filename
	dir, fileName := filepath.Split(file)
	file = filepath.Join(filepath.Base(dir), fileName)

	return file, line
}
