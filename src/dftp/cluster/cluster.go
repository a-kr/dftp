package cluster

import (
	"dftp/dfsfat"
	"dftp/httputils"
	"dftp/localfs"
	"net/http"
	"sync"
	"time"
)

/*
* Cluster represents a collection of peers which this node exchanges updates with.
 */

type Cluster struct {
	sync.RWMutex
	PublicClusterInfo
	DfsRoot *dfsfat.TreeNode
	LocalFs *localfs.LocalFs

	client *http.Client

	mux *http.ServeMux
}

type PublicClusterInfo struct {
	Me    *NodeInfo
	Peers map[string]*NodeInfo
}

const (
	StateNever = iota
	StatePending
	StateDone
)

type NodeInfo struct {
	sync.Mutex
	Name                   string
	PublicAddr             string
	MgmtAddr               string
	LastAlive              int64
	LastUpdatePushed       int64
	LastUpdateReceived     int64
	LastFullUpdateReceived int64
	GreetState             int `json:"-"`
	PushState              int `json:"-"`
}

func (n *NodeInfo) GetName() string {
	if n == nil {
		return "<nil>"
	}
	return n.Name
}

func New(dfs *dfsfat.TreeNode, localfs *localfs.LocalFs, publicAddr string, mgmtAddr string) *Cluster {
	c := &Cluster{}
	c.DfsRoot = dfs
	c.LocalFs = localfs
	c.Peers = make(map[string]*NodeInfo)
	c.Me = &NodeInfo{
		Name:       localfs.MyNodeName,
		PublicAddr: publicAddr,
		MgmtAddr:   mgmtAddr,
		LastAlive:  time.Now().Unix(),
	}
	c.client = httputils.MakeTimeoutingHttpClient(3 * time.Second)
	return c
}
