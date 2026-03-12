//go:build evm

package benchmark

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

const maxBarWidth = 40

// minTracesForFlowchart filters out root span types with too few traces
// to produce meaningful charts (orphan spans from sampling).
const minTracesForFlowchart = 10

// spanNode is a tree node used to build the span hierarchy.
type spanNode struct {
	span     richSpan
	children []*spanNode
}

// printFlowcharts renders a single combined ASCII tree showing the longest
// trace for each distinct root span type. Bars are scaled relative to the
// longest root so durations are visually comparable.
func printFlowcharts(t testing.TB, spans []richSpan) {
	t.Helper()
	if len(spans) == 0 {
		return
	}

	byTrace := groupByTrace(spans)
	byRoot := groupTracesByRoot(byTrace)

	rootNames := make([]string, 0, len(byRoot))
	for name := range byRoot {
		rootNames = append(rootNames, name)
	}
	sort.Strings(rootNames)

	type rootEntry struct {
		node *spanNode
		name string
	}

	var roots []rootEntry
	var maxDur time.Duration
	for _, rootName := range rootNames {
		traces := byRoot[rootName]
		if len(traces) < minTracesForFlowchart {
			continue
		}
		root, _ := longestRootTrace(traces)
		if root == nil {
			continue
		}
		roots = append(roots, rootEntry{node: root, name: rootName})
		if root.span.duration > maxDur {
			maxDur = root.span.duration
		}
	}
	if len(roots) == 0 {
		return
	}

	var b strings.Builder
	fmt.Fprintf(&b, "\n--- Flowchart (longest trace per operation) ---\n")
	for i, r := range roots {
		b.WriteString("\n")
		renderTree(&b, r.node, "", true, maxDur)
		if i < len(roots)-1 {
			b.WriteString("\n")
		}
	}
	t.Log(b.String())
}

// printAggregateFlowcharts builds a single combined ASCII tree showing
// average durations for each distinct root span type.
func printAggregateFlowcharts(t testing.TB, spans []richSpan) {
	t.Helper()
	if len(spans) == 0 {
		return
	}

	byTrace := groupByTrace(spans)
	byRoot := groupTracesByRoot(byTrace)
	hosts := uniqueHosts(spans)

	rootNames := make([]string, 0, len(byRoot))
	for name := range byRoot {
		rootNames = append(rootNames, name)
	}
	sort.Strings(rootNames)

	type aggEntry struct {
		node   *aggregateNode
		traces int
	}

	var aggs []aggEntry
	var maxAvg time.Duration
	for _, rootName := range rootNames {
		traces := byRoot[rootName]
		if len(traces) < minTracesForFlowchart {
			continue
		}
		agg := buildAggregateTree(traces)
		if agg == nil {
			continue
		}
		aggs = append(aggs, aggEntry{node: agg, traces: len(traces)})
		if agg.avgDuration > maxAvg {
			maxAvg = agg.avgDuration
		}
	}
	if len(aggs) == 0 {
		return
	}

	var b strings.Builder
	hostStr := strings.Join(hosts, ", ")
	if hostStr == "" {
		hostStr = "local"
	}
	fmt.Fprintf(&b, "\n--- Average Pipeline (hosts: %s) ---\n", hostStr)
	for i, a := range aggs {
		b.WriteString("\n")
		renderAggregateTree(&b, a.node, "", true, maxAvg)
		if i < len(aggs)-1 {
			b.WriteString("\n")
		}
	}
	t.Log(b.String())
}

// groupTracesByRoot groups traces by their root span name.
func groupTracesByRoot(byTrace map[string][]richSpan) map[string]map[string][]richSpan {
	result := make(map[string]map[string][]richSpan)
	for traceID, spans := range byTrace {
		root := buildTree(spans)
		if root == nil {
			continue
		}
		rootName := root.span.name
		if result[rootName] == nil {
			result[rootName] = make(map[string][]richSpan)
		}
		result[rootName][traceID] = spans
	}
	return result
}

func groupByTrace(spans []richSpan) map[string][]richSpan {
	m := make(map[string][]richSpan)
	for _, s := range spans {
		if s.traceID != "" {
			m[s.traceID] = append(m[s.traceID], s)
		}
	}
	return m
}

// longestRootTrace finds the trace whose root span has the longest duration
// and returns the built tree.
func longestRootTrace(byTrace map[string][]richSpan) (*spanNode, string) {
	var bestRoot *spanNode
	var bestTraceID string
	var bestDur time.Duration

	for traceID, spans := range byTrace {
		root := buildTree(spans)
		if root == nil {
			continue
		}
		if root.span.duration > bestDur {
			bestDur = root.span.duration
			bestRoot = root
			bestTraceID = traceID
		}
	}
	return bestRoot, bestTraceID
}

func buildTree(spans []richSpan) *spanNode {
	byID := make(map[string]*spanNode, len(spans))
	idSet := make(map[string]bool, len(spans))
	for i := range spans {
		node := &spanNode{span: spans[i]}
		byID[spans[i].spanID] = node
		idSet[spans[i].spanID] = true
	}

	var root *spanNode
	for _, node := range byID {
		parent, hasParent := byID[node.span.parentSpanID]
		if hasParent {
			parent.children = append(parent.children, node)
		}
		// root: no parent span ID or parent not in the set
		if node.span.parentSpanID == "" || !idSet[node.span.parentSpanID] {
			if root == nil || node.span.duration > root.span.duration {
				root = node
			}
		}
	}

	sortChildren(root)
	return root
}

func sortChildren(node *spanNode) {
	if node == nil {
		return
	}
	sort.Slice(node.children, func(i, j int) bool {
		return node.children[i].span.startTime.Before(node.children[j].span.startTime)
	})
	for _, child := range node.children {
		sortChildren(child)
	}
}

// connectorLen is the byte length of tree connector strings used in prefix slicing.
const connectorLen = 3

func renderTree(b *strings.Builder, node *spanNode, prefix string, isLast bool, rootDur time.Duration) {
	bar := durationBar(node.span.duration, rootDur)
	fmt.Fprintf(b, "%-48s %s %s\n", prefix+node.span.name, bar, formatDuration(node.span.duration))

	childPrefix := prefix
	if len(prefix) >= connectorLen {
		if isLast {
			childPrefix = prefix[:len(prefix)-connectorLen] + "   "
		} else {
			childPrefix = prefix[:len(prefix)-connectorLen] + "|  "
		}
	}

	for i, child := range node.children {
		last := i == len(node.children)-1
		var connector string
		if last {
			connector = "'- "
		} else {
			connector = "|- "
		}
		renderTree(b, child, childPrefix+connector, last, rootDur)
	}
}

func durationBar(d, rootDur time.Duration) string {
	if rootDur <= 0 {
		return ""
	}
	ratio := float64(d) / float64(rootDur)
	width := int(ratio * maxBarWidth)
	if width < 1 && d > 0 {
		width = 1
	}
	return strings.Repeat("#", width)
}

func formatDuration(d time.Duration) string {
	ms := float64(d) / float64(time.Millisecond)
	if ms >= 1 {
		return fmt.Sprintf("%.1fms", ms)
	}
	return fmt.Sprintf("%.1fus", float64(d)/float64(time.Microsecond))
}

func uniqueHosts(spans []richSpan) []string {
	seen := make(map[string]bool)
	for _, s := range spans {
		if s.hostName != "" {
			seen[s.hostName] = true
		}
	}
	hosts := make([]string, 0, len(seen))
	for h := range seen {
		hosts = append(hosts, h)
	}
	sort.Strings(hosts)
	return hosts
}

func truncateID(id string) string {
	if len(id) > 12 {
		return id[:12] + "..."
	}
	return id
}

// aggregateNode represents a canonical span in the aggregate tree.
type aggregateNode struct {
	name        string
	count       int
	avgDuration time.Duration
	children    []*aggregateNode
}

func buildAggregateTree(byTrace map[string][]richSpan) *aggregateNode {
	// build canonical structure from a representative trace (the longest root)
	root, _ := longestRootTrace(byTrace)
	if root == nil {
		return nil
	}

	// collect durations for each operation name across all traces
	dursByName := make(map[string][]time.Duration)
	countsByName := make(map[string]int)
	for _, spans := range byTrace {
		// count each unique operation once per trace
		seen := make(map[string]bool)
		for _, s := range spans {
			dursByName[s.name] = append(dursByName[s.name], s.duration)
			if !seen[s.name] {
				countsByName[s.name]++
				seen[s.name] = true
			}
		}
	}

	return buildAggNode(root, dursByName, countsByName)
}

func buildAggNode(node *spanNode, dursByName map[string][]time.Duration, countsByName map[string]int) *aggregateNode {
	durs := dursByName[node.span.name]
	var total time.Duration
	for _, d := range durs {
		total += d
	}
	var avg time.Duration
	if len(durs) > 0 {
		avg = total / time.Duration(len(durs))
	}

	agg := &aggregateNode{
		name:        node.span.name,
		count:       countsByName[node.span.name],
		avgDuration: avg,
	}

	for _, child := range node.children {
		agg.children = append(agg.children, buildAggNode(child, dursByName, countsByName))
	}
	return agg
}

func renderAggregateTree(b *strings.Builder, node *aggregateNode, prefix string, isLast bool, rootAvg time.Duration) {
	bar := durationBar(node.avgDuration, rootAvg)
	countStr := ""
	if len(prefix) > 0 {
		countStr = fmt.Sprintf(" (%d calls)", node.count)
	}
	fmt.Fprintf(b, "%-48s %s %s avg%s\n", prefix+node.name, bar, formatDuration(node.avgDuration), countStr)

	childPrefix := prefix
	if len(prefix) >= connectorLen {
		if isLast {
			childPrefix = prefix[:len(prefix)-connectorLen] + "   "
		} else {
			childPrefix = prefix[:len(prefix)-connectorLen] + "|  "
		}
	}

	for i, child := range node.children {
		last := i == len(node.children)-1
		var connector string
		if last {
			connector = "'- "
		} else {
			connector = "|- "
		}
		renderAggregateTree(b, child, childPrefix+connector, last, rootAvg)
	}
}
