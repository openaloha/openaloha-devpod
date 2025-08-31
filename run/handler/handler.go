package handler

import (
	"context"
	"io"
)

type RunHandler interface {
	Run(cmds []string, stdout io.Writer, stderr io.Writer, cancelChan chan context.CancelFunc) error
}