package handler

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"

	"openaloha.io/openaloha-devpod/run/factory"
	"openaloha.io/openaloha-devpod/run/handler"
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


func (r *CmdRunHandler) Run(cmds []string, stdout io.Writer, stderr io.Writer, cancelChan chan context.CancelFunc) error {
	fmt.Printf("run cmd: %s\n", cmds)

	if len(cmds) == 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	if cancelChan != nil {
		cancelChan <- cancel
	}

	for _, cmd := range cmds {
		fmt.Printf("run cmd: %s\n", cmd)

		// 使用 shell 来执行命令，这样可以正确处理带参数的命令
		var cmdObj *exec.Cmd
		if strings.TrimSpace(cmd) == "" {
			continue
		}

		// 在 macOS/Linux 上使用 sh -c 来执行命令
		cmdObj = exec.CommandContext(ctx, "sh", "-c", cmd)
		cmdObj.Stdout = stdout
		cmdObj.Stderr = stderr

		// 关键：让子进程在一个新的进程组里
		cmdObj.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		// 设置命令的工作目录
		if r.Workspace != "" {
			cmdObj.Dir = r.Workspace
		}

		err := cmdObj.Start()
		if err != nil {
			return fmt.Errorf("run cmd %s failed, err: %w", cmd, err)
		}

		// 另起一个 goroutine 监听 ctx.Done()
		go func() {
			<-ctx.Done()
			// 给整个进程组发信号（-PID 表示进程组）
			_ = syscall.Kill(-cmdObj.Process.Pid, syscall.SIGKILL)
		}()

		_ = cmdObj.Wait()
	}

	return nil
}
