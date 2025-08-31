package handler

import (
	"io"

	"openaloha.io/openaloha-devpod/config"
	"openaloha.io/openaloha-devpod/run/handler"
)

type CodingHandler interface {
	Coding(question string, stdout io.Writer, stderr io.Writer) error
	FinishInit(config config.Config, runhandler *handler.RunHandler)
}
