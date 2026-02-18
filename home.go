package main

import (
	"os"

	"github.com/steffakasid/eslog"
)

var home string

func initHome() {
	var err error
	home, err = os.UserHomeDir()
	eslog.LogIfErrorf(err, eslog.Fatalf, "Error getting user home directory: %s", err)
}
