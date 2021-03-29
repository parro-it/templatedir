package templatedir

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

var pkgroot = func() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}()

func TestArgs(t *testing.T) {
	args, err := DefaultArgs()
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, pkgroot, args["Root"])
	assert.Equal(t, "parro-it", args["Author"])
	assert.Equal(t, "templatedir", args["RepoName"])

}

func TestGetRepoInfoFromGit(t *testing.T) {
	cwd, err := os.Getwd()
	if !assert.NoError(t, err) {
		return
	}
	info, err := getRepoInfoFromGit(cwd)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, repoInfo{
		Author:   "parro-it",
		RepoName: "templatedir",
		Root:     cwd,
	}, info)

}

func TestGetGHInfos(t *testing.T) {
	user, repo, err := getGHInfos(repoInfo{
		Author:   "parro-it",
		RepoName: "templatedir",
		Root:     pkgroot,
	})

	if !assert.NoError(t, err) ||
		(user == nil && repo == nil) {
		return
	}

	assert.Equal(t, "Templare aids you in making GH repos from GH template repos.", repo.GetDescription())
	assert.Equal(t, "Andrea Parodi", user.GetName())

}
