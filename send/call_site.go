/*
Call Site Sender

Call site loggers provide a way to record the line number and file
name where the logging call was made, which is particularly useful in
tracing down log messages.

This sender does *not* attach this data to the Message object, and the
call site information is only logged when formatting the message
itself. Additionally the call site includes the file name and its
enclosing directory.

When constructing the Sender you must specify a "depth"
argument This sets the offset for the call site relative to the
Sender's Send() method. Grip's default logger (e.g. the grip.Info()
methods and friends) requires a depth of 2, while in *most* other
cases you will want to use a depth of 1. The LogMany, and
Emergency[Panic,Fatal] methods also include an extra level of
indirection.

Create a call site logger with one of the following constructors:

   NewCallSiteConsoleLogger(<name>, <depth>, <LevelInfo>)
   MakeCallSiteConsoleLogger(<depth>)
   NewCallSiteFileLogger(<name>, <fileName>, <depth>, <LevelInfo>)
   MakeCallSiteFileLogger(<fileName>, <depth>)
*/
package send

// NewCallSit returns a fully configured Sender
// implementation that writes log messages to standard output in a
// format that includes the filename and line number of the call site
// of the logger.
func NewCallSit(name string, depth int, l LevelInfo) (Sender, error) {
	return setup(MakeCallSite(depth), name, l)
}

// MakeCallSite constructs an unconfigured call site
// logger that writes output to standard output. You must set the name
// of the logger using SetName or your Journaler's SetSender method
// before using this logger.
func MakeCallSite(depth int) Sender {
	s := MakeStdOutput()
	_ = s.SetFormatter(MakeCallSiteFormatter(depth))

	return s
}

// NewCallSiteFile returns a fully configured Sender
// implementation that writes log messages to a specified file in a
// format that includes the filename and line number of the call site
// of the logger.
func NewCallSiteFile(name, fileName string, depth int, l LevelInfo) (Sender, error) {
	s, err := MakeCallSiteFile(fileName, depth)
	if err != nil {
		return nil, err
	}

	return setup(s, name, l)
}

// MakeCallSiteFile constructs an unconfigured call site logger
// that writes output to the specified hours. You must set the name of
// the logger using SetName or your Journaler's SetSender method
// before using this logger.
func MakeCallSiteFile(fileName string, depth int) (Sender, error) {
	s, err := MakeFile(fileName)
	if err != nil {
		return nil, err
	}

	if err := s.SetFormatter(MakeCallSiteFormatter(depth)); err != nil {
		return nil, err
	}

	return s, nil
}
