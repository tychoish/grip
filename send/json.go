package send

// NewJSON builds a Sender instance that prints log
// messages in a JSON formatted to standard output. The JSON formated
// message is taken by calling the Raw() method on the
// message.Composer and Marshalling the results.
func NewJSON(name string, l LevelInfo) (Sender, error) {
	return setup(MakeJSON(), name, l)
}

// MakeJSON returns an un-configured JSON console logging
// instance.
func MakeJSON() Sender {
	s := MakePlain()
	_ = s.SetFormatter(MakeJSONFormatter())

	return s
}

// NewJSONFile builds a Sender instance that write JSON
// formated log messages to a file, with one-line per message. The
// JSON formated message is taken by calling the Raw() method on the
// message.Composer and Marshalling the results.
func NewJSONFile(name, file string, l LevelInfo) (Sender, error) {
	s, err := MakeJSONFile(file)
	if err != nil {
		return nil, err
	}

	return setup(s, name, l)
}

// MakeJSONFile creates an un-configured JSON logger that writes
// output to the specified file.
func MakeJSONFile(file string) (Sender, error) {
	s, err := MakePlainFile(file)
	if err != nil {
		return nil, err
	}

	if err = s.SetFormatter(MakeJSONFormatter()); err != nil {
		return nil, err
	}

	return s, nil
}
