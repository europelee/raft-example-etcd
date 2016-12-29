package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strings"

	"time"

	"github.com/coreos/etcd/raft/raftpb"
)

var (
	peersStr   = flag.String("peers", "http://10.77.9.48:9900,http://10.77.9.49:9901,http://10.77.9.50:9902", "raft peers")
	id         = flag.Int("id", 1, "peer id")
	qps        = flag.Int("qps", 500, "proposals per second")
	tickMs     = flag.Uint("heartbeat-interval", 100, "time (in milliseconds) of a heartbeat interval")
	electionMs = flag.Uint("election-timeout", 1000, "time (in milliseconds) for an election to timeout")
)

var (
	data = make([]byte, 6144)
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	flag.Parse()

	for i := 0; i < 6144; i++ {
		data[i] = byte(rand.Intn(255))
	}

	peers := strings.Split(*peersStr, ",")
	proposeC := make(chan []byte, len(peers))
	confChangeC := make(chan raftpb.ConfChange, len(peers))

	commitC, errorC, leadership, rc := newRaftNode(*id, peers, false, *tickMs, *electionMs, proposeC, confChangeC)

	//处理错误
	go func() {
		for {
			select {
			case err := <-errorC:
				log.Printf("Error: %v\n", err)
			}
		}
	}()

	go func() {
		for {
			select {
			case isLeader := <-leadership:
				log.Printf("current peer is leader?: %t", isLeader)
			}
		}

	}()

	go func() {
		sms := 1e6 / *qps
		ticker := time.NewTicker(time.Duration(sms) * time.Microsecond)
		for range ticker.C {
			if rc.hasLeader == 1 {
				proposeC <- data
			}
		}
	}()

	//处理commit
	go func() {
		for c := range commitC {
			if c != nil {
				fmt.Printf("commit\n")
			} else {
				fmt.Printf("replay完成\n")
			}

		}
	}()

	doneC := make(chan struct{}, 1)
	<-doneC
}
