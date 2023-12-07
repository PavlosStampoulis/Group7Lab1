package dlab

import (
	"crypto/sha1"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/rpc"
)

// Global variableto assign what comes after -r, to later compare the successors lists and see if hit the max capacity or not
var globalNumberSuccessors int64

func hashString(elt string) *big.Int {
	hasher := sha1.New()
	hasher.Write([]byte(elt))
	return new(big.Int).SetBytes(hasher.Sum(nil))
}

// start a thread that listens for RPCs from other nodes
func (n *Node) Server() {
	rpc.Register(n)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", string(n.address))

	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)

}

func getLocalAddress() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}

func between(startN NodeAddress, eltN NodeAddress, end *big.Int, inclusive bool) bool {
	start := hashString(string(startN))
	elt := hashString(string(eltN))
	if end.Cmp(start) > 0 {
		return (start.Cmp(elt) < 0 && elt.Cmp(end) < 0) || (inclusive && elt.Cmp(end) == 0)
	} else {
		return start.Cmp(elt) < 0 || elt.Cmp(end) < 0 || (inclusive && elt.Cmp(end) == 0)
	}
}
