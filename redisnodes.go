package main

import (
	"flag"
	"fmt"
	"github.com/go-redis/redis"
	"net"
	"os"
	"sort"
	"strings"
)

type Node struct {
	addr string
	host string
	master string
	mode string
	nodeId string
	slaves []string
	slots string
}

func NewNode(nodeId string, addr string, mode string, master string, slots string) *Node {
	n := new(Node)
	n.nodeId = nodeId
	n.addr = addr
	n.mode = mode
	if mode == "master" {
		n.slots = slots
	} else if mode == "slave" {
		n.master = master
	}

	portPos := strings.LastIndex(addr, ":")
	hostName := addr
	if portPos > 0 {
		hostName = hostName[:portPos]
		hostNames, err := net.LookupAddr(hostName)
		if err == nil {
			hostName = hostNames[0]
		}
	}
	n.host = hostName

	return n
}

type ByMaster [][]string

func (a ByMaster) Len() int {
	return len(a)
}

func (a ByMaster) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByMaster) Less(i, j int) bool {
	if a[i][2] == a[j][2] {
		return false
	}
	if a[i][2] == "master" && a[j][2] != "master" {
		return true
	}
	return false
}

func main() {

	server := flag.String("server", ":6379", "A server")
	flag.Parse()

	client := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs: []string{*server},
	})
	defer client.Close()

	val, err := client.ClusterNodes().Result()
	if err != nil {
		fmt.Println(err)
		flag.Usage()
		os.Exit(1)
	}
	rawlist := strings.Split(val, "\n")
	var nodelist [][]string

	// Remove "myself" from the list if its in there.
	for _, v := range rawlist {
		nodeinfo := strings.Split(v, " ")
		if len(nodeinfo) < 3 {
			break
		}
		nodeinfo[2] = strings.Replace(nodeinfo[2], "myself,", "", 1)
		nodelist = append(nodelist, nodeinfo)
	}

	// Sort masters to the top of the list.
	sort.Sort(ByMaster(nodelist))

	nodes := make(map[string]*Node)
	// Get our master count, and build our Node tree.
	mastercount := 0
	for _, v := range nodelist {
		slots := ""
		if len(v) >= 9 {
			slots = v[8]
		}
		node := NewNode(v[0], v[1], v[2], v[3], slots)
		nodes[node.nodeId] = node

		if node.mode == "slave" {
			nodes[node.master].slaves = append(nodes[node.master].slaves, node.nodeId)
		} else if node.mode == "master" {
			mastercount++
		}
	}

	fork := string('├');
	last := string('└');
	row := string('─');
	// col := '│';
	// tee := '┬';

	mastersprinted := 0
	for _, n := range nodes {
		if n.mode != "master" {
			continue
		}
		mastersprinted++

		lastmaster := mastersprinted == mastercount
		slavecount := len(n.slaves)
		slavesprinted := 0

		fmt.Printf("%v %v %v\n", n.nodeId, n.host, n.slots)

		for _, slaveid := range n.slaves {
			slavesprinted++

			rightchar := fork
			if slavesprinted == slavecount {
				rightchar = last
			}

			slave := nodes[slaveid]
			fmt.Printf("%v%v %v %v\n", rightchar, row, slave.nodeId, slave.host)
		}
		if !lastmaster {
			fmt.Println()
		}
	}
}