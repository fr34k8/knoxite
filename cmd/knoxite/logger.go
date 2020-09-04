/*
 * knoxite
 *     Copyright (c) 2020, Matthias Hartmann <mahartma@mahartma.com>
 *
 *   For license see LICENSE
 */

package main

import "fmt"

type Verbosity int

const (
	Debug = iota
	Info
	Warning
)

func (v Verbosity) String() string {
	return [...]string{"Debug", "Info", "Warning", "Error"}[v]
}

type Logger struct {
	VerbosityLevel int
}

var (
	logger Logger
	printV = func(verbosity Verbosity, s string) {
		fmt.Println(verbosity.String() + ": " + s)
	}
)

func (l Logger) Log(verbosity Verbosity, s string) {
	switch l.VerbosityLevel {
	case Warning:
		printV(verbosity, s)
		fallthrough
	case Info:
		printV(verbosity, s)
		fallthrough
	case Debug:
		printV(verbosity, s)
	}
}
