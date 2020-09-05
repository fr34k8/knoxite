/*
 * knoxite
 *     Copyright (c) 2020, Matthias Hartmann <mahartma@mahartma.com>
 *
 *   For license see LICENSE
 */

package knoxite

type Verbosity int

const (
	Debug = iota
	Info
	Warning
)

func (v Verbosity) String() string {
	return [...]string{"Debug", "Info", "Warning", "Error"}[v]
}
