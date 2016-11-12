package main

import (
	"dftp/cluster"
	"dftp/dfsfat"
	"dftp/ftpface"
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
	optFtpAddr       = flag.String("ftp-listen", ":2121", "host:port for public FTP interface to listen on")
	optHttpMgmtAddr  = flag.String("http-mgmt-listen", ":7041", "host:port for private cluster management HTTP interface to listen on")
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

	cluster := cluster.New(dfs, localfs, *optHttpAddr, *optHttpMgmtAddr)
	go cluster.ServeHttp(*optHttpMgmtAddr)

	if *optHttpAddr != "" {
		server := httpface.Server{
			DfsRoot: dfs,
			Cluster: cluster,
		}
		go server.ServeHttp(*optHttpAddr)
	}

	if *optFtpAddr != "" {
		server := ftpface.Server{
			DfsRoot: dfs,
			Cluster: cluster,
		}
		go server.ServeFtp(*optFtpAddr)
	}

	for {
		time.Sleep(1 * time.Second)
	}
}
