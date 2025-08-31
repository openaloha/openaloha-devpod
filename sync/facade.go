package sync

import (
	"fmt"
	"openaloha.io/openaloha-devpod/config"
	_ "openaloha.io/openaloha-devpod/internal/sync/handler"
	"openaloha.io/openaloha-devpod/runfunc"
	"openaloha.io/openaloha-devpod/sync/factory"
	"openaloha.io/openaloha-devpod/sync/handler"
)

// SyncFacade is the facade for the sync service
type SyncFacade struct {
	Config config.Config
}

// Sync is the method to init and refresh code
func (f *SyncFacade) Sync(initFunc runfunc.InitFunc, refreshFunc runfunc.RefreshFunc) error {
	// get sync handler by sync type
	syncHandler, err := getSyncHandler(f.Config.Sync.Type)
	if err != nil {
		return err
	}

	// init code by sync handler
	if err := syncHandler.Init(f.Config.Workspace, f.Config.Sync, initFunc); err != nil {
		fmt.Println("init code by sync handler error: ", err)
		return err
	}

	// refresh code by sync handler
	if err := syncHandler.Refresh(f.Config.Workspace, f.Config.Sync, refreshFunc); err != nil {
		fmt.Println("refresh code by sync handler error: ", err)
		return err
	}


	// execute init func
	// if err := initFunc(); err != nil {
	// 	fmt.Println("execute init func error: ", err)
	// }

	// // execute refresh func with updated files
	// for{
	// 	files := <- refreshFileStartFlag
	// 	fmt.Println("refresh func with updated files: ", files)
	// 	if err = refreshFunc(files); err != nil {
	// 		fmt.Println("refresh func error, err", err)
	// 	}
	// }

	return nil
}

// get sync handler by sync type
func getSyncHandler(syncType string) (handler.SyncHandler, error) {
	syncHandler, err := factory.New(syncType)
	if err != nil {
		return nil, err
	}
	return syncHandler, nil
}
