package builtins

import (
	"fmt"
	"os"
	"os/user"

	"github.com/danwakefield/gosh/T"
	"github.com/danwakefield/gosh/variables"
)

func CdCmd(scp *variables.Scope, ioc *T.IOContainer, args []string) T.ExitStatus {
	// This does not conform to the posix spec.
	// Very simplified. Cd to first arg or attempt to cd to home dir
	cdTarget := "."
	args = args[1:] // Cut out the cd from args list

	_ = "breakpoint"

	if len(args) == 0 {
		homeDir := scp.Get("HOME")
		if homeDir.Val == "" {
			// Implementation defined behaviour. We try to grab
			// the homedir of the current user
			u, err := user.Current()
			if err == nil {
				cdTarget = u.HomeDir
			}
		} else {
			cdTarget = homeDir.Val
		}
	} else {
		cdTarget = args[0]
	}

	if err := scp.SetPwd(cdTarget); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		return T.ExitFailure
	}
	return T.ExitSuccess
}
