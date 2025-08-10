package handler

import "io"

type RunHandler interface {
	Run(cmds []string, stdout io.Writer, stderr io.Writer) error
}