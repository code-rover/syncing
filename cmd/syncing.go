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
	// args := flag.Args()
	// if len(args) != 2 {
	// 	return
	// 	//flag.Usage()
	// }
	// from := flag.Args()[0]
	// to := flag.Args()[1]

	flag.Parse()

	if *isServer {
		receiver.RunServer()
		return
	}

	err := sender.Send("10.81.6.101", "36000", "darren", "/home/darren/syncing/syncing")
	if err != nil {
		panic(err)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `syncing version: syncing/0.0.1\n  Usage: syncing [OPTION]... SRC DEST`)
	flag.PrintDefaults()
}
