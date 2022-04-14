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
	Sender() send.Sender

	// Send allows you to push a composer which stores its own
	// priorty (or uses the sender's default priority).
	Send(interface{})

	// Build produces a message builder that provides a chainable
	// interface for building logging messages.
	Build() *message.Builder

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

func (g *loggerImpl) Sender() send.Sender     { return g.impl }
func (g *loggerImpl) Send(m interface{})      { g.send(g.impl.Level().Default, message.Convert(m)) }
func (g *loggerImpl) Build() *message.Builder { return message.NewBuilder(g.impl.Send) }

// implementation

func (g *loggerImpl) send(l level.Priority, m message.Composer) {
	if err := m.SetPriority(l); err != nil {
		g.impl.ErrorHandler()(err, m)
		return
	}

	g.impl.Send(m)
}

// For sending logging messages, in most cases, use the
// Journaler.sender.Send() method, but we have a couple of methods to
// use for the Panic/Fatal helpers.
func (g *loggerImpl) sendPanic(l level.Priority, m message.Composer) {
	if err := m.SetPriority(l); err != nil {
		g.impl.ErrorHandler()(err, m)
		return
	}

	// the Send method in the Sender interface will perform this
	// check but to add fatal methods we need to do this here.
	if g.impl.Level().ShouldLog(m) {
		g.impl.Send(m)
		panic(m.String())
	}
}

func (g *loggerImpl) sendFatal(l level.Priority, m message.Composer) {
	if err := m.SetPriority(l); err != nil {
		g.impl.ErrorHandler()(err, m)
		return
	}

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

func (g *loggerImpl) Log(l level.Priority, m any)                 { g.send(l, message.Convert(m)) }
func (g *loggerImpl) Logf(l level.Priority, msg string, a ...any) { g.send(l, composerf(msg, a)) }
func (g *loggerImpl) LogWhen(c bool, l level.Priority, m any)     { g.send(l, makeWhen(c, m)) }
func (g *loggerImpl) EmergencyPanic(m any)                        { g.sendPanic(level.Emergency, message.Convert(m)) }
func (g *loggerImpl) EmergencyFatal(m any)                        { g.sendFatal(level.Emergency, message.Convert(m)) }
func (g *loggerImpl) Emergency(m any)                             { g.send(level.Emergency, message.Convert(m)) }
func (g *loggerImpl) Emergencyf(m string, a ...any)               { g.send(level.Emergency, composerf(m, a)) }
func (g *loggerImpl) EmergencyWhen(c bool, m any)                 { g.send(level.Emergency, makeWhen(c, m)) }
func (g *loggerImpl) Alert(m any)                                 { g.send(level.Alert, message.Convert(m)) }
func (g *loggerImpl) Alertf(m string, a ...any)                   { g.send(level.Alert, composerf(m, a)) }
func (g *loggerImpl) AlertWhen(c bool, m any)                     { g.send(level.Alert, makeWhen(c, m)) }
func (g *loggerImpl) Critical(m any)                              { g.send(level.Critical, message.Convert(m)) }
func (g *loggerImpl) Criticalf(m string, a ...any)                { g.send(level.Critical, composerf(m, a)) }
func (g *loggerImpl) CriticalWhen(c bool, m any)                  { g.send(level.Critical, makeWhen(c, m)) }
func (g *loggerImpl) Error(m any)                                 { g.send(level.Error, message.Convert(m)) }
func (g *loggerImpl) Errorf(m string, a ...any)                   { g.send(level.Error, composerf(m, a)) }
func (g *loggerImpl) ErrorWhen(c bool, m any)                     { g.send(level.Error, makeWhen(c, m)) }
func (g *loggerImpl) Warning(m any)                               { g.send(level.Warning, message.Convert(m)) }
func (g *loggerImpl) Warningf(m string, a ...any)                 { g.send(level.Warning, composerf(m, a)) }
func (g *loggerImpl) WarningWhen(c bool, m any)                   { g.send(level.Warning, makeWhen(c, m)) }
func (g *loggerImpl) Notice(m any)                                { g.send(level.Notice, message.Convert(m)) }
func (g *loggerImpl) Noticef(m string, a ...any)                  { g.send(level.Notice, composerf(m, a)) }
func (g *loggerImpl) NoticeWhen(c bool, m any)                    { g.send(level.Notice, makeWhen(c, m)) }
func (g *loggerImpl) Info(m any)                                  { g.send(level.Info, message.Convert(m)) }
func (g *loggerImpl) Infof(m string, a ...any)                    { g.send(level.Info, composerf(m, a)) }
func (g *loggerImpl) InfoWhen(c bool, m any)                      { g.send(level.Info, makeWhen(c, m)) }
func (g *loggerImpl) Debug(m any)                                 { g.send(level.Debug, message.Convert(m)) }
func (g *loggerImpl) Debugf(m string, a ...any)                   { g.send(level.Debug, composerf(m, a)) }
func (g *loggerImpl) DebugWhen(c bool, m any)                     { g.send(level.Debug, makeWhen(c, m)) }

func makeWhen(cond bool, m any) message.Composer         { return message.When(cond, message.Convert(m)) }
func composerf(tmpl string, args []any) message.Composer { return message.MakeFormat(tmpl, args...) }
