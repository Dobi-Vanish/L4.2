package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	pb "mygrep/grep/proto"
	"mygrep/internal/coordinator"
	"mygrep/internal/node"
	"mygrep/internal/searcher"

	"google.golang.org/grpc"
)

func main() {
	var (
		id          = flag.Int("id", 0, "node id (0..N-1)")
		peers       = flag.String("peers", "", "comma-separated list of host:port")
		file        = flag.String("file", "", "file to search")
		pattern     = flag.String("pattern", "", "regex pattern")
		ignoreCase  = flag.Bool("i", false, "ignore case")
		lineNumbers = flag.Bool("n", false, "show line numbers")
		invertMatch = flag.Bool("v", false, "invert match (show non-matching lines)")
	)
	flag.Parse()

	if *file == "" || *pattern == "" || *peers == "" {
		log.Fatal("Usage: mygrep -id=0 -peers=addr1,addr2,... -file=path -pattern=regex [-i] [-n] [-v]")
	}
	peerList := strings.Split(*peers, ",")
	totalNodes := len(peerList)
	if *id < 0 || *id >= totalNodes {
		log.Fatalf("id must be in [0, %d]", totalNodes-1)
	}

	log.Printf("Node %d starting, total nodes = %d", *id, totalNodes)

	searcherObj, err := searcher.New(*pattern, *ignoreCase, *invertMatch, *lineNumbers)
	if err != nil {
		log.Fatal(err)
	}
	lines, found, err := searcherObj.SearchLinesInFile(*file, *id, totalNodes)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Node %d: found=%v, lines=%d", *id, found, len(lines))

	var coord *coordinator.Coordinator
	if *id == 0 {
		coord = coordinator.New(totalNodes)
		log.Printf("Node %d is coordinator", *id)
	}

	nodeInstance := node.NewNode(*id, peerList, coord)
	nodeInstance.SetResult(found, lines)

	grpcServer := grpc.NewServer()
	pb.RegisterGrepServiceServer(grpcServer, nodeInstance)

	lis, err := net.Listen("tcp", peerList[*id])
	if err != nil {
		log.Fatalf("Node %d: listen error: %v", *id, err)
	}
	go grpcServer.Serve(lis)
	log.Printf("Node %d: gRPC server listening on %s", *id, peerList[*id])

	if *id != 0 {
		log.Printf("Node %d: sending result to coordinator", *id)
		if err := nodeInstance.SendResultToCoordinator(); err != nil {
			log.Printf("Node %d: failed to send result: %v", *id, err)
		} else {
			log.Printf("Node %d: result sent successfully", *id)
		}
	} else {
		log.Printf("Node %d: coordinator voting for itself", *id)
		coord.AddVote(*id, found, lines)
	}

	if *id == 0 {
		log.Printf("Node %d: waiting for quorum...", *id)
		<-coord.StopSignal()
		log.Printf("Node %d: stop signal received, collecting all lines and exiting", *id)
		allLines := coord.GetAllLines()
		for _, line := range allLines {
			fmt.Print(line)
		}
		time.Sleep(100 * time.Millisecond)
		nodeInstance.BroadcastStopSignal()
		time.Sleep(200 * time.Millisecond)
	} else {
		log.Printf("Node %d: waiting for stop signal from coordinator", *id)
		<-nodeInstance.StopChan()
		log.Printf("Node %d: received stop signal", *id)
	}
	log.Printf("Node %d: exiting", *id)
}
