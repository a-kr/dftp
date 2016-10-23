package cluster

import (
	"dftp/httputil"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

/*
* Management HTTP interface used for peer-to-peer communications.
 */

func (c *Cluster) ServeHttp(addr string) {
	c.mux = http.NewServeMux()
	httputil.HandleFunc(c.mux, "/", c.HttpIndex)
	httputil.HandleFunc(c.mux, "/cluster/", c.HttpCluster)
	httputil.HandleFunc(c.mux, "/join/", c.HttpJoin)
	httputil.HandleFunc(c.mux, "/update/", c.HttpUpdate)
	log.Printf("HTTP mgmt interface listening on %s...", addr)
	if err := http.ListenAndServe(addr, c.mux); err != nil {
		log.Fatalf("http: %s", err)
	}
}

func (c *Cluster) HttpIndex(w http.ResponseWriter, r *http.Request) {
	http.Error(w, `Hi!
		* GET /cluster/  to list peers
		* POST /join/?peer=ip:port  to initiate cluster membership
	`, 404)
}

// GET /cluster/:  get list of peers
// POST /cluster/:  greeting: ask peer to update information about client node
func (c *Cluster) HttpCluster(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		r.ParseForm()
		info := &NodeInfo{}
		info.Name = r.FormValue("name")
		info.PublicAddr = r.FormValue("public-addr")
		info.MgmtAddr = r.FormValue("mgmt-addr")
		if info.Name == "" || info.PublicAddr == "" || info.MgmtAddr == "" {
			http.Error(w, "name, public-addr and mgmt-addr are required parameters", http.StatusBadRequest)
			return
		}
		// TODO: validation: PublicAddr, MgmtAddr must be in form <host>:<port> or :<port>
		info.PublicAddr = combineHostAndPort(r.RemoteAddr, info.PublicAddr)
		info.MgmtAddr = combineHostAndPort(r.RemoteAddr, info.MgmtAddr)
		c.UpdateNode(info)
		if r.FormValue("request-full-update") == "true" {
			c.RLock()
			node, ok := c.Peers[info.Name]
			c.RUnlock()
			if ok {
				c.SchedulePush(node)
			}
		}
	}
	c.httpClusterInfoResponse(w, r)
}

func (c *Cluster) httpClusterInfoResponse(w http.ResponseWriter, r *http.Request) {
	c.RLock()
	defer c.RUnlock()
	enc := json.NewEncoder(w)
	err := enc.Encode(c.PublicClusterInfo)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}

// POST /join/: initiate joining a cluster
func (c *Cluster) HttpJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, `Use POST /join/?peer=ip:port`, http.StatusMethodNotAllowed)
		return
	}
	r.ParseForm()
	if len(r.Form["peer"]) == 0 {
		http.Error(w, `Specify at least one 'peer'`, http.StatusBadRequest)
		return
	}
	for _, peerAddr := range r.Form["peer"] {
		c.GreetNode(peerAddr, nil, true)
	}
	c.httpClusterInfoResponse(w, r)
}

// POST /update/: ask peer to update list of files
func (c *Cluster) HttpUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, `Use POST /update/`, http.StatusMethodNotAllowed)
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, `Content-Type must be application/json`, http.StatusBadRequest)
		return
	}
	dec := json.NewDecoder(r.Body)
	upd := UpdateData{}
	err := dec.Decode(&upd)
	if err != nil {
		http.Error(w, fmt.Sprintf(`Error decoding json: %s`, err), http.StatusBadRequest)
		return
	}
	go c.ReceiveUpdate(&upd)
	http.Error(w, "ok", http.StatusOK)
}
