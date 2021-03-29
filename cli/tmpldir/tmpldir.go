package main

import (
	"fmt"
	"os"

	"github.com/parro-it/vs/osfs"

	"github.com/parro-it/templatedir"
)

func dieOnErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n\n", err.Error())
		os.Exit(1)
	}
}

func main() {
	fmt.Println("templatedir")

	var targetDir string
	var err error

	if len(os.Args) > 1 {
		targetDir = os.Args[1]
	} else {
		targetDir, err = os.Getwd()
		dieOnErr(err)
	}

	fmt.Println("->	applying to directory ", targetDir)
	fsys := osfs.DirWriteFS(targetDir)

	var args templatedir.Args
	args, err = templatedir.DefaultArgs()
	dieOnErr(err)

	err = templatedir.RenderTo(fsys, fsys, args)
	dieOnErr(err)
}
