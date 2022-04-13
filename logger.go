// Basic Journaler
//
// The Journaler implement provides helpers for sending messages at the
// following levels:
//
//    Emergency + (fatal/panic)
//    Alert
//    Critical
//    Error
//    Warning
//    Notice
//    Info
//    Debug
//
// These helpers also include Log* helpers to parameterize the level, as
// well as the Send method for default logging (or when the level is
// on the massage itself.)
package grip

import (
	"os"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

// The base type for all Journaling methods provided by the Grip
// package. The package logger uses systemd logging on Linux, when
// possible, falling back to standard output-native when systemd
// logging is not available.

// Logger describes the public interface of the the Grip
// interface. Used to enforce consistency between the grip and logging
// packages.
type Logger interface {
	// Method to access the underlying message sending backend.
	GetSender() send.Sender

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

// journalerImpl provides the core implementation of the Logging interface. The
// interface is mirrored in the "grip" package's public inte1rface, to
// provide a single, global logging interface that requires minimal
// configuration.
type loggerImpl struct {
	impl send.Sender
}

// NewLogger builds a new logging interface from a sender implementation.
func NewLogger(s send.Sender) Logger { return &loggerImpl{impl: s} }

func (g *loggerImpl) GetSender() send.Sender { return g.impl }
func (g *loggerImpl) Send(m interface{}) {
	g.impl.Send(message.ConvertToComposer(g.impl.Level().Default, m))
}

// implementation

func (g *loggerImpl) send(m message.Composer) { g.impl.Send(m) }

// For sending logging messages, in most cases, use the
// Journaler.sender.Send() method, but we have a couple of methods to
// use for the Panic/Fatal helpers.
func (g *loggerImpl) sendPanic(m message.Composer) {
	// the Send method in the Sender interface will perform this
	// check but to add fatal methods we need to do this here.
	if g.impl.Level().ShouldLog(m) {
		g.impl.Send(m)
		panic(m.String())
	}
}

func (g *loggerImpl) sendFatal(m message.Composer) {
	// the Send method in the Sender interface will perform this
	// check but to add fatal methods we need to do this here.
	if g.impl.Level().ShouldLog(m) {
		g.impl.Send(m)
		os.Exit(1)
	}
}

///////////////////////////////////
//
// method implementation

func (g *loggerImpl) Log(l level.Priority, msg interface{}) {
	g.send(message.ConvertToComposer(l, msg))
}
func (g *loggerImpl) Logf(l level.Priority, msg string, a ...interface{}) {
	g.send(message.NewFormattedMessage(l, msg, a...))
}
func (g *loggerImpl) LogWhen(conditional bool, l level.Priority, m interface{}) {
	g.send(message.When(conditional, message.ConvertToComposer(l, m)))
}

func (g *loggerImpl) EmergencyPanic(msg interface{}) {
	g.sendPanic(message.ConvertToComposer(level.Emergency, msg))
}
func (g *loggerImpl) EmergencyFatal(msg interface{}) {
	g.sendFatal(message.ConvertToComposer(level.Emergency, msg))
}

func (g *loggerImpl) Emergency(msg interface{}) {
	g.send(message.ConvertToComposer(level.Emergency, msg))
}
func (g *loggerImpl) Emergencyf(msg string, a ...interface{}) {
	g.send(message.NewFormattedMessage(level.Emergency, msg, a...))
}
func (g *loggerImpl) EmergencyWhen(conditional bool, m interface{}) {
	g.send(message.When(conditional, message.ConvertToComposer(level.Emergency, m)))
}

func (g *loggerImpl) Alert(msg interface{}) {
	g.send(message.ConvertToComposer(level.Alert, msg))
}
func (g *loggerImpl) Alertf(msg string, a ...interface{}) {
	g.send(message.NewFormattedMessage(level.Alert, msg, a...))
}
func (g *loggerImpl) AlertWhen(conditional bool, m interface{}) {
	g.send(message.When(conditional, message.ConvertToComposer(level.Alert, m)))
}

func (g *loggerImpl) Critical(msg interface{}) {
	g.send(message.ConvertToComposer(level.Critical, msg))
}
func (g *loggerImpl) Criticalf(msg string, a ...interface{}) {
	g.send(message.NewFormattedMessage(level.Critical, msg, a...))
}
func (g *loggerImpl) CriticalWhen(conditional bool, m interface{}) {
	g.send(message.When(conditional, message.ConvertToComposer(level.Critical, m)))
}

func (g *loggerImpl) Error(msg interface{}) {
	g.send(message.ConvertToComposer(level.Error, msg))
}
func (g *loggerImpl) Errorf(msg string, a ...interface{}) {
	g.send(message.NewFormattedMessage(level.Error, msg, a...))
}
func (g *loggerImpl) ErrorWhen(conditional bool, m interface{}) {
	g.send(message.When(conditional, message.ConvertToComposer(level.Error, m)))
}

func (g *loggerImpl) Warning(msg interface{}) {
	g.send(message.ConvertToComposer(level.Warning, msg))
}
func (g *loggerImpl) Warningf(msg string, a ...interface{}) {
	g.send(message.NewFormattedMessage(level.Warning, msg, a...))
}
func (g *loggerImpl) WarningWhen(conditional bool, m interface{}) {
	g.send(message.When(conditional, message.ConvertToComposer(level.Warning, m)))
}

func (g *loggerImpl) Notice(msg interface{}) {
	g.send(message.ConvertToComposer(level.Notice, msg))
}
func (g *loggerImpl) Noticef(msg string, a ...interface{}) {
	g.send(message.NewFormattedMessage(level.Notice, msg, a...))
}
func (g *loggerImpl) NoticeWhen(conditional bool, m interface{}) {
	g.send(message.When(conditional, message.ConvertToComposer(level.Notice, m)))
}

func (g *loggerImpl) Info(msg interface{}) {
	g.send(message.ConvertToComposer(level.Info, msg))
}
func (g *loggerImpl) Infof(msg string, a ...interface{}) {
	g.send(message.NewFormattedMessage(level.Info, msg, a...))
}
func (g *loggerImpl) InfoWhen(conditional bool, m interface{}) {
	g.send(message.When(conditional, message.ConvertToComposer(level.Info, m)))
}

func (g *loggerImpl) Debug(msg interface{}) {
	g.send(message.ConvertToComposer(level.Debug, msg))
}
func (g *loggerImpl) Debugf(msg string, a ...interface{}) {
	g.send(message.NewFormattedMessage(level.Debug, msg, a...))
}
func (g *loggerImpl) DebugWhen(conditional bool, m interface{}) {
	g.send(message.When(conditional, message.ConvertToComposer(level.Debug, m)))
}
