package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/openaloha/openaloha-devpod/config"
	"github.com/openaloha/openaloha-devpod/constant"
	_ "github.com/openaloha/openaloha-devpod/internal/run/handler"
	"github.com/openaloha/openaloha-devpod/run/factory"
	runhandler "github.com/openaloha/openaloha-devpod/run/handler"
	"github.com/openaloha/openaloha-devpod/runfunc"
	"github.com/openaloha/openaloha-devpod/sync"
)

func main() {
	// parse config from environment variables
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
	if err := startSync(config, initFunc, refreshFunc); err != nil {
		fmt.Println("start sync job error: ", err)
		return
	}
}

// parse config from environment variables
func parseConfig() (config.Config, error) {
	// init config
	var config config.Config

	// parse config
	// common config
	flag.StringVar(&config.Workspace, "workspace", constant.DEFAULT_WORKSPACE, "workspace")
	// sync config
	flag.StringVar(&config.Sync.Type, "sync.type", constant.SYNC_TYPE_GIT, " sync type")
	flag.StringVar(&config.Sync.Git.Url, "sync.git.url", "", "sync url for git")
	flag.StringVar(&config.Sync.Git.Branch, "sync.git.branch", "", "git branch")
	flag.StringVar(&config.Sync.Git.SyncInterval, "sync.git.syncInterval", "", "git syncInterval")
	// run config
	initCmd := flag.String("run.init.cmds", "", "init cmd")
	refreshCmd := flag.String("run.refresh", "[]", "refresh cmd")

	// TODO:validate config

	flag.Parse()

	if initCmd != nil && *initCmd != "" {
		config.Run.Init.Cmds = strings.Split(*initCmd, ";")
	}
	if refreshCmd != nil && *refreshCmd != "" {
		if err := json.Unmarshal([]byte(*refreshCmd), &config.Run.Refresh); err != nil {
			return config, err
		}
	}

	return config, nil
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
func buildInitFunc(config config.Config, handler runhandler.RunHandler) runfunc.InitFunc {
	return func() error {
		fmt.Println("init func")
		// run init cmd
		if err := handler.Run(config.Run.Init.Cmds); err != nil {
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
		if err := handler.Run(refreshCmds); err != nil {
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
