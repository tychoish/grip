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

// Logger provides the public interface of the grip Logger.
type Logger struct {
	impl send.Sender
}

// NewLogger builds a new logging interface from a sender implementation.
func NewLogger(s send.Sender) Logger { return Logger{impl: s} }

func (g Logger) Sender() send.Sender     { return g.impl }
func (g Logger) Build() *message.Builder { return message.NewBuilder(g.impl.Send) }

// implementation

func (g Logger) send(l level.Priority, m message.Composer) {
	if err := m.SetPriority(l); err != nil {
		g.impl.ErrorHandler()(err, m)
		return
	}

	g.impl.Send(m)
}

// For sending logging messages, in most cases, use the
// Journaler.sender.Send() method, but we have a couple of methods to
// use for the Panic/Fatal helpers.
func (g Logger) sendPanic(l level.Priority, m message.Composer) {
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

func (g Logger) sendFatal(l level.Priority, m message.Composer) {
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

func (g Logger) Log(l level.Priority, m any)                 { g.send(l, message.Convert(m)) }
func (g Logger) Logf(l level.Priority, msg string, a ...any) { g.send(l, composerf(msg, a)) }
func (g Logger) LogWhen(c bool, l level.Priority, m any)     { g.send(l, makeWhen(c, m)) }
func (g Logger) EmergencyPanic(m any)                        { g.sendPanic(level.Emergency, message.Convert(m)) }
func (g Logger) EmergencyFatal(m any)                        { g.sendFatal(level.Emergency, message.Convert(m)) }
func (g Logger) Emergency(m any)                             { g.send(level.Emergency, message.Convert(m)) }
func (g Logger) Emergencyf(m string, a ...any)               { g.send(level.Emergency, composerf(m, a)) }
func (g Logger) EmergencyWhen(c bool, m any)                 { g.send(level.Emergency, makeWhen(c, m)) }
func (g Logger) Alert(m any)                                 { g.send(level.Alert, message.Convert(m)) }
func (g Logger) Alertf(m string, a ...any)                   { g.send(level.Alert, composerf(m, a)) }
func (g Logger) AlertWhen(c bool, m any)                     { g.send(level.Alert, makeWhen(c, m)) }
func (g Logger) Critical(m any)                              { g.send(level.Critical, message.Convert(m)) }
func (g Logger) Criticalf(m string, a ...any)                { g.send(level.Critical, composerf(m, a)) }
func (g Logger) CriticalWhen(c bool, m any)                  { g.send(level.Critical, makeWhen(c, m)) }
func (g Logger) Error(m any)                                 { g.send(level.Error, message.Convert(m)) }
func (g Logger) Errorf(m string, a ...any)                   { g.send(level.Error, composerf(m, a)) }
func (g Logger) ErrorWhen(c bool, m any)                     { g.send(level.Error, makeWhen(c, m)) }
func (g Logger) Warning(m any)                               { g.send(level.Warning, message.Convert(m)) }
func (g Logger) Warningf(m string, a ...any)                 { g.send(level.Warning, composerf(m, a)) }
func (g Logger) WarningWhen(c bool, m any)                   { g.send(level.Warning, makeWhen(c, m)) }
func (g Logger) Notice(m any)                                { g.send(level.Notice, message.Convert(m)) }
func (g Logger) Noticef(m string, a ...any)                  { g.send(level.Notice, composerf(m, a)) }
func (g Logger) NoticeWhen(c bool, m any)                    { g.send(level.Notice, makeWhen(c, m)) }
func (g Logger) Info(m any)                                  { g.send(level.Info, message.Convert(m)) }
func (g Logger) Infof(m string, a ...any)                    { g.send(level.Info, composerf(m, a)) }
func (g Logger) InfoWhen(c bool, m any)                      { g.send(level.Info, makeWhen(c, m)) }
func (g Logger) Debug(m any)                                 { g.send(level.Debug, message.Convert(m)) }
func (g Logger) Debugf(m string, a ...any)                   { g.send(level.Debug, composerf(m, a)) }
func (g Logger) DebugWhen(c bool, m any)                     { g.send(level.Debug, makeWhen(c, m)) }

func makeWhen(cond bool, m any) message.Composer         { return message.When(cond, message.Convert(m)) }
func composerf(tmpl string, args []any) message.Composer { return message.MakeFormat(tmpl, args...) }
