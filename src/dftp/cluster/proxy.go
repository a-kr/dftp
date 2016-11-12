package cluster

import (
	"dftp/dfsfat"
	"dftp/localfs"
	"fmt"
	"io"
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
	client       http.Client
}

const (
	MaxRedirectDepth = 2
)

var (
	TooManyRedirectsError = fmt.Errorf("too many proxy redirects while serving the file")
	UnknownNodeError      = fmt.Errorf("file resides on unknown node")
)

// Open file for reading.
// The caller must Close() the returned file afterwards.
func (p *Proxy) OpenRead(path string, entry *dfsfat.TreeNodeReadonly, nRedirects int) (io.ReadCloser, error) {
	if entry.OwnerNode == p.Cluster.LocalFs.MyNodeName {
		f, err := p.Cluster.LocalFs.OpenRead(path)
		if err != nil {
			return nil, err
		}
		return f, nil
	}

	if nRedirects >= MaxRedirectDepth {
		return nil, TooManyRedirectsError
	}
	nRedirects += 1

	p.Cluster.RLock()
	node, ok := p.Cluster.Peers[entry.OwnerNode]
	p.Cluster.RUnlock()
	if !ok {
		return nil, UnknownNodeError
	}

	url := fmt.Sprintf("http://%s/fs/%s?redirN=%d", node.PublicAddr, path, nRedirects)
	resp, err := p.client.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("proxy error: %s", resp.Status)
	}
	return resp.Body, nil
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
