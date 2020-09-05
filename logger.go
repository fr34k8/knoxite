/*
 * knoxite
 *     Copyright (c) 2020, Matthias Hartmann <mahartma@mahartma.com>
 *
 *   For license see LICENSE
 */

package knoxite

import (
	"fmt"
)

type Logger struct {
	VerbosityLevel int
}

var (
	logger Logger
	printV = func(verbosity Verbosity, s string) {
		fmt.Println(verbosity.String() + ": " + s)
	}
)

func (l *Logger) Log(verbosity Verbosity, s string) {
	switch verbosity {
	case Debug:
		if l.VerbosityLevel == Debug {
			printV(verbosity, s)
		}
		fallthrough
	case Info:
		if l.VerbosityLevel == Info {
			printV(verbosity, s)
		}
		fallthrough
	case Warning:
		if l.VerbosityLevel == Warning {
			printV(verbosity, s)
		}
	}
}
