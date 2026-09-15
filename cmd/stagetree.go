package cmd

import (
	"encoding/json"
	"net/http"
	"sync"

	"jenkins/internal/jenkins"
)

const (
	// stageFetchConcurrency caps in-flight Jenkins API requests per level.
	stageFetchConcurrency = 10

	// maxStageTreeDepth bounds the walk. Real pipelines nest a handful of
	// levels; this only exists so a self-referential stage graph cannot spin
	// forever.
	maxStageTreeDepth = 64
)

// fetchStage is indirected through a variable so tests can walk a synthetic
// stage tree without a Jenkins server. Production always uses fetchStageDetail.
var fetchStage = fetchStageDetail

// isFailedStatus reports whether a stage status counts as a failure.
func isFailedStatus(status string) bool {
	return status == "FAILED" || status == "ABORTED"
}

// keepFailedLeaf selects the deepest failed stages: a failed stage none of
// whose children also failed. That is the stage whose log holds the actual
// error, rather than a parent that merely propagated it.
func keepFailedLeaf(stage jenkins.Stage) bool {
	if !isFailedStatus(stage.Status) {
		return false
	}
	for _, child := range stage.StageFlowNodes {
		if isFailedStatus(child.Status) {
			return false
		}
	}
	return true
}

// keepAnyLeaf selects every stage that has no children.
func keepAnyLeaf(stage jenkins.Stage) bool {
	return len(stage.StageFlowNodes) == 0
}

// collectLeafStages walks a build's stage tree breadth-first, fetching each
// level's detail concurrently, and appends every stage that keep accepts.
//
// This replaces three copies of a worker pool whose result consumer also fed
// the job queue. That shape deadlocked on any build with more stages than the
// guessed channel buffer (len(stages)*2): the consumer blocked sending to a
// full jobs channel, all ten workers blocked sending to a full results
// channel, and with the consumer stuck nothing was left to drain either one.
// `jenkins failed` and `jenkins diagnose` both died with "all goroutines are
// asleep - deadlock!" instead of reporting why a build failed.
//
// Collecting a whole level before deriving the next removes the feedback edge,
// so there is no buffer to size and no ordering between producer and consumer
// to get wrong.
func collectLeafStages(stages []jenkins.Stage, path []string, keep func(jenkins.Stage) bool, leaves *[]StageWithPath) {
	type node struct {
		stage jenkins.Stage
		path  []string
	}

	level := make([]node, 0, len(stages))
	for _, stage := range stages {
		level = append(level, node{stage: stage, path: clonePath(path)})
	}

	for depth := 0; depth < maxStageTreeDepth && len(level) > 0; depth++ {
		fetched := make([]jenkins.Stage, len(level))
		errs := make([]error, len(level))

		sem := make(chan struct{}, stageFetchConcurrency)
		var wg sync.WaitGroup
		for i := range level {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				fetched[i], errs[i] = fetchStage(level[i].stage)
			}(i)
		}
		wg.Wait()

		var next []node
		for i, n := range level {
			if errs[i] != nil {
				// A stage we cannot fetch is skipped rather than fatal: one
				// unreachable node should not hide every other failure.
				verbose("Error fetching stage %q: %v", n.stage.Name, errs[i])
				continue
			}
			stage := fetched[i]

			if keep(stage) {
				*leaves = append(*leaves, StageWithPath{Stage: stage, Path: n.path})
			}

			if len(stage.StageFlowNodes) == 0 {
				continue
			}
			childPath := append(clonePath(n.path), stage.Name)
			for _, child := range stage.StageFlowNodes {
				next = append(next, node{stage: child, path: clonePath(childPath)})
			}
		}
		level = next
	}
}

// fetchStageDetail retrieves a stage's full detail, including its child flow
// nodes, which the parent listing does not include.
func fetchStageDetail(stage jenkins.Stage) (jenkins.Stage, error) {
	res, err := jenkinsClient.Request(http.MethodGet, stage.Links.Self.HREF)
	if err != nil {
		return jenkins.Stage{}, err
	}
	defer res.Body.Close()

	var detail jenkins.Stage
	if err := json.NewDecoder(res.Body).Decode(&detail); err != nil {
		return jenkins.Stage{}, err
	}
	return detail, nil
}

// clonePath copies a path slice, so that appending a stage name in one branch
// of the walk cannot overwrite a sibling's ancestry through a shared backing
// array.
func clonePath(path []string) []string {
	out := make([]string, len(path))
	copy(out, path)
	return out
}
