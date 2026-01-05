package htcrawl

import (
	"strings"
	"time"
)

type DOMNode struct {
	Nelements       int
	Simhash         uint32
	LastSeenAt      int64
	SeenCount       int
	TotDomMutations int
}

type DOMDeduplicator struct {
	domNodes              []*DOMNode
	elementsDiffThreshold float64
	simhashDiffThreshold  float64
}

func NewDOMDeduplicator() *DOMDeduplicator {
	return &DOMDeduplicator{
		domNodes:              make([]*DOMNode, 0),
		elementsDiffThreshold: 15.0,
		simhashDiffThreshold:  0.75,
	}
}

func (dd *DOMDeduplicator) newNode(domArray []string, totDomMutations int) *DOMNode {
	domText := make([]string, 0, len(domArray))
	for _, e := range domArray {
		if e == "" {
			continue
		}
		parts := strings.Split(e, " ")
		if len(parts) > 0 {
			name := strings.ToLower(parts[0])
			if len(parts) > 1 && parts[1] != "" {
				name += "-" + strings.ReplaceAll(parts[1], " ", "")
			}
			domText = append(domText, name)
		}
	}

	return &DOMNode{
		Nelements:       len(domArray),
		Simhash:         SimHash(domText),
		LastSeenAt:      time.Now().Unix(),
		SeenCount:       1,
		TotDomMutations: totDomMutations,
	}
}

func (dd *DOMDeduplicator) compare(node1, node2 *DOMNode) bool {
	maxElements := float64(MaxInt(node1.Nelements, node2.Nelements))
	elementsDiff := (AbsInt(node1.Nelements-node2.Nelements) * 100) / int(maxElements)

	if float64(elementsDiff) > dd.elementsDiffThreshold {
		return false
	}

	simhashDiff := Similarity(node1.Simhash, node2.Simhash)
	if simhashDiff < dd.simhashDiffThreshold {
		return false
	}

	return true
}

func (dd *DOMDeduplicator) getNode(node *DOMNode) *DOMNode {
	for _, n := range dd.domNodes {
		if dd.compare(n, node) {
			return n
		}
	}
	return nil
}

func (dd *DOMDeduplicator) Reset() {
	dd.domNodes = make([]*DOMNode, 0)
}

type AddNodeResult struct {
	Added           bool
	LastSeenAt      int64
	SeenCount       int
	TotDomMutations int
}

func (dd *DOMDeduplicator) AddNode(domArray []string, totDomMutations int) *AddNodeResult {
	newNode := dd.newNode(domArray, totDomMutations)
	existingNode := dd.getNode(newNode)

	if existingNode == nil {
		dd.domNodes = append(dd.domNodes, newNode)
		return &AddNodeResult{
			Added: true,
		}
	}

	existingNode.LastSeenAt = time.Now().Unix()
	existingNode.SeenCount++

	result := &AddNodeResult{
		Added:           false,
		LastSeenAt:      existingNode.LastSeenAt,
		SeenCount:       existingNode.SeenCount,
		TotDomMutations: existingNode.TotDomMutations,
	}

	return result
}

func (dd *DOMDeduplicator) GetNodeCount() int {
	return len(dd.domNodes)
}

func (dd *DOMDeduplicator) GetTotalSeenCount() int {
	total := 0
	for _, node := range dd.domNodes {
		total += node.SeenCount
	}
	return total
}

func AbsInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
