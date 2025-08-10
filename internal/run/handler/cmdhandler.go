package handler

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/openaloha/openaloha-devpod/run/factory"
	"github.com/openaloha/openaloha-devpod/run/handler"
)

type CmdRunHandler struct {
	Workspace string
}

func init() {
	factory.Register("cmd", func(workspace string) handler.RunHandler {
		return &CmdRunHandler{
			Workspace: workspace,
		}
	})
}

func (r *CmdRunHandler) Run(cmds []string, stdout io.Writer, stderr io.Writer) error {
	fmt.Printf("run cmd: %s\n", cmds)

	if len(cmds) == 0 {
		return nil
	}

	for _, cmd := range cmds {
		fmt.Printf("run cmd: %s\n", cmd)

		// 使用 shell 来执行命令，这样可以正确处理带参数的命令
		var cmdObj *exec.Cmd
		if strings.TrimSpace(cmd) == "" {
			continue
		}

		// 在 macOS/Linux 上使用 sh -c 来执行命令
		cmdObj = exec.Command("sh", "-c", cmd)
		cmdObj.Stdout = stdout
		cmdObj.Stderr = stderr

		// 设置命令的工作目录
		if r.Workspace != "" {
			cmdObj.Dir = r.Workspace
		}

		err := cmdObj.Run()
		if err != nil {
			return fmt.Errorf("run cmd %s failed, err: %w", cmd, err)
		}
	}

	return nil
}