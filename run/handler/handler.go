package handler

type RunHandler interface {
	Run(cmds []string) error
}