package cluster

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"net"
	"time"
)

/*
* Multicast peer discovery.
*
* Every node periodically transmits information about its cluster name and management address.
* Once a node receives information anout previously-unknown peer, it initiates greeting procedure via HTTP.
*
* Multicast is used only for discovery.
 */

const (
	DiscoveryPingPeriod = 55 * time.Second
)

type DiscoveryPing struct {
	Type        string
	ClusterName string
	NodeName    string
	MgmtAddr    string
}

func (c *Cluster) StartMulticastDiscovery(mcastAddrStr string) {
	mcaddr, err := net.ResolveUDPAddr("udp", mcastAddrStr)
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}

	conn, err := net.ListenMulticastUDP("udp", nil, mcaddr)
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}

	listen := func() {
		for {
			b := make([]byte, 1024)
			n, clientAddr, err := conn.ReadFromUDP(b)
			if err != nil {
				log.Fatalf("ReadFromUDP: %s", err)
			}
			b = b[:n]
			ping := &DiscoveryPing{}
			err = json.Unmarshal(b, ping)
			if err != nil {
				log.Printf("WARN: cannot unmarshal multicast message from %v: %s: `%s`", clientAddr, err, hex.EncodeToString(b))
				continue
			}
			if ping.Type == "ping" && ping.MgmtAddr != "" && ping.NodeName != c.Me.Name {
				ping.MgmtAddr = combineHostAndPort(clientAddr.String(), ping.MgmtAddr)
				if !c.KnownMgmtAdr(ping.MgmtAddr) {
					log.Printf("INFO: multicast discovered new peer: %v", ping)
					c.GreetNode(ping.MgmtAddr, nil, true)
				}
			}
		}
	}

	ping := func(socket net.Conn) {
		ping := &DiscoveryPing{
			Type:        "ping",
			ClusterName: c.Name,
			NodeName:    c.Me.Name,
			MgmtAddr:    c.Me.MgmtAddr,
		}
		jsonPing, err := json.Marshal(ping)
		if err != nil {
			log.Fatalf("cannot serialize ping: %s", err)
		}
		_, err = socket.Write(jsonPing)
		if err != nil {
			log.Printf("ERROR: sending multicast ping: %s", err)
		}
	}

	pingloop := func() {
		socket, err := net.DialUDP("udp", nil, mcaddr)
		if err != nil {
			log.Fatalf("DialUDP: %s", err)
		}
		ping(socket)

		for _ = range time.NewTicker(DiscoveryPingPeriod).C {
			ping(socket)
		}
	}

	go listen()
	go pingloop()
}
