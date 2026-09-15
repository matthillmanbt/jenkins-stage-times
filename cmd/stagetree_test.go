package cmd

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"jenkins/internal/jenkins"
)

// stubStage builds a stage whose Self HREF is its id, so the fake fetcher can
// look it up.
func stubStage(id, name, status string, children ...jenkins.Stage) jenkins.Stage {
	s := jenkins.Stage{StageFlowNodes: children}
	s.ID = id
	s.Name = name
	s.Status = status
	s.Links.Self.HREF = "/stage/" + id
	return s
}

// withFakeFetcher points fetchStage at an in-memory tree for the duration of a
// test. The parent listing in the real API omits grandchildren, so the fake
// mirrors that: a fetch returns the node with its immediate children only.
func withFakeFetcher(t *testing.T, nodes map[string]jenkins.Stage) {
	t.Helper()
	original := fetchStage
	fetchStage = func(stage jenkins.Stage) (jenkins.Stage, error) {
		full, ok := nodes[stage.ID]
		if !ok {
			return jenkins.Stage{}, fmt.Errorf("no such stage %q", stage.ID)
		}
		return full, nil
	}
	t.Cleanup(func() { fetchStage = original })
}

// runWithTimeout fails the test if fn has not returned in time, which is what
// the original worker-pool deadlock looked like from the outside.
func runWithTimeout(t *testing.T, d time.Duration, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("collectLeafStages did not return within %s (deadlock regression)", d)
	}
}

// TestCollectLeafStages_WideTreeDoesNotDeadlock is the regression test for the
// original implementation, a worker pool whose result consumer also fed the job
// queue through channels buffered at len(stages)*2. Any tree with more nodes
// than that buffer wedged: the consumer blocked writing to a full jobs channel
// while every worker blocked writing to a full results channel.
//
// The shape below is deliberately much larger than 2*len(roots) -- 4 roots but
// 4 + 4*60 = 244 nodes, comparable to the 207 leaves a real bpam build has.
func TestCollectLeafStages_WideTreeDoesNotDeadlock(t *testing.T) {
	const roots = 4
	const childrenPerRoot = 60

	nodes := map[string]jenkins.Stage{}
	var topLevel []jenkins.Stage

	for r := 0; r < roots; r++ {
		var children []jenkins.Stage
		for c := 0; c < childrenPerRoot; c++ {
			child := stubStage(fmt.Sprintf("r%d-c%d", r, c), "child", "FAILED")
			children = append(children, child)
			nodes[child.ID] = child
		}
		root := stubStage(fmt.Sprintf("r%d", r), "root", "FAILED", children...)
		nodes[root.ID] = root
		// The top-level listing carries the root without its children, exactly
		// as the job API returns it.
		topLevel = append(topLevel, stubStage(root.ID, "root", "FAILED"))
	}

	withFakeFetcher(t, nodes)

	var leaves []StageWithPath
	runWithTimeout(t, 10*time.Second, func() {
		collectLeafStages(topLevel, nil, keepFailedLeaf, &leaves)
	})

	if want := roots * childrenPerRoot; len(leaves) != want {
		t.Errorf("got %d failed leaves, want %d", len(leaves), want)
	}
}

// TestCollectLeafStages_KeepFailedLeafPicksDeepestFailure checks that a failed
// parent is not reported when a failed child explains the failure, and that a
// successful sibling is ignored.
func TestCollectLeafStages_KeepFailedLeafPicksDeepestFailure(t *testing.T) {
	failedGrandchild := stubStage("3", "checkout", "FAILED")
	failedChild := stubStage("2", "build", "FAILED", failedGrandchild)
	okChild := stubStage("4", "lint", "SUCCESS")
	root := stubStage("1", "pipeline", "FAILED", failedChild, okChild)

	nodes := map[string]jenkins.Stage{
		"1": root, "2": failedChild, "3": failedGrandchild, "4": okChild,
	}
	withFakeFetcher(t, nodes)

	var leaves []StageWithPath
	collectLeafStages([]jenkins.Stage{stubStage("1", "pipeline", "FAILED")}, nil, keepFailedLeaf, &leaves)

	if len(leaves) != 1 {
		t.Fatalf("got %d leaves, want 1: %+v", len(leaves), leaves)
	}
	if leaves[0].Stage.ID != "3" {
		t.Errorf("got leaf id %q, want %q (the deepest failure)", leaves[0].Stage.ID, "3")
	}
	if got, want := strings.Join(leaves[0].Path, " > "), "pipeline > build"; got != want {
		t.Errorf("got path %q, want %q", got, want)
	}
}

// TestCollectLeafStages_SiblingPathsAreNotAliased guards the path handling:
// building a child path with append on a shared backing array let one branch
// overwrite a sibling's ancestry.
func TestCollectLeafStages_SiblingPathsAreNotAliased(t *testing.T) {
	leafA := stubStage("a1", "leaf-a", "FAILED")
	leafB := stubStage("b1", "leaf-b", "FAILED")
	branchA := stubStage("a", "branch-a", "FAILED", leafA)
	branchB := stubStage("b", "branch-b", "FAILED", leafB)
	root := stubStage("r", "root", "FAILED", branchA, branchB)

	nodes := map[string]jenkins.Stage{
		"r": root, "a": branchA, "b": branchB, "a1": leafA, "b1": leafB,
	}
	withFakeFetcher(t, nodes)

	var leaves []StageWithPath
	collectLeafStages([]jenkins.Stage{stubStage("r", "root", "FAILED")}, nil, keepFailedLeaf, &leaves)

	paths := map[string]string{}
	for _, l := range leaves {
		paths[l.Stage.ID] = strings.Join(l.Path, " > ")
	}
	if got, want := paths["a1"], "root > branch-a"; got != want {
		t.Errorf("leaf a1 path = %q, want %q", got, want)
	}
	if got, want := paths["b1"], "root > branch-b"; got != want {
		t.Errorf("leaf b1 path = %q, want %q", got, want)
	}
}

// TestCollectLeafStages_KeepAnyLeaf covers the `diagnose --all` predicate.
func TestCollectLeafStages_KeepAnyLeaf(t *testing.T) {
	child := stubStage("2", "step", "SUCCESS")
	root := stubStage("1", "stage", "SUCCESS", child)
	withFakeFetcher(t, map[string]jenkins.Stage{"1": root, "2": child})

	var leaves []StageWithPath
	collectLeafStages([]jenkins.Stage{stubStage("1", "stage", "SUCCESS")}, nil, keepAnyLeaf, &leaves)

	if len(leaves) != 1 || leaves[0].Stage.ID != "2" {
		t.Errorf("got %+v, want only the childless stage id 2", leaves)
	}
}

// TestCollectLeafStages_FetchErrorIsSkipped checks that one unreachable stage
// does not hide the others.
func TestCollectLeafStages_FetchErrorIsSkipped(t *testing.T) {
	good := stubStage("good", "good", "FAILED")
	withFakeFetcher(t, map[string]jenkins.Stage{"good": good})

	var leaves []StageWithPath
	collectLeafStages(
		[]jenkins.Stage{stubStage("missing", "missing", "FAILED"), stubStage("good", "good", "FAILED")},
		nil, keepFailedLeaf, &leaves)

	if len(leaves) != 1 || leaves[0].Stage.ID != "good" {
		t.Errorf("got %+v, want just the fetchable stage", leaves)
	}
}

// TestCollectLeafStages_CyclicGraphTerminates ensures the depth bound holds
// when a stage reports itself as its own child.
func TestCollectLeafStages_CyclicGraphTerminates(t *testing.T) {
	self := stubStage("loop", "loop", "FAILED", stubStage("loop", "loop", "FAILED"))
	withFakeFetcher(t, map[string]jenkins.Stage{"loop": self})

	var leaves []StageWithPath
	runWithTimeout(t, 10*time.Second, func() {
		collectLeafStages([]jenkins.Stage{stubStage("loop", "loop", "FAILED")}, nil, keepFailedLeaf, &leaves)
	})
	// Never a leaf (it always has a failed child), so the value under test is
	// simply that the walk terminated instead of recursing forever.
	if len(leaves) != 0 {
		t.Errorf("got %d leaves, want 0", len(leaves))
	}
}
