package main

import (
	"flag"
	"fmt"
	"os"
	"syncing/receiver"
	"syncing/sender"
)

func main() {
	cwd, _ := os.Getwd()
	println(cwd)

	isServer := flag.Bool("server", false, "is server mode?")
	step := flag.Uint("step", 500, "step")
	lpath := flag.String("lpath", "./", "local path")
	rpath := flag.String("rpath", "", "remote path")
	execPath := flag.String("exec", "~/syncing/syncing", "exec path")
	flag.Parse()

	if *isServer {
		receiver.RunServer()
		return
	}

	if *rpath == "" {
		usage()
		return
	}
	if *lpath == "." {
		*lpath = "./"
	}

	// args := flag.Args()
	// if len(args) != 2 {
	// 	flag.Usage()
	// 	return
	// }
	// //from := flag.Args()[0]
	// remoteBasePath := flag.Args()[1]
	// p2 := os.Args[2]
	// fmt.Printf("remoteBasePath: %s\n", remoteBasePath)
	// fmt.Printf("p2: %s\n", p2)

	// if remoteBasePath == "." {
	// 	remoteBasePath = "/home/darren/201907"
	// }

	err := sender.Start("10.81.6.101", "36000", "darren", *execPath, *lpath, *rpath, int(*step))
	//err := sender.Start("192.168.1.105", "22", "darren", "/home/darren/syncing/syncing", remoteBasePath)
	if err != nil {
		panic(err)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `syncing version: syncing/0.0.1\n  Usage: syncing [OPTION]... SRC DEST`)
	flag.PrintDefaults()
}
