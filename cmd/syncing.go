package main

import (
	"flag"
	"fmt"
	"os"
	"syncing/receiver"
	"syncing/sender"
)

func main() {
	isServer := flag.Bool("server", false, "is server mode?")
	flag.Parse()

	args := flag.Args()
	if len(args) != 2 {
		flag.Usage()
		return
	}
	//from := flag.Args()[0]
	remoteBasePath := flag.Args()[1]

	if *isServer {
		receiver.RunServer()
		return
	}

	if remoteBasePath == "." {
		remoteBasePath = "/home/darren/201907"
	}

	err := sender.Send("10.81.6.101", "36000", "darren", "/home/darren/syncing/syncing", remoteBasePath)
	//err := sender.Send("192.168.1.105", "22", "darren", "/home/darren/syncing/syncing")
	if err != nil {
		panic(err)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `syncing version: syncing/0.0.1\n  Usage: syncing [OPTION]... SRC DEST`)
	flag.PrintDefaults()
}
