package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"syncing/receiver"
	"syncing/sender"
)

func main() {
	isServer := flag.Bool("server", false, "is server mode?")
	var param sender.Params
	flag.IntVar(&param.Step, "step", 100, "step size")
	flag.StringVar(&param.ExecPath, "exec", "~/syncing/syncing", "remote exec path")
	flag.IntVar(&param.Port, "port", 36000, "server port")
	flag.BoolVar(&param.Delete, "delete", false, "delete remote files?")
	flag.Parse()

	if *isServer {
		receiver.RunServer()
		return
	}

	args := flag.Args()
	if len(args) != 2 {
		flag.Usage()
		return
	}
	param.LocalBasePath = flag.Args()[0]
	param.RemoteBasePath = flag.Args()[1]

	if !strings.HasSuffix(param.LocalBasePath, "/") {
		param.LocalBasePath += "/"
	}
	if param.LocalBasePath == "." {
		param.LocalBasePath = "./"
	}

	err := sender.Start(&param)
	if err != nil {
		panic(err)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `syncing version: syncing/0.0.1\n  Usage: syncing [OPTION]... SRC DEST`)
	flag.PrintDefaults()
}
