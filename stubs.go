package main

import (
	"fmt"
	"os"

	"github.com/danwakefield/gosh/T"
)

func ExitShellWithMessage(ex T.ExitStatus, msg string) {
	fmt.Println(msg)
	os.Exit(int(ex))
}
