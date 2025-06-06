// Global Logging Infrastructure
//
// By default (following from the standard library logger and other
// popular logging pacakges,) grip provides a "default" package level
// logging instance. This logging instance is accessible via a number
// of package functions which mirror the interface of the Logger type
// itself.
//
// During init() this logging instance wraps and uses the underlying
// writer of the standard library's logger. You can use
// SetGlobalLogger to configure your own logging (and sender!)
// infrastructure for default operations, though this function is not
// thread safe.
//
// In many cases, it might make sense to attach a Logger instance to a
// context, as Logging is a global concern and your application is
// likely already using contexts. The WithLogger and Context method
// make it possible to attach and access your logger. The Context
// method will *always* return a Logging instance, either the instance
// attached to a context or the global instance if one is not
// configured.
//
// # Basic Logger
//
// The Logger type provides helpers for sending messages at the
// following levels:
//
//	Emergency + (fatal/panic)
//	Alert
//	Critical
//	Error
//	Warning
//	Notice
//	Info
//	Debug
//	Trace
//
// These helpers also include Log* helpers to parameterize the level, as
// well as the Send method for default logging (or when the level is
// on the massage itself.)
package grip

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

var std Logger

func init() { setupDefault() }

func setupDefault() {
	sender := send.MakeStdOutput()
	if !strings.Contains(os.Args[0], "go-build") {
		sender.SetName(filepath.Base(os.Args[0]))
	} else {
		sender.SetName("grip")
	}

	sender.SetPriority(level.Info)

	std = NewLogger(sender)
}

// minimallist wrapper to make the atomic not panic because of the interface
type sender struct{ send.Sender }

type converter struct{ message.Converter }

// Logger provides the public interface of the grip Logger.
//
// Package level functions mirror all methods on the Logger type to
// access a "global" Logger instance in the grip package.
type Logger struct {
	impl *adt.Atomic[sender]
	conv *adt.Atomic[converter]
}

// NewLogger builds a new logging interface from a sender implementation.
func NewLogger(s send.Sender) Logger {
	return MakeLogger(s, message.DefaultConverter())
}

// MakeLogger constructs a new sender with the specified converter function.
func MakeLogger(s send.Sender, c message.Converter) Logger {
	return Logger{
		impl: adt.NewAtomic(sender{s}),
		conv: adt.NewAtomic(converter{c}),
	}

}

func composerf(tmpl string, args []any) message.Composer { return message.MakeFormat(tmpl, args...) }
func composerln(args []any) message.Composer             { return message.MakeLines(args...) }

// Clone creates a new Logger with the same message sender and
// converter; however they are fully independent loggers.
func (g Logger) Clone() Logger                               { return MakeLogger(g.Sender(), g.conv.Get()) }
func (g Logger) Build() *message.Builder                     { return message.NewBuilder(g.Sender().Send, g.conv.Get()) }
func (g Logger) Sender() send.Sender                         { return g.impl.Get().Sender }
func (g Logger) Convert(m any) message.Composer              { return g.conv.Get().Convert(m) }
func (g Logger) SetSender(s send.Sender)                     { g.impl.Set(sender{s}) }
func (g Logger) SetConverter(m message.Converter)            { g.conv.Set(converter{m}) }
func (g Logger) Log(l level.Priority, m any)                 { g.send(l, m) }
func (g Logger) Logf(l level.Priority, msg string, a ...any) { g.send(l, composerf(msg, a)) }
func (g Logger) Logln(l level.Priority, a ...any)            { g.send(l, composerln(a)) }
func (g Logger) LogWhen(c bool, l level.Priority, m any)     { g.send(l, g.makeWhen(c, m)) }
func (g Logger) EmergencyPanic(m any)                        { g.sendPanic(level.Emergency, m) }
func (g Logger) EmergencyFatal(m any)                        { g.sendFatal(level.Emergency, m) }
func (g Logger) Emergency(m any)                             { g.send(level.Emergency, m) }
func (g Logger) Emergencyf(m string, a ...any)               { g.send(level.Emergency, composerf(m, a)) }
func (g Logger) Emergencyln(a ...any)                        { g.send(level.Emergency, composerln(a)) }
func (g Logger) EmergencyWhen(c bool, m any)                 { g.send(level.Emergency, g.makeWhen(c, m)) }
func (g Logger) Alert(m any)                                 { g.send(level.Alert, m) }
func (g Logger) Alertf(m string, a ...any)                   { g.send(level.Alert, composerf(m, a)) }
func (g Logger) Alertln(a ...any)                            { g.send(level.Alert, composerln(a)) }
func (g Logger) AlertWhen(c bool, m any)                     { g.send(level.Alert, g.makeWhen(c, m)) }
func (g Logger) Critical(m any)                              { g.send(level.Critical, m) }
func (g Logger) Criticalf(m string, a ...any)                { g.send(level.Critical, composerf(m, a)) }
func (g Logger) Criticalln(a ...any)                         { g.send(level.Critical, composerln(a)) }
func (g Logger) CriticalWhen(c bool, m any)                  { g.send(level.Critical, g.makeWhen(c, m)) }
func (g Logger) Error(m any)                                 { g.send(level.Error, m) }
func (g Logger) Errorf(m string, a ...any)                   { g.send(level.Error, composerf(m, a)) }
func (g Logger) Errorln(a ...any)                            { g.send(level.Error, composerln(a)) }
func (g Logger) ErrorWhen(c bool, m any)                     { g.send(level.Error, g.makeWhen(c, m)) }
func (g Logger) Warning(m any)                               { g.send(level.Warning, m) }
func (g Logger) Warningf(m string, a ...any)                 { g.send(level.Warning, composerf(m, a)) }
func (g Logger) Warningln(a ...any)                          { g.send(level.Warning, composerln(a)) }
func (g Logger) WarningWhen(c bool, m any)                   { g.send(level.Warning, g.makeWhen(c, m)) }
func (g Logger) Notice(m any)                                { g.send(level.Notice, m) }
func (g Logger) Noticef(m string, a ...any)                  { g.send(level.Notice, composerf(m, a)) }
func (g Logger) Noticeln(a ...any)                           { g.send(level.Notice, composerln(a)) }
func (g Logger) NoticeWhen(c bool, m any)                    { g.send(level.Notice, g.makeWhen(c, m)) }
func (g Logger) Info(m any)                                  { g.send(level.Info, m) }
func (g Logger) Infof(m string, a ...any)                    { g.send(level.Info, composerf(m, a)) }
func (g Logger) Infoln(a ...any)                             { g.send(level.Info, composerln(a)) }
func (g Logger) InfoWhen(c bool, m any)                      { g.send(level.Info, g.makeWhen(c, m)) }
func (g Logger) Debug(m any)                                 { g.send(level.Debug, m) }
func (g Logger) Debugf(m string, a ...any)                   { g.send(level.Debug, composerf(m, a)) }
func (g Logger) Debugln(a ...any)                            { g.send(level.Debug, composerln(a)) }
func (g Logger) DebugWhen(c bool, m any)                     { g.send(level.Debug, g.makeWhen(c, m)) }
func (g Logger) Trace(m any)                                 { g.send(level.Trace, m) }
func (g Logger) Tracef(m string, a ...any)                   { g.send(level.Trace, composerf(m, a)) }
func (g Logger) Traceln(a ...any)                            { g.send(level.Trace, composerln(a)) }
func (g Logger) TraceWhen(c bool, m any)                     { g.send(level.Trace, g.makeWhen(c, m)) }

func Clone() Logger                                     { return std.Clone() }
func Sender() send.Sender                               { return std.Sender() }
func Build() *message.Builder                           { return std.Build() }
func Convert(m any) message.Composer                    { return std.Convert(m) }
func SetSender(s send.Sender)                           { std.SetSender(s) }
func SetConverter(c message.Converter)                  { std.SetConverter(c) }
func Log(l level.Priority, msg any)                     { std.Log(l, msg) }
func Logf(l level.Priority, msg string, a ...any)       { std.Logf(l, msg, a...) }
func Logln(l level.Priority, a ...any)                  { std.Logln(l, a...) }
func LogWhen(conditional bool, l level.Priority, m any) { std.LogWhen(conditional, l, m) }
func EmergencyPanic(msg any)                            { std.EmergencyPanic(msg) }
func EmergencyFatal(msg any)                            { std.EmergencyFatal(msg) }
func Emergency(msg any)                                 { std.Emergency(msg) }
func Emergencyf(msg string, a ...any)                   { std.Emergencyf(msg, a...) }
func Emergencyln(a ...any)                              { std.Emergencyln(a...) }
func EmergencyWhen(conditional bool, m any)             { std.EmergencyWhen(conditional, m) }
func Alert(msg any)                                     { std.Alert(msg) }
func Alertf(msg string, a ...any)                       { std.Alertf(msg, a...) }
func Alertln(a ...any)                                  { std.Alertln(a...) }
func AlertWhen(conditional bool, m any)                 { std.AlertWhen(conditional, m) }
func Critical(msg any)                                  { std.Critical(msg) }
func Criticalf(msg string, a ...any)                    { std.Criticalf(msg, a...) }
func Criticalln(a ...any)                               { std.Criticalln(a...) }
func CriticalWhen(conditional bool, m any)              { std.CriticalWhen(conditional, m) }
func Error(msg any)                                     { std.Error(msg) }
func Errorf(msg string, a ...any)                       { std.Errorf(msg, a...) }
func Errorln(a ...any)                                  { std.Errorln(a...) }
func ErrorWhen(conditional bool, m any)                 { std.ErrorWhen(conditional, m) }
func Warning(msg any)                                   { std.Warning(msg) }
func Warningf(msg string, a ...any)                     { std.Warningf(msg, a...) }
func Warningln(a ...any)                                { std.Warningln(a...) }
func WarningWhen(conditional bool, m any)               { std.WarningWhen(conditional, m) }
func Notice(msg any)                                    { std.Notice(msg) }
func Noticef(msg string, a ...any)                      { std.Noticef(msg, a...) }
func Noticeln(a ...any)                                 { std.Noticeln(a...) }
func NoticeWhen(conditional bool, m any)                { std.NoticeWhen(conditional, m) }
func Info(msg any)                                      { std.Info(msg) }
func Infof(msg string, a ...any)                        { std.Infof(msg, a...) }
func Infoln(a ...any)                                   { std.Infoln(a...) }
func InfoWhen(conditional bool, message any)            { std.InfoWhen(conditional, message) }
func Debug(msg any)                                     { std.Debug(msg) }
func Debugf(msg string, a ...any)                       { std.Debugf(msg, a...) }
func Debugln(a ...any)                                  { std.Debugln(a...) }
func DebugWhen(conditional bool, m any)                 { std.DebugWhen(conditional, m) }
func Trace(msg any)                                     { std.Trace(msg) }
func Tracef(msg string, a ...any)                       { std.Tracef(msg, a...) }
func Traceln(a ...any)                                  { std.Traceln(a...) }
func TraceWhen(conditional bool, m any)                 { std.TraceWhen(conditional, m) }

// implementation

///////////////////////////////////
//
// method implementation

func (g Logger) send(l level.Priority, in any) {
	m := g.Convert(in)
	m.SetPriority(l)
	g.impl.Get().Send(m)
}

// Convert runs the custom converter if set, falling back to
// message.Convert if indicated by the custom converter or if the
// custom converter is not set.

func (g Logger) makeWhen(cond bool, m any) message.Composer {
	return message.When(cond, message.MakeFuture(func() message.Composer { return g.Convert(m) }))
}

// For sending logging messages, in most cases, use the
// Journaler.sender.Send() method, but we have a couple of methods to
// use for the Panic/Fatal helpers.
func (g Logger) sendPanic(l level.Priority, in any) {
	m := g.Convert(in)
	m.SetPriority(l)

	s := g.impl.Get()

	// the Send method in the Sender interface will perform this
	// check but to add fatal methods we need to do this here.
	if send.ShouldLog(s, m) {
		s.Send(m)
		panic(m.String())
	}
}

func (g Logger) sendFatal(l level.Priority, in any) {
	m := g.Convert(in)
	m.SetPriority(l)

	s := g.impl.Get()

	// the Send method in the Sender interface will perform this
	// check but to add fatal methods we need to do this here.
	if send.ShouldLog(s, m) {
		s.Send(m)
		os.Exit(1)
	}
}
