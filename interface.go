package grip

import (
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/send"
)

// The base type for all Journaling methods provided by the Grip
// package. The package logger uses systemd logging on Linux, when
// possible, falling back to standard output-native when systemd
// logging is not available.

// Journaler describes the public interface of the the Grip
// interface. Used to enforce consistency between the grip and logging
// packages.
type Journaler interface {
	Name() string
	SetName(string)

	// Methods to access the underlying message sending backend.
	GetSender() send.Sender
	SetSender(send.Sender) error

	// Send allows you to push a composer which stores its own
	// priorty (or uses the sender's default priority).
	Send(interface{})

	// Specify a log level as an argument rather than a method
	// name.
	Log(level.Priority, interface{})
	Logf(level.Priority, string, ...interface{})
	LogWhen(bool, level.Priority, interface{})

	// Methods for sending messages at specific levels. If you
	// send a message at a level that is below the threshold, then it is a no-op.

	// Emergency methods have "panic" and "fatal" variants that
	// call panic or os.Exit(1). It is impossible for "Emergency"
	// to be below threshold, however, if the message isn't
	// loggable (e.g. error is nil, or message is empty,) these
	// methods will not panic/error.
	EmergencyFatal(interface{})
	EmergencyPanic(interface{})

	// For each level, in addition to a basic logger that takes
	// strings and message.Composer objects (and tries to do its
	// best with everythingelse.) Each Level also has "When"
	// variants that only log if the passed condition are true.
	Emergency(interface{})
	Emergencyf(string, ...interface{})
	EmergencyWhen(bool, interface{})

	Alert(interface{})
	Alertf(string, ...interface{})
	AlertWhen(bool, interface{})

	Critical(interface{})
	Criticalf(string, ...interface{})
	CriticalWhen(bool, interface{})

	Error(interface{})
	Errorf(string, ...interface{})
	ErrorWhen(bool, interface{})

	Warning(interface{})
	Warningf(string, ...interface{})
	WarningWhen(bool, interface{})

	Notice(interface{})
	Noticef(string, ...interface{})
	NoticeWhen(bool, interface{})

	Info(interface{})
	Infof(string, ...interface{})
	InfoWhen(bool, interface{})

	Debug(interface{})
	Debugf(string, ...interface{})
	DebugWhen(bool, interface{})
}
