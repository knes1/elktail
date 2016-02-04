package main

import (
	"log"
	"io"
)

var (
	Trace   *log.Logger
	Info    *log.Logger
	Error   *log.Logger
)

func InitLogging(traceHandle io.Writer, infoHandle io.Writer, errorHandle io.Writer, printLines bool) {
	flag := 0
	if printLines {
		flag = log.Lshortfile
	}

	Trace = log.New(traceHandle,
		"TRACE: ", flag)

	Info = log.New(infoHandle,
		"INFO: ", flag)

	Error = log.New(errorHandle,
		"ERROR: ", flag)
}
