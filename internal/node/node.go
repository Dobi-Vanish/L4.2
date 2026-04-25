package node

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	pb "mygrep/grep/proto"
	"mygrep/internal/coordinator"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Node struct {
	pb.UnimplementedGrepServiceServer
	id           int
	coord        *coordinator.Coordinator
	peers        []string
	found        bool
	lines        []string
	stopCh       chan struct{}
	mu           sync.Mutex
	stopNotified bool
}

func NewNode(id int, peers []string, coord *coordinator.Coordinator) *Node {
	return &Node{
		id:     id,
		peers:  peers,
		coord:  coord,
		stopCh: make(chan struct{}),
	}
}

func (n *Node) SetResult(found bool, lines []string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.found = found
	n.lines = lines
}

func (n *Node) ReportResult(ctx context.Context, req *pb.ResultRequest) (*pb.ResultResponse, error) {
	if n.coord != nil {
		n.coord.AddVote(int(req.NodeId), req.Found, req.Lines)
	}
	return &pb.ResultResponse{Ack: true}, nil
}

func (n *Node) BroadcastStop(ctx context.Context, req *pb.StopRequest) (*pb.StopResponse, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if !n.stopNotified {
		close(n.stopCh)
		n.stopNotified = true
	}
	return &pb.StopResponse{}, nil
}

func (n *Node) StopChan() <-chan struct{} {
	return n.stopCh
}

func (n *Node) SendResultToCoordinator() error {
	coordAddr := n.peers[0]
	var conn *grpc.ClientConn
	var err error
	for i := 0; i < 5; i++ {
		conn, err = grpc.Dial(coordAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err == nil {
			break
		}
		log.Printf("node %d: dial attempt %d failed: %v", n.id, i+1, err)
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to coordinator after retries: %w", err)
	}
	defer conn.Close()
	client := pb.NewGrepServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	n.mu.Lock()
	req := &pb.ResultRequest{
		NodeId: int32(n.id),
		Found:  n.found,
		Lines:  n.lines,
	}
	n.mu.Unlock()
	_, err = client.ReportResult(ctx, req)
	if err != nil {
		log.Printf("node %d: ReportResult RPC failed: %v", n.id, err)
	}
	return err
}

func (n *Node) BroadcastStopSignal() {
	if n.coord == nil {
		return
	}
	log.Printf("Node %d: broadcasting stop signal to all peers", n.id)
	for _, addr := range n.peers {
		if addr == n.peers[n.id] {
			continue
		}
		go func(addr string) {
			conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("node %d: cannot dial %s: %v", n.id, addr, err)
				return
			}
			defer conn.Close()
			client := pb.NewGrepServiceClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, err = client.BroadcastStop(ctx, &pb.StopRequest{})
			if err != nil {
				log.Printf("node %d: broadcast stop to %s failed: %v", n.id, addr, err)
			} else {
				log.Printf("node %d: stop signal sent to %s", n.id, addr)
			}
		}(addr)
	}
}
