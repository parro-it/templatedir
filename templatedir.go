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

	"github.com/parro-it/vs/writefs"
)

func RenderTo(srcfs fs.FS, destfs writefs.WriteFS) error {
	res, walkErrs := walkDir(srcfs)
	allFilesDone := sync.WaitGroup{}
	allFilesDone.Add(runtime.NumCPU())
	errs := make(chan error)
	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			defer allFilesDone.Done()
			for src := range res {
				fmt.Println("src --> ", src)
				select {
				case err := <-errs:
					select {
					case errs <- err:
						fmt.Println("goroutines failed: exit.", src)
					default:
					}
					return
				default:
				}

				err := renderFile(srcfs, destfs, src)
				if err != nil {
					fmt.Println("ERR", err.Error())
					select {
					case errs <- err:
					default:
					}
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
