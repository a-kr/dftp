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

	Proxy *Proxy

	client *http.Client

	mux *http.ServeMux
}

type PublicClusterInfo struct {
	Name string
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

func New(dfs *dfsfat.TreeNode, localfs *localfs.LocalFs, clusterName string, publicAddr string, mgmtAddr string, multicastAddr string) *Cluster {
	c := &Cluster{}
	c.Name = clusterName
	c.DfsRoot = dfs
	c.LocalFs = localfs
	c.Proxy = NewProxy(c, localfs)
	c.Peers = make(map[string]*NodeInfo)
	c.Me = &NodeInfo{
		Name:       localfs.MyNodeName,
		PublicAddr: publicAddr,
		MgmtAddr:   mgmtAddr,
		LastAlive:  time.Now().Unix(),
	}
	c.client = httputils.MakeTimeoutingHttpClient(10 * time.Second)
	if multicastAddr != "" {
		c.StartMulticastDiscovery(multicastAddr)
	}
	return c
}

func (c *Cluster) KnownMgmtAdr(addr string) bool {
	c.RLock()
	defer c.RUnlock()
	if c.Me.MgmtAddr == addr {
		return true
	}
	for _, p := range c.Peers {
		if p.MgmtAddr == addr {
			return true
		}
	}
	return false
}
