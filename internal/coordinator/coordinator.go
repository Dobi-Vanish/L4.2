package coordinator

import (
	"log"
	"sync"
	"time"
)

type Coordinator struct {
	mu          sync.Mutex
	votes       map[int]bool
	totalNodes  int
	votesNeeded int
	stopCh      chan struct{}
	allLines    []string
	quorumDone  bool
}

func New(totalNodes int) *Coordinator {
	return &Coordinator{
		votes:       make(map[int]bool),
		totalNodes:  totalNodes,
		votesNeeded: totalNodes/2 + 1,
		stopCh:      make(chan struct{}),
		allLines:    []string{},
		quorumDone:  false,
	}
}

func (c *Coordinator) AddVote(nodeID int, found bool, lines []string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.votes[nodeID]; exists {
		return false
	}
	c.votes[nodeID] = found
	c.allLines = append(c.allLines, lines...)

	log.Printf("Coordinator: vote from node %d, found=%v, total votes now %d (need %d)",
		nodeID, found, len(c.votes), c.votesNeeded)

	if len(c.votes) >= c.votesNeeded && !c.quorumDone {
		c.quorumDone = true
		log.Println("Coordinator: quorum reached, waiting 500ms for late results...")
		go func() {
			time.Sleep(500 * time.Millisecond)
			c.mu.Lock()
			defer c.mu.Unlock()
			close(c.stopCh)
		}()
		return true
	}
	return false
}

func (c *Coordinator) StopSignal() <-chan struct{} {
	return c.stopCh
}

func (c *Coordinator) GetAllLines() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.allLines
}
