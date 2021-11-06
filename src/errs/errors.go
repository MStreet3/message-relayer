package errs

type FatalSocketError struct {
}

func (e FatalSocketError) Error() string {
	return "network socket experienced a fatal error"
}
