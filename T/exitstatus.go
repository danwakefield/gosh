package T

// ExitStatus TODO
type ExitStatus int

const (
	ExitSuccess        ExitStatus = 0
	ExitFailure        ExitStatus = 1
	ExitNotExecutable  ExitStatus = 126
	ExitUnknownCommand ExitStatus = 127
)
