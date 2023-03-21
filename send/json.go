package send

// MakeJSON builds a Sender instance that prints log
// messages in a JSON formatted to standard output. The JSON formated
// message is taken by calling the Raw() method on the
// message.Composer and Marshalling the results.
func MakeJSON() Sender {
	s := MakePlain()
	s.SetFormatter(MakeJSONFormatter())

	return s
}

// MakeJSONFile creates an un-configured JSON logger that writes
// output to the specified file.
func MakeJSONFile(file string) (Sender, error) {
	s, err := MakePlainFile(file)
	if err != nil {
		return nil, err
	}

	s.SetFormatter(MakeJSONFormatter())

	return s, nil
}
