package handler

import (
	"fmt"
	"io"

	"openaloha.io/openaloha-devpod/coding/factory"
	"openaloha.io/openaloha-devpod/config"
	"openaloha.io/openaloha-devpod/constant"
	"openaloha.io/openaloha-devpod/run/handler"
)

type GeminiCodingHandler struct {
	runhandler handler.RunHandler
	baseurl    string
	apikey     string
}

func init() {
	factory.Register(constant.CODING_TYPE_GEMINI, &GeminiCodingHandler{})
}

func (c *GeminiCodingHandler) Coding(question string, stdout io.Writer, stderr io.Writer) error {
	err := c.runhandler.Run([]string{fmt.Sprintf("gemini -y -a -p \"%s\"", question)}, stdout, stderr)
	if err != nil {
		return err
	}
	return nil
}

func (c *GeminiCodingHandler) FinishInit(config config.Config, runhandler *handler.RunHandler) {
	c.runhandler = *runhandler
	c.baseurl = config.Coding.Gemini.Baseurl
	c.apikey = config.Coding.Gemini.Apikey
}
