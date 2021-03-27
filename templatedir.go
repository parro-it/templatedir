// Package templatedir provide two
// functions that allow to render a
// whole directory of go templates
// to itself or to another directory.
package templatedir

import (
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"

	"github.com/parro-it/vs/syncfs"
	"github.com/parro-it/vs/writefs"
)

// RenderTo ...
func RenderTo(srcfs fs.FS, destfsys writefs.WriteFS) error {

	destfs := syncfs.New(destfsys).(writefs.WriteFS)

	res, walkErrs := walkDir(srcfs)
	allFilesDone := sync.WaitGroup{}
	allFilesDone.Add(runtime.NumCPU())
	errs := make(SyncErrors)
	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			defer allFilesDone.Done()
			for src := range res {
				fmt.Println("src --> ", src)
				if errs.Failed() {
					return
				}

				if errs.SetFailedOnErr(renderFile(srcfs, destfs, src)) {
					return
				}
			}
		}()
	}
	err := <-walkErrs

	allFilesDone.Wait()

	if err == nil {
		select {
		case err = <-errs:
		default:
		}
	}
	close(errs)
	return err
}

// RenderToSelf ...
func RenderToSelf(fsys writefs.WriteFS) error {
	return nil
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

func renderFile(srcfs fs.FS, destfs writefs.WriteFS, src string) error {
	tmpl, err := template.ParseFS(srcfs, src)
	if err != nil {
		return err
	}

	destdir := strings.Split(path.Dir(src), "/")
	if len(destdir) == 0 {
		destdir = []string{"."}
	}

	var pathAccum string
	fmt.Println("destdir", destdir)

	for _, seg := range destdir {
		if pathAccum != "" {
			pathAccum += "/"
		}
		pathAccum += seg

		err = writefs.MkDir(destfs, pathAccum, fs.FileMode(0644))
		if err != nil && !errors.Is(fs.ErrExist, err) {

			return err
		}
		fmt.Println("created dir", pathAccum)
	}

	outname := src[:len(src)-len(".template")]
	dest, err := writefs.OpenFile(destfs, outname, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fs.FileMode(0644))
	if err != nil {
		fmt.Println("open err", err)
		return err
	}
	defer dest.Close()
	return tmpl.Execute(dest, map[string]int{
		"Count": 42,
	})
}
