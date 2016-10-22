package main

import (
	"dftp/dfsfat"
	"dftp/httpface"
	"dftp/localfs"
	"flag"
	"log"
	"os"
	"time"
)

var (
	optDfsRoot       = flag.String("dfsroot", "", "local directory corresponding to local DFS root")
	optDfsMountPoint = flag.String("dfsmount", "", "path inside DFS where local tree will be mounted (not necessarily unique path)")
	optMyNodeName    = flag.String("node-name", "", "node name to use instead of hostname")
	optHttpAddr      = flag.String("http-listen", ":7040", "host:port for public HTTP interface to listen on")
)

func main() {
	flag.Parse()

	myNodeName, err := os.Hostname()
	if err != nil {
		myNodeName = *optMyNodeName
		if myNodeName == "" {
			log.Fatalf("FATAL: node name not known (set hostname, or specify --node-name)")
		}
	}

	if *optDfsRoot == "" {
		log.Fatalf("FATAL: specify --dfsroot")
	}

	dfs := dfsfat.NewRootNode()
	localfs := localfs.NewLocalFs(*optDfsRoot, *optDfsMountPoint, dfs, myNodeName)
	localfs.ScanOnce()
	if *optHttpAddr != "" {
		server := httpface.Server{
			DfsRoot: dfs,
			LocalFs: localfs,
		}
		go server.ServeHttp(*optHttpAddr)
	}

	/*
	dfs.Walk(func(path string, info os.FileInfo, _ error) error {
		log.Printf("%s: %v", path, info.Name())
		return nil
	})
	*/

	for {
		time.Sleep(1 * time.Second)
	}
}
