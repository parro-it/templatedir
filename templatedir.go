// Package templatedir provide two
// functions that allow to render a
// whole directory of go templates
// to itself or to another directory.
package templatedir

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"text/template"

	"github.com/parro-it/vs/syncfs"
	"github.com/parro-it/vs/writefs"
)

// RenderTo ...
func RenderTo(srcfs fs.FS, destfsys writefs.WriteFS, args interface{}) error {

	destfs := syncfs.New(destfsys).(writefs.WriteFS)

	res, walkErrs := walkDir(srcfs)
	allFilesDone := sync.WaitGroup{}
	allFilesDone.Add(runtime.NumCPU())
	errs := make(SyncErrors, 1)
	r := renderer{
		srcfs:  srcfs,
		destfs: destfs,
		args:   args,
	}
	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			defer allFilesDone.Done()
			for src := range res {
				fmt.Println("src --> ", src)
				if errs.Failed() {
					return
				}

				if errs.SetFailedOnErr(r.renderFile(src)) {
					return
				}
			}
		}()
	}
	err := <-walkErrs

	allFilesDone.Wait()

	if err == nil {
		err = errs.Close()
	}
	return err
}

func walkDir(fsys fs.FS) (chan string, chan error) {
	res := make(chan string)
	errs := make(chan error)
	go func() {
		err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
			if !d.IsDir() && strings.HasSuffix(path, ".template") {
				res <- path
			}
			return nil
		})
		close(res)
		if err != nil {
			errs <- err
		}
		close(errs)
	}()

	return res, errs
}

func mkDirRec(fsys fs.FS, dir string) error {
	destdir := strings.Split(dir, "/")
	if len(destdir) == 0 {
		destdir = []string{"."}
	}

	var pathAccum string

	for _, seg := range destdir {
		if pathAccum != "" {
			pathAccum += "/"
		}
		pathAccum += seg

		err := writefs.MkDir(fsys, pathAccum, fs.FileMode(0644))
		if err != nil && !errors.Is(err, fs.ErrExist) {
			return err
		}
	}

	return nil
}

type renderer struct {
	srcfs  fs.FS
	destfs writefs.WriteFS
	args   interface{}
}

func (r renderer) renderFile(src string) error {
	tmpl, err := template.ParseFS(r.srcfs, src)
	if err != nil {
		return err
	}

	err = mkDirRec(r.destfs, path.Dir(src))
	if err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}

	err = writefs.Remove(r.destfs, src)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	outname := strings.TrimSuffix(src, ".template")

	t, err := template.New("filename").Parse(outname)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	err = t.Execute(&buf, r.args)
	if err != nil {
		return err
	}
	outname = buf.String()

	dest, err := writefs.OpenFile(r.destfs, outname, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fs.FileMode(0644))
	if err != nil {
		return err
	}
	defer dest.Close()

	return tmpl.Execute(dest, r.args)
}
