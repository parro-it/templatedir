package main

import (
	"fmt"
	"os"

	"github.com/parro-it/vs/osfs"

	"github.com/parro-it/templatedir"
)

type errorChecker struct {
	err error
}

func (checker errorChecker) dieOnErr() {
	if checker.err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s\n\n", checker.err.Error())
		os.Exit(1)
	}
}

func main() {
	fmt.Println("templatedir")

	var targetDir string
	var check errorChecker

	if len(os.Args) > 1 {
		targetDir = os.Args[1]
	} else {
		targetDir, check.err = os.Getwd()
		check.dieOnErr()
	}

	fmt.Println("->	applying to directory ", targetDir)
	fsys := osfs.DirWriteFS(targetDir)
	check.err = templatedir.RenderTo(fsys, fsys)
	check.dieOnErr()
}
