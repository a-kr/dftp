package cluster

import (
	"dftp/localfs"
	"log"
	"net/http"
	"net/http/httputil"
	"sync"
)

// Transparent handling of local or remote file operations
type Proxy struct {
	Cluster      *Cluster
	LocalFs      *localfs.LocalFs
	proxiesMutex sync.Mutex
	proxies      map[string]*httputil.ReverseProxy
}

func NewProxy(cluster *Cluster, localfs *localfs.LocalFs) *Proxy {
	p := &Proxy{
		Cluster: cluster,
		LocalFs: localfs,
		proxies: make(map[string]*httputil.ReverseProxy),
	}
	return p
}

func (p *Proxy) GetHttpProxy(nodeName string) *httputil.ReverseProxy {
	p.proxiesMutex.Lock()
	defer p.proxiesMutex.Unlock()
	proxy, ok := p.proxies[nodeName]
	if !ok {
		p.Cluster.RLock()
		node, ok := p.Cluster.Peers[nodeName]
		p.Cluster.RUnlock()
		if !ok {
			return nil
		}

		proxy = &httputil.ReverseProxy{}
		proxy.Director = func(r *http.Request) {
			r.URL.Scheme = "http"
			r.URL.Host = node.PublicAddr
			log.Printf("Proxy download request to %s", r.URL)
		}
		p.proxies[nodeName] = proxy

	}
	return proxy
}
