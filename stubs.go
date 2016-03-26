package main

import (
	"fmt"
	"os"
)

func ExitShellWithMessage(ex ExitStatus, msg string) {
	fmt.Println(msg)
	os.Exit(int(ex))
}
