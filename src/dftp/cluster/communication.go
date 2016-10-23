package cluster

import (
	"bytes"
	"dftp/dfsfat"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (c *Cluster) GreetNode(addr string, node *NodeInfo, requestFullUpdate bool) {
	log.Printf("Greeting %s (%s)...", node.GetName(), addr)
	vals := url.Values{}
	vals.Set("name", c.Me.Name)
	vals.Set("public-addr", c.Me.PublicAddr)
	vals.Set("mgmt-addr", c.Me.MgmtAddr)
	if requestFullUpdate {
		vals.Set("request-full-update", "true")
	}

	r, err := c.client.PostForm(fmt.Sprintf("http://%s/cluster/", addr), vals)
	if err != nil {
		log.Printf("Error communicating with %s: %s", addr, err)
		return
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		s, _ := ioutil.ReadAll(r.Body)
		log.Printf("Error communicating with %s: HTTP status %d (%s)", addr, r.StatusCode, string(s))
		return
	}
	decoder := json.NewDecoder(r.Body)

	rInfo := PublicClusterInfo{}
	err = decoder.Decode(&rInfo)
	if err != nil {
		log.Printf("Error decoding cluster info from %s: %s", addr, err)
		return
	}

	rInfo.Me.LastAlive = time.Now().Unix()
	rInfo.Me.MgmtAddr = addr
	rInfo.Me.PublicAddr = combineHostAndPort(addr, rInfo.Me.PublicAddr)
	rInfo.Me.GreetState = StateDone
	rInfo.Peers[rInfo.Me.Name] = rInfo.Me
	for _, p := range rInfo.Peers {
		c.UpdateNode(p)
	}
}

func (c *Cluster) UpdateNode(newinfo *NodeInfo) {
	if newinfo.Name == c.Me.Name {
		return
	}
	c.Lock()
	node, ok := c.Peers[newinfo.Name]
	if !ok {
		log.Printf("Met new node: %s", newinfo.Name)
		node = &NodeInfo{
			Name: newinfo.Name,
		}
		c.Peers[newinfo.Name] = node
	}
	c.Unlock()

	node.Lock()
	node.PublicAddr = newinfo.PublicAddr
	node.MgmtAddr = newinfo.MgmtAddr
	node.LastAlive = newinfo.LastAlive
	if newinfo.GreetState == StateDone {
		node.GreetState = newinfo.GreetState
	}
	node.Unlock()

	if node.GreetState == StateNever {
		c.ScheduleGreet(node)
	}
	if node.PushState == StateNever {
		c.SchedulePush(node)
	}

}

func (c *Cluster) ScheduleGreet(node *NodeInfo) {
	node.Lock()
	defer node.Unlock()
	if node.GreetState != StateNever {
		return
	}
	node.GreetState = StatePending
	go c.GreetNode(node.MgmtAddr, node, true)
}

func (c *Cluster) SchedulePush(node *NodeInfo) {
	node.Lock()
	defer node.Unlock()
	if node.PushState == StatePending {
		return
	}
	node.PushState = StatePending
	go c.PushFullUpdate(node)
}

type UpdateData struct {
	Files          []*dfsfat.FileAnnouncement
	UpdateTime     int64
	Full           bool
	SenderNodeName string
}

func (c *Cluster) PushFullUpdate(node *NodeInfo) {
	log.Printf("Pushing full update to %s...", node.Name)

	var buf bytes.Buffer
	var err error
	var r *http.Response
	enc := json.NewEncoder(&buf)

	files, scanT := c.LocalFs.GetLastFullScan()
	upd := &UpdateData{
		Files:          files,
		UpdateTime:     scanT,
		Full:           true,
		SenderNodeName: c.Me.Name,
	}
	err = enc.Encode(upd)
	if err != nil {
		goto fail
	}
	r, err = c.client.Post(fmt.Sprintf("http://%s/update/", node.MgmtAddr), "application/json", &buf)
	if err != nil {
		goto fail
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		s, _ := ioutil.ReadAll(r.Body)
		err = fmt.Errorf("HTTP status %d (%s)", r.StatusCode, string(s))
		goto fail
	}
	log.Printf("Pushed full update to %s", node.Name)

	node.Lock()
	node.PushState = StateDone
	node.LastUpdatePushed = time.Now().Unix()
	node.Unlock()
	return
fail:
	log.Printf("Error pushing last update to %s: %s", node.Name, err)
	node.Lock()
	node.PushState = StateNever
	node.Unlock()
}

func (c *Cluster) ReceiveUpdate(upd *UpdateData) {
	log.Printf("Received update (files: %d, full: %v) from %s", len(upd.Files), upd.Full, upd.SenderNodeName)
	c.RLock()
	node, ok := c.Peers[upd.SenderNodeName]
	c.RUnlock()
	if !ok {
		log.Printf("Warning: unknown update sender: %s", upd.SenderNodeName)
		return
	}
	c.DfsRoot.Update(upd.Files)
	// TODO: if full update, remove older files owned by upd.SenderNodeName
	node.Lock()
	node.LastUpdateReceived = upd.UpdateTime
	node.LastFullUpdateReceived = upd.UpdateTime
	node.Unlock()
}

// returns <addr1.host>:<addr2.port>
func combineHostAndPort(addr1, addr2 string) string {
	parts1 := strings.Split(addr1, ":")
	parts2 := strings.Split(addr2, ":")
	s := parts1[0] + ":" + parts2[1]
	return s
}
