/*
 * knoxite
 *     Copyright (c) 2020, Matthias Hartmann <mahartma@mahartma.com>
 *
 *   For license see LICENSE
 */

package main

import (
	"fmt"

	"github.com/knoxite/knoxite"
)

type Logger struct {
	VerbosityLevel int
}

var (
	logger Logger
	printV = func(verbosity knoxite.Verbosity, s string) {
		fmt.Println(verbosity.String() + ": " + s)
	}
)

func (l *Logger) Log(verbosity knoxite.Verbosity, s string) {
	switch verbosity {
	case knoxite.Debug:
		if l.VerbosityLevel == knoxite.Debug {
			printV(verbosity, s)
		}
		fallthrough
	case knoxite.Info:
		if l.VerbosityLevel == knoxite.Info {
			printV(verbosity, s)
		}
		fallthrough
	case knoxite.Warning:
		if l.VerbosityLevel == knoxite.Warning {
			printV(verbosity, s)
		}
	}
}
