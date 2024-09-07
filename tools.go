package main

import (
	"errors"
	"os"
	"runtime"
)

func checkPermission() error {
	currentOS := runtime.GOOS
	if currentOS == "linux" && os.Getuid() != 0 {
		return errors.New("this program need root role.")
	}
	if currentOS != "linux" {
		return errors.New("only support linux os.")
	}
	return nil
}
