package main

import (
	"dotxt/cmd"
	"dotxt/pkg/logging"
	"os"
)

func main() {
	defer func() {
		err := logging.Close()
		if err != nil {
			os.Exit(2)
		}
	}()
	defer func() {
		if r := recover(); r != nil {
			logging.Logger.Fatalf("UNEXPECTED PANIC: %v\n%s", r)
		}
	}()
	if err := cmd.Execute(); err != nil {
		logging.Logger.Errorf("error running command: %v", err)
		os.Exit(1)
	}
}
