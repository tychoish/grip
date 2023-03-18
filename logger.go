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
//
// These helpers also include Log* helpers to parameterize the level, as
// well as the Send method for default logging (or when the level is
// on the massage itself.)
package grip

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

var std Logger

func init() {
	sender := send.WrapWriter(log.Writer())
	if !strings.Contains(os.Args[0], "go-build") {
		sender.SetName(filepath.Base(os.Args[0]))
	} else {
		sender.SetName("grip")
	}

	_ = sender.SetLevel(send.LevelInfo{Default: level.Debug, Threshold: level.Info})

	std = NewLogger(sender)
}

type ctxKeyType struct{}

// WithLogger attaches a Logger instance to the context
func WithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, ctxKeyType{}, logger)
}

// Context resolves a logger from the given context, and if one does
// not exist (or the context is nil), produces the global Logger
// instance.
func Context(ctx context.Context) Logger {
	if ctx == nil {
		return std
	}

	val := ctx.Value(ctxKeyType{})
	if l, ok := val.(Logger); ok {
		return l
	}
	return std
}

// Logger provides the public interface of the grip Logger.
//
// Package level functions mirror all methods on the Logger type to
// access a "global" Logger instance in the grip package.
type Logger struct {
	impl send.Sender
}

// NewLogger builds a new logging interface from a sender implementation.
func NewLogger(s send.Sender) Logger     { return Logger{impl: s} }
func (g Logger) Sender() send.Sender     { return g.impl }
func (g Logger) Build() *message.Builder { return message.NewBuilder(g.impl.Send) }

// implementation

///////////////////////////////////
//
// method implementation

func (g Logger) safeSetPriority(l level.Priority, m message.Composer) bool {
	if err := m.SetPriority(l); err != nil {
		g.impl.ErrorHandler()(err, m)
		return true
	}

	return false
}

func (g Logger) send(l level.Priority, m message.Composer) {
	if g.safeSetPriority(l, m) {
		return
	}
	g.impl.Send(m)
}

// For sending logging messages, in most cases, use the
// Journaler.sender.Send() method, but we have a couple of methods to
// use for the Panic/Fatal helpers.
func (g Logger) sendPanic(l level.Priority, m message.Composer) {
	if g.safeSetPriority(l, m) {
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
	if g.safeSetPriority(l, m) {
		return
	}

	// the Send method in the Sender interface will perform this
	// check but to add fatal methods we need to do this here.
	if g.impl.Level().ShouldLog(m) {
		g.impl.Send(m)
		os.Exit(1)
	}
}

func makeWhen(cond bool, m any) message.Composer             { return message.When(cond, message.Convert(m)) }
func composerf(tmpl string, args []any) message.Composer     { return message.MakeFormat(tmpl, args...) }
func composerln(args []any) message.Composer                 { return message.MakeLines(args...) }
func (g Logger) Log(l level.Priority, m any)                 { g.send(l, message.Convert(m)) }
func (g Logger) Logf(l level.Priority, msg string, a ...any) { g.send(l, composerf(msg, a)) }
func (g Logger) Logln(l level.Priority, a ...any)            { g.send(l, composerln(a)) }
func (g Logger) LogWhen(c bool, l level.Priority, m any)     { g.send(l, makeWhen(c, m)) }
func (g Logger) EmergencyPanic(m any)                        { g.sendPanic(level.Emergency, message.Convert(m)) }
func (g Logger) EmergencyFatal(m any)                        { g.sendFatal(level.Emergency, message.Convert(m)) }
func (g Logger) Emergency(m any)                             { g.send(level.Emergency, message.Convert(m)) }
func (g Logger) Emergencyf(m string, a ...any)               { g.send(level.Emergency, composerf(m, a)) }
func (g Logger) Emergencyln(a ...any)                        { g.send(level.Emergency, composerln(a)) }
func (g Logger) EmergencyWhen(c bool, m any)                 { g.send(level.Emergency, makeWhen(c, m)) }
func (g Logger) Alert(m any)                                 { g.send(level.Alert, message.Convert(m)) }
func (g Logger) Alertf(m string, a ...any)                   { g.send(level.Alert, composerf(m, a)) }
func (g Logger) Alertln(a ...any)                            { g.send(level.Alert, composerln(a)) }
func (g Logger) AlertWhen(c bool, m any)                     { g.send(level.Alert, makeWhen(c, m)) }
func (g Logger) Critical(m any)                              { g.send(level.Critical, message.Convert(m)) }
func (g Logger) Criticalf(m string, a ...any)                { g.send(level.Critical, composerf(m, a)) }
func (g Logger) Criticalln(a ...any)                         { g.send(level.Critical, composerln(a)) }
func (g Logger) CriticalWhen(c bool, m any)                  { g.send(level.Critical, makeWhen(c, m)) }
func (g Logger) Error(m any)                                 { g.send(level.Error, message.Convert(m)) }
func (g Logger) Errorf(m string, a ...any)                   { g.send(level.Error, composerf(m, a)) }
func (g Logger) Errorln(a ...any)                            { g.send(level.Error, composerln(a)) }
func (g Logger) ErrorWhen(c bool, m any)                     { g.send(level.Error, makeWhen(c, m)) }
func (g Logger) Warning(m any)                               { g.send(level.Warning, message.Convert(m)) }
func (g Logger) Warningf(m string, a ...any)                 { g.send(level.Warning, composerf(m, a)) }
func (g Logger) Warningln(a ...any)                          { g.send(level.Warning, composerln(a)) }
func (g Logger) WarningWhen(c bool, m any)                   { g.send(level.Warning, makeWhen(c, m)) }
func (g Logger) Notice(m any)                                { g.send(level.Notice, message.Convert(m)) }
func (g Logger) Noticef(m string, a ...any)                  { g.send(level.Notice, composerf(m, a)) }
func (g Logger) Noticeln(a ...any)                           { g.send(level.Notice, composerln(a)) }
func (g Logger) NoticeWhen(c bool, m any)                    { g.send(level.Notice, makeWhen(c, m)) }
func (g Logger) Info(m any)                                  { g.send(level.Info, message.Convert(m)) }
func (g Logger) Infof(m string, a ...any)                    { g.send(level.Info, composerf(m, a)) }
func (g Logger) Infoln(a ...any)                             { g.send(level.Info, composerln(a)) }
func (g Logger) InfoWhen(c bool, m any)                      { g.send(level.Info, makeWhen(c, m)) }
func (g Logger) Debug(m any)                                 { g.send(level.Debug, message.Convert(m)) }
func (g Logger) Debugf(m string, a ...any)                   { g.send(level.Debug, composerf(m, a)) }
func (g Logger) Debugln(a ...any)                            { g.send(level.Debug, composerln(a)) }
func (g Logger) DebugWhen(c bool, m any)                     { g.send(level.Debug, makeWhen(c, m)) }
func (g Logger) Trace(m any)                                 { g.send(level.Trace, message.Convert(m)) }
func (g Logger) Tracef(m string, a ...any)                   { g.send(level.Trace, composerf(m, a)) }
func (g Logger) Traceln(a ...any)                            { g.send(level.Trace, composerln(a)) }
func (g Logger) TraceWhen(c bool, m any)                     { g.send(level.Trace, makeWhen(c, m)) }

// SetGlobalJournaler allows you to override the standard logger,
// that is used by calls in the grip package. This call is not thread
// safe relative to other logging calls, or the GetGlobalJournaler
// call, although all journaling methods are safe: as a result be sure
// to only call this method during package and process initialization.
func SetGlobalLogger(l Logger)                          { std = l }
func Sender() send.Sender                               { return std.Sender() }
func Build() *message.Builder                           { return std.Build() }
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
