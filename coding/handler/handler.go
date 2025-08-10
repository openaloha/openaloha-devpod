package handler

import (
	"io"

	"github.com/openaloha/openaloha-devpod/config"
	"github.com/openaloha/openaloha-devpod/run/handler"
)

type CodingHandler interface {
	Coding(question string, stdout io.Writer, stderr io.Writer) error
	FinishInit(config config.Config, runhandler *handler.RunHandler)
}
