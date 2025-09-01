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
	// 创建主context用于优雅关闭
	mainCtx, mainCancel := context.WithCancel(context.Background())
	defer mainCancel()

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// parse config from file or command line arguments
	config, err := parseConfig()
	if err != nil {
		fmt.Println("parse config error: ", err)
		return
	}
	fmt.Println("config: ", config)

	// init workspace
	initWorkspace(config.Workspace)

	// init run handler
	handler, err := factory.New("cmd", config.Workspace)
	if err != nil {
		fmt.Println("init run handler error: ", err)
		return
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
	serverErrChan := server.ListenAndServe()

	var cancelChan chan context.CancelFunc = make(chan context.CancelFunc, 10)
	
	// build init func
	initFunc := buildInitFunc(config, handler, cancelChan, mainCtx)

	// build refresh func
	refreshFunc := buildRefreshFunc(config, handler, cancelChan, mainCtx)

	// 在goroutine中启动sync job
	syncErrChan := make(chan error, 1)
	go func() {
		if err := startSync(config, initFunc, refreshFunc); err != nil {
			fmt.Println("start sync job error: ", err)
			syncErrChan <- err
		}
	}()

	// 等待信号或错误
	select {
	case sig := <-sigChan:
		fmt.Printf("\n收到信号 %v，开始优雅关闭...\n", sig)
		mainCancel()
		
		// 优雅关闭服务器
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("服务器关闭错误: %v\n", err)
		}
		
		// 取消所有正在运行的命令
		for {
			select {
			case cancelFunc := <-cancelChan:
				cancelFunc()
			default:
				goto cleanup
			}
		}
		
	case err := <-serverErrChan:
		if err != nil {
			fmt.Printf("服务器错误: %v\n", err)
		}
		mainCancel()
		
	case err := <-syncErrChan:
		if err != nil {
			fmt.Printf("同步任务错误: %v\n", err)
		}
		mainCancel()
	}

	cleanup:
	fmt.Println("程序已优雅关闭")
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
func startSync(config config.Config, initFunc runfunc.InitFunc, refreshFunc runfunc.RefreshFunc) error {
	syncFacade := &sync.SyncFacade{
		Config: config,
	}
	return syncFacade.Sync(initFunc, refreshFunc)
}

// buildInitFunc returns a new InitFunc
func buildInitFunc(config config.Config, handler runhandler.RunHandler, cancelChan chan context.CancelFunc, ctx context.Context) runfunc.InitFunc {
	return func() error {
		go func() error {
			fmt.Println("init func")
			// 检查context是否已取消
			select {
			case <-ctx.Done():
				fmt.Println("init func cancelled")
				return ctx.Err()
			default:
			}
			// run init cmd
			if err := handler.Run(config.Run.Init.Cmds, os.Stdout, os.Stderr, cancelChan); err != nil {
				return err
			}
			return nil
		}()
		return nil
	}
}

// buildRefreshFunc returns a new RefreshFunc
func buildRefreshFunc(config config.Config, handler runhandler.RunHandler, cancelChan chan context.CancelFunc, ctx context.Context) runfunc.RefreshFunc {
	return func(files []*os.File) error {
		
		fmt.Println("refresh func")

		// 检查context是否已取消
		select {
		case <-ctx.Done():
			fmt.Println("refresh func cancelled")
			return ctx.Err()
		default:
		}

		// get refresh cmd
		refreshCmds, err := matchRefreshCmd(files, config.Run.Refresh)
		if err != nil {
			fmt.Println("match refresh cmd error: ", err)
			return err
		}
		if len(refreshCmds) == 0 {
			return nil
		}

	loop:
		for {
			select{
				// cancel all running cmd
				case cancelItem := <-cancelChan:
					fmt.Println("cancel running cmd")
					cancelItem()
				case <-ctx.Done():
					fmt.Println("refresh func cancelled during execution")
					return ctx.Err()
				default:
					// run refresh cmd
					go func() error {
						fmt.Println("run refresh cmd")
						if err := handler.Run(refreshCmds, os.Stdout, os.Stderr, cancelChan); err != nil {
							fmt.Println("run refresh cmd error: ", err)
							return err
						}
						return nil
					}()
					break loop
			}
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
