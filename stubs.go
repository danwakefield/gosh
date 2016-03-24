package main

import (
	"fmt"
	"os"
)

func expandArg(s string) string {
	return s
}

func ExitShellWithMessage(ex ExitStatus, msg string) {
	fmt.Println(msg)
	os.Exit(int(ex))
}
