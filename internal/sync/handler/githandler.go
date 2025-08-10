package handler

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/openaloha/openaloha-devpod/config"
	"github.com/openaloha/openaloha-devpod/constant"
	"github.com/openaloha/openaloha-devpod/runfunc"
	"github.com/openaloha/openaloha-devpod/sync/factory"
)

// GitSyncHandler is the handler for the git sync service
type GitSyncHandler struct {
}

func init() {
	factory.Register(constant.SYNC_TYPE_GIT, &GitSyncHandler{})
}

// Init is the method to initialize code
func (h *GitSyncHandler) Init(workspace string, syncConfig config.SyncConfig, initFunc runfunc.InitFunc) error {
	// git clone
	if err := h.GitClone(workspace, syncConfig.Git.Url, syncConfig.Git.Branch); err != nil {
		return err
	}

	// execute init func
	initFunc()

	return nil
}

// Refresh is the method to refresh code
func (h *GitSyncHandler) Refresh(ctx context.Context, workspace string, syncConfig config.SyncConfig, refreshFunc runfunc.RefreshFunc) error {
	// parse sync interval
	duration, err := time.ParseDuration(syncConfig.Git.SyncInterval)
	if err != nil {
		return err
	}

	// git pull from repo
	ticker := time.NewTicker(duration)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			fmt.Println("[git sync handler] 收到停止信号，正在退出...")
			return ctx.Err()
		case <-ticker.C:
			fmt.Printf("[%s] 执行任务, 当前时间: %s\n",
				"git sync handler",
				time.Now().Format("2006-01-02 15:04:05"))
			// git pull and get updated files
			updatedFiles, err := h.GitPullWithFiles(workspace, syncConfig.Git.Branch)
			if err != nil {
				fmt.Println("git pull error, err", err)
				continue
			}

			if len(updatedFiles) == 0 {
				continue
			}

			// convert file paths to file objects
			files := make([]*os.File, 0, len(updatedFiles))
			for _, filePath := range updatedFiles {
				// 构建完整的文件路径
				fullPath := workspace + "/" + filePath
				file, err := os.Open(fullPath)
				if err == nil {
					files = append(files, file)
				}
				defer file.Close()
			}

			// execute refresh func with updated files
			if err := refreshFunc(files); err != nil {
				fmt.Println("refresh func error, err", err)
			}
		}
	}

	return nil
}

// GitClone is the method to git clone
func (h *GitSyncHandler) GitClone(workspace string, url string, branch string) error {
	// git clone from repo
	if _, err := git.PlainClone(workspace, &git.CloneOptions{
		URL:           url,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		Progress:      os.Stdout,
	}); err != nil {
		return err
	}
	return nil
}

// GitPull is the method to git pull
func (h *GitSyncHandler) GitPull(workspace string, branch string) error {
	// open git repo
	repo, err := git.PlainOpen(workspace)
	if err != nil {
		return err
	}

	// Get the working directory for the repository
	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	// 获取pull前的HEAD commit
	ref, err := repo.Head()
	if err != nil {
		return err
	}
	oldCommit := ref.Hash()

	// git pull from repo
	if err = worktree.Pull(&git.PullOptions{
		RemoteName: "origin",
		Progress:   os.Stdout,
	}); err != nil {
		// 如果是"already up-to-date"错误，不算真正的错误
		if err == git.NoErrAlreadyUpToDate {
			fmt.Println("Repository is already up-to-date")
			return nil
		}
		return err
	}

	// 获取pull后的HEAD commit
	newRef, err := repo.Head()
	if err != nil {
		return err
	}
	newCommit := newRef.Hash()

	// 获取本次更新的文件列表
	updatedFiles, err := h.getUpdatedFiles(repo, oldCommit, newCommit)
	if err != nil {
		return err
	}

	// 打印更新的文件
	if len(updatedFiles) > 0 {
		fmt.Printf("Updated files in this pull:\n")
		for _, file := range updatedFiles {
			fmt.Printf("  %s\n", file)
		}
	} else {
		fmt.Println("No files updated in this pull")
	}

	return nil
}

// GitPullWithFiles is the method to git pull and return updated files
func (h *GitSyncHandler) GitPullWithFiles(workspace string, branch string) ([]string, error) {
	// open git repo
	repo, err := git.PlainOpen(workspace)
	if err != nil {
		return nil, err
	}

	// Get the working directory for the repository
	worktree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	// 获取pull前的HEAD commit
	ref, err := repo.Head()
	if err != nil {
		return nil, err
	}
	oldCommit := ref.Hash()

	// git pull from repo
	if err = worktree.Pull(&git.PullOptions{
		RemoteName: "origin",
		Progress:   os.Stdout,
	}); err != nil {
		// 如果是"already up-to-date"错误，返回空列表
		if err == git.NoErrAlreadyUpToDate {
			return []string{}, nil
		}
		return nil, err
	}

	// 获取pull后的HEAD commit
	newRef, err := repo.Head()
	if err != nil {
		return nil, err
	}
	newCommit := newRef.Hash()

	// 获取本次更新的文件列表
	return h.getUpdatedFiles(repo, oldCommit, newCommit)
}

// getUpdatedFiles 获取两个commit之间变更的文件列表
func (h *GitSyncHandler) getUpdatedFiles(repo *git.Repository, oldCommit, newCommit plumbing.Hash) ([]string, error) {
	// 如果commit相同，说明没有更新
	if oldCommit == newCommit {
		return []string{}, nil
	}

	// 获取两个commit对象
	oldCommitObj, err := repo.CommitObject(oldCommit)
	if err != nil {
		return nil, err
	}

	newCommitObj, err := repo.CommitObject(newCommit)
	if err != nil {
		return nil, err
	}

	// 获取两个commit的tree
	oldTree, err := oldCommitObj.Tree()
	if err != nil {
		return nil, err
	}

	newTree, err := newCommitObj.Tree()
	if err != nil {
		return nil, err
	}

	// 比较两个tree，获取变更的文件
	changes, err := object.DiffTree(oldTree, newTree)
	if err != nil {
		return nil, err
	}

	var updatedFiles []string
	for _, change := range changes {
		// 获取文件路径
		if change.From.Name != "" {
			// 删除或修改的文件
			updatedFiles = append(updatedFiles, change.From.Name)
		}
		if change.To.Name != "" && change.To.Name != change.From.Name {
			// 新增或重命名的文件
			updatedFiles = append(updatedFiles, change.To.Name)
		}
	}

	return updatedFiles, nil
}
