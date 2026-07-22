package logging

import (
	"io"
	"log"
	"os"
)

func newLoggers(stdout, stderr io.Writer) (*log.Logger, *log.Logger) {
	return log.New(stdout, "", log.LstdFlags), log.New(stderr, "", log.LstdFlags)
}

var Info, Error = newLoggers(os.Stdout, os.Stderr)
