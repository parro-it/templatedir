package templatedir

import (
	"embed"
	"errors"
	"io/fs"
	"testing"

	"github.com/parro-it/vs/memfs"
	"github.com/parro-it/vs/writefs"
	"github.com/stretchr/testify/assert"
)

//go:embed fixtures
var fixtureRootFS embed.FS
var fixtureFS, _ = fs.Sub(fixtureRootFS, "fixtures")

func TestWalkDir(t *testing.T) {
	res, errs := walkDir(fixtureFS)
	var actual []string
	for s := range res {
		actual = append(actual, s)
	}
	err := <-errs
	assert.NoError(t, err)
	assert.Equal(t, []string{
		"dir1/dir2/file3.txt.template",
		"dir1/dir3/file4.template",
	}, actual)
}

func TestRenderFile(t *testing.T) {
	outfs := memfs.NewFS()
	err := renderFile(fixtureFS, outfs, "dir1/dir2/file3.txt.template")
	assert.NoError(t, err)

	actual, err := fs.ReadFile(outfs, "dir1/dir2/file3.txt")
	assert.NoError(t, err)

	assert.Equal(t, "you pass 42.", string(actual))
}

func TestRenderTo(t *testing.T) {
	outfs := memfs.NewFS()
	err := RenderTo(fixtureFS, outfs)
	assert.NoError(t, err)

	actual, err := fs.ReadFile(outfs, "dir1/dir2/file3.txt")
	assert.NoError(t, err)
	assert.Equal(t, "you pass 42.", string(actual))

	actual, err = fs.ReadFile(outfs, "dir1/dir3/file4")
	assert.NoError(t, err)
	assert.Equal(t, "another 42.", string(actual))
}

func TestTemplateFilesRemovedFromDest(t *testing.T) {
	outfs := memfs.NewFS()
	err := writefs.MkDir(outfs, "dir1", fs.FileMode(0755))
	assert.NoError(t, err)
	err = writefs.MkDir(outfs, "dir1/dir2", fs.FileMode(0755))
	assert.NoError(t, err)
	_, err = writefs.WriteFile(outfs, "dir1/dir2/file3.txt.template", []byte{0x42})
	assert.NoError(t, err)

	err = RenderTo(fixtureFS, outfs)
	assert.NoError(t, err)

	actual, err := fs.ReadFile(outfs, "dir1/dir2/file3.txt")
	assert.NoError(t, err)
	assert.Equal(t, "you pass 42.", string(actual))

	_, err = fs.ReadFile(outfs, "dir1/dir2/file3.txt.template")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, fs.ErrNotExist))

}
