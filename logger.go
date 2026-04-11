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

// Clone creates a new Logger with the same message sender and
// converter; however they are fully independent loggers.
func (g Logger) Clone() Logger                    { return MakeLogger(g.Sender(), g.conv.Get()) }
func (g Logger) Build() *message.Builder          { return message.NewBuilder(g.Sender().Send, g.conv.Get()) }
func (g Logger) Sender() send.Sender              { return g.impl.Get().Sender }
func (g Logger) Convert(m any) message.Composer   { return g.conv.Get().Convert(m) }
func (g Logger) SetSender(s send.Sender)          { g.impl.Set(sender{s}) }
func (g Logger) SetConverter(m message.Converter) { g.conv.Set(converter{m}) }
func (g Logger) Send(m message.Composer)          { g.Sender().Send(m) }
func (g Logger) Log(l level.Priority, m any)      { g.Send(g.make(l, m)) }
func (g Logger) EmergencyPanic(m any)             { g.sendPanic(level.Emergency, m) }
func (g Logger) EmergencyFatal(m any)             { g.sendFatal(level.Emergency, m) }
func (g Logger) Emergency(m any)                  { g.Log(level.Emergency, m) }
func (g Logger) Alert(m any)                      { g.Log(level.Alert, m) }
func (g Logger) Critical(m any)                   { g.Log(level.Critical, m) }
func (g Logger) Error(m any)                      { g.Log(level.Error, m) }
func (g Logger) Warning(m any)                    { g.Log(level.Warning, m) }
func (g Logger) Notice(m any)                     { g.Log(level.Notice, m) }
func (g Logger) Info(m any)                       { g.Log(level.Info, m) }
func (g Logger) Debug(m any)                      { g.Log(level.Debug, m) }
func (g Logger) Trace(m any)                      { g.Log(level.Trace, m) }

func MakeKV() *message.KV                            { return message.NewKV() }
func KV(key string, value any) *message.KV           { return MakeKV().KV(key, value) }
func Build() *message.Builder                        { return std.Build() }
func When(cond bool, m any) message.Composer         { return message.When(cond, m) }
func MPrintln(args ...any) message.Composer          { return message.MakeLines(args...) }
func MPrintf(t string, args ...any) message.Composer { return message.MakeFormat(t, args...) }
func Convert(m any) message.Composer                 { return std.Convert(m) }

func Clone() Logger                    { return std.Clone() }
func Sender() send.Sender              { return std.Sender() }
func SetSender(s send.Sender)          { std.SetSender(s) }
func SetConverter(c message.Converter) { std.SetConverter(c) }
func Send(m message.Composer)          { std.Send(m) }
func Log(l level.Priority, msg any)    { std.Log(l, msg) }
func EmergencyPanic(msg any)           { std.EmergencyPanic(msg) }
func EmergencyFatal(msg any)           { std.EmergencyFatal(msg) }
func Emergency(msg any)                { std.Emergency(msg) }
func Alert(msg any)                    { std.Alert(msg) }
func Critical(msg any)                 { std.Critical(msg) }
func Error(msg any)                    { std.Error(msg) }
func Warning(msg any)                  { std.Warning(msg) }
func Notice(msg any)                   { std.Notice(msg) }
func Info(msg any)                     { std.Info(msg) }
func Debug(msg any)                    { std.Debug(msg) }
func Trace(msg any)                    { std.Trace(msg) }

// implementation

///////////////////////////////////
//
// method implementation

func (g Logger) make(l level.Priority, in any) message.Composer {
	m := g.Convert(in)
	m.SetPriority(l)
	return m
}

func (g Logger) ms(l level.Priority, i any) (message.Composer, send.Sender) {
	return g.make(l, i), g.Sender()
}

func (g Logger) sendPanic(l level.Priority, in any) {
	if m, s := g.ms(l, in); send.ShouldLog(s, m) {
		s.Send(m)
		panic(m.String())
	}
}

func (g Logger) sendFatal(l level.Priority, in any) {
	// the Send method in the Sender interface will perform this
	// check but to add fatal methods we need to do this here.
	if m, s := g.ms(l, in); send.ShouldLog(s, m) {
		s.Send(m)
		os.Exit(1)
	}
}
