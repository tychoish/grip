package grip

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/send"
)

var std Logger

func init() {
	sender := send.WrapWriterLogger(log.Writer())
	if !strings.Contains(os.Args[0], "go-build") {
		sender.SetName(filepath.Base(os.Args[0]))
	} else {
		sender.SetName("grip")
	}

	std = MakeGrip(sender)
}

// SetDefaultStandardLogger set's the standard library's global
// logging instance to use grip's global logger at the specified
// level.
func SetDefaultStandardLogger(p level.Priority) {
	log.SetFlags(0)
	log.SetOutput(send.MakeWriterSender(std.GetSender(), p))
}

// MakeStandardLogger constructs a standard library logging instance
// that logs all messages to the global grip logging instance.
func MakeStandardLogger(p level.Priority) *log.Logger {
	return send.MakeStandardLogger(std.GetSender(), p)
}

// GetGlobalLogger returns the global journal instance used by
// this library. This call is not thread safe relative to other
// logging calls, or SetGlobalJournaler call, although all journaling
// methods are safe.
func GetGlobalLogger() Logger { return std }

// SetGlobalJournaler allows you to override the standard logger,
// that is used by calls in the grip package. This call is not thread
// safe relative to other logging calls, or the GetGlobalJournaler
// call, although all journaling methods are safe: as a result be sure
// to only call this method during package and process initialization.
func SetGlobalLogger(l Logger) { std = l }

func Log(l level.Priority, msg interface{})                     { std.Log(l, msg) }
func Logf(l level.Priority, msg string, a ...interface{})       { std.Logf(l, msg, a...) }
func LogWhen(conditional bool, l level.Priority, m interface{}) { std.LogWhen(conditional, l, m) }
func Send(msg interface{})                                      { std.Send(msg) }
func EmergencyFatal(msg interface{})                            { std.EmergencyFatal(msg) }
func Emergency(msg interface{})                                 { std.Emergency(msg) }
func Emergencyf(msg string, a ...interface{})                   { std.Emergencyf(msg, a...) }
func EmergencyPanic(msg interface{})                            { std.EmergencyPanic(msg) }
func EmergencyWhen(conditional bool, m interface{})             { std.EmergencyWhen(conditional, m) }
func Alert(msg interface{})                                     { std.Alert(msg) }
func Alertf(msg string, a ...interface{})                       { std.Alertf(msg, a...) }
func AlertWhen(conditional bool, m interface{})                 { std.AlertWhen(conditional, m) }
func Critical(msg interface{})                                  { std.Critical(msg) }
func Criticalf(msg string, a ...interface{})                    { std.Criticalf(msg, a...) }
func CriticalWhen(conditional bool, m interface{})              { std.CriticalWhen(conditional, m) }
func Error(msg interface{})                                     { std.Error(msg) }
func Errorf(msg string, a ...interface{})                       { std.Errorf(msg, a...) }
func ErrorWhen(conditional bool, m interface{})                 { std.ErrorWhen(conditional, m) }
func Warning(msg interface{})                                   { std.Warning(msg) }
func Warningf(msg string, a ...interface{})                     { std.Warningf(msg, a...) }
func WarningWhen(conditional bool, m interface{})               { std.WarningWhen(conditional, m) }
func Notice(msg interface{})                                    { std.Notice(msg) }
func Noticef(msg string, a ...interface{})                      { std.Noticef(msg, a...) }
func NoticeWhen(conditional bool, m interface{})                { std.NoticeWhen(conditional, m) }
func Info(msg interface{})                                      { std.Info(msg) }
func Infof(msg string, a ...interface{})                        { std.Infof(msg, a...) }
func InfoWhen(conditional bool, message interface{})            { std.InfoWhen(conditional, message) }
func Debug(msg interface{})                                     { std.Debug(msg) }
func Debugf(msg string, a ...interface{})                       { std.Debugf(msg, a...) }
func DebugWhen(conditional bool, m interface{})                 { std.DebugWhen(conditional, m) }
