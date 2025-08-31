package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"openaloha.io/openaloha-devpod/api"
	codingfactory "openaloha.io/openaloha-devpod/coding/factory"
	"openaloha.io/openaloha-devpod/config"
	"openaloha.io/openaloha-devpod/constant"
	_ "openaloha.io/openaloha-devpod/internal/coding/handler"
	_ "openaloha.io/openaloha-devpod/internal/run/handler"
	"openaloha.io/openaloha-devpod/run/factory"
	runhandler "openaloha.io/openaloha-devpod/run/handler"
	"openaloha.io/openaloha-devpod/runfunc"
	"openaloha.io/openaloha-devpod/sync"
)

func main() {
	// 创建可取消的context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// parse config from file or command line arguments
	config, err := parseConfig()
	if err != nil {
		fmt.Println("parse config error: ", err)
		return
	}

	// init workspace
	initWorkspace(config.Workspace)

	// init run handler
	handler, err := factory.New("cmd", config.Workspace)
	if err != nil {
		fmt.Println("init run handler error: ", err)
		return
	}

	// build init func
	initFunc := buildInitFunc(config, handler)

	// build refresh func
	refreshFunc := buildRefreshFunc(config, handler)

	// start sync job to sync code
	if err = startSync(ctx, config, initFunc, refreshFunc); err != nil {
		fmt.Println("start sync job error: ", err)
	}

	// init coding handler
	coding, err := codingfactory.New(constant.CODING_TYPE_GEMINI)
	coding.FinishInit(config, &handler)
	if err != nil {
		fmt.Println("init coding handler error: ", err)
		return
	}

	// start server
	fmt.Println("start server")
	server := api.NewDevPodServer(":10003", coding)
	errChan, err := server.ListenAndServe()
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		return
	}

	// 在goroutine中监听信号
	go func() {
		<-sigChan
		fmt.Println("\n收到停止信号，正在优雅停止...")
		// 优雅停止服务器
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("服务器停止失败: %v\n", err)
		}
		cancel() // 取消主context
	}()

	// 等待信号或服务器错误
	select {
	case <-ctx.Done():
		fmt.Println("程序已优雅停止")
	case err := <-errChan:
		if err != nil {
			fmt.Printf("Server error: %v\n", err)
		}
		cancel() // 如果服务器出错，也要取消context
	}
}

// parse config from YAML file
func parseConfig() (config.Config, error) {
	// define config file path flag
	configFile := flag.String("config", "config.yaml", "path to YAML configuration file")
	flag.Parse()

	// load config from file
	cfg, err := config.LoadFromFile(*configFile)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to load config file %s: %v", *configFile, err)
	}

	// TODO:validate config

	return *cfg, nil
}

// init workspace
func initWorkspace(workspace string) error {
	// clear workspace
	if err := os.RemoveAll(workspace); err != nil {
		return err
	}

	// create workspace
	if err := os.MkdirAll(workspace, 0755); err != nil {
		return err
	}

	return nil
}

// start sync job to sync code
func startSync(ctx context.Context, config config.Config, initFunc runfunc.InitFunc, refreshFunc runfunc.RefreshFunc) error {
	syncFacade := &sync.SyncFacade{
		Config: config,
	}
	return syncFacade.Sync(ctx, initFunc, refreshFunc)
}

// buildInitFunc returns a new InitFunc
func buildInitFunc(config config.Config, handler runhandler.RunHandler) runfunc.InitFunc {
	return func() error {
		fmt.Println("init func")
		// run init cmd
		if err := handler.Run(config.Run.Init.Cmds, os.Stdout, os.Stderr); err != nil {
			return err
		}
		return nil
	}
}

// buildRefreshFunc returns a new RefreshFunc
func buildRefreshFunc(config config.Config, handler runhandler.RunHandler) runfunc.RefreshFunc {
	return func(files []*os.File) error {
		fmt.Println("refresh func")

		// get refresh cmd
		refreshCmds, err := matchRefreshCmd(files, config.Run.Refresh)
		if err != nil {
			return err
		}

		// run refresh cmd
		if err := handler.Run(refreshCmds, os.Stdout, os.Stderr); err != nil {
			return err
		}
		return nil
	}
}

// match refresh cmd
func matchRefreshCmd(files []*os.File, refreshConfigs []config.RefreshCmd) ([]string, error) {
	// sort files group by file suffix
	fileMap := make(map[string][]*os.File)
	for _, file := range files {
		fileName := file.Name()
		// get file suffix
		fileSuffix := getFileSuffix(fileName)
		fileMap[fileSuffix] = append(fileMap[fileSuffix], file)
	}

	for _, refreshConfig := range refreshConfigs {
		if matchRefreshCmdCondition(fileMap, refreshConfig.Condition) {
			return refreshConfig.Cmds, nil
		}
	}

	return nil, errors.New("no match refresh config")
}

// match condition
func matchRefreshCmdCondition(fileMap map[string][]*os.File, condition string) bool {
	if condition == "" {
		return true
	}

	// config1：*.java/*.go/*.py
	if strings.HasPrefix(condition, "*.") {
		// get file suffix
		fileSuffix := condition[2:]
		// match file suffix
		if files, ok := fileMap[fileSuffix]; ok {
			if len(files) > 0 {
				return true
			}
		}
	}

	return false
}

// get file suffix
func getFileSuffix(fileName string) string {
	fileSuffix := ""
	if lastDotIndex := strings.LastIndex(fileName, "."); lastDotIndex != -1 {
		fileSuffix = fileName[lastDotIndex+1:]
	}
	return fileSuffix
}
