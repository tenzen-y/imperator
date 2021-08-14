package main

import (
	"fmt"
	"github.com/tenzen-y/imperator/cmd/operator/cmd"
	"os"
)

func main() {
	command, err := cmd.NewRootCmd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err = command.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
