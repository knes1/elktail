/* Copyright (C) 2016 Krešimir Nesek
 *
 * Based on idea by William Kennedy: https://www.goinggo.net/2013/11/using-log-package-in-go.html
 *
 * This software may be modified and distributed under the terms
 * of the MIT license.  See the LICENSE file for details.
 */
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
