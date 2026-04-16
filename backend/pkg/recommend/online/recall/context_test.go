package recall

import "testing"

func TestListProbeJoinCandidateGoodsIds(t *testing.T) {
	probeContext := map[string]any{
		"contentBased": map[string]any{
			"enabled":       true,
			"joinCandidate": true,
			"goodsIds":      []int64{11, 12, 11},
		},
		"collaborativeFiltering": map[string]any{
			"enabled":       true,
			"joinCandidate": false,
			"goodsIds":      []int64{21, 22},
		},
		"similarUser": map[string]any{
			"enabled": true,
			"userIds": []int64{31, 32, 31},
		},
	}

	contentBasedGoodsIds := ListContentBasedJoinCandidateGoodsIds(probeContext)
	if len(contentBasedGoodsIds) != 2 || contentBasedGoodsIds[0] != 11 || contentBasedGoodsIds[1] != 12 {
		t.Fatalf("unexpected content based goods ids: %+v", contentBasedGoodsIds)
	}

	collaborativeFilteringGoodsIds := ListCollaborativeFilteringJoinCandidateGoodsIds(probeContext)
	if len(collaborativeFilteringGoodsIds) != 0 {
		t.Fatalf("unexpected collaborative filtering goods ids: %+v", collaborativeFilteringGoodsIds)
	}

	similarUserIds := ListSimilarUserProbeUserIds(probeContext)
	if len(similarUserIds) != 2 || similarUserIds[0] != 31 || similarUserIds[1] != 32 {
		t.Fatalf("unexpected similar user ids: %+v", similarUserIds)
	}
}

func TestAppendJoinContext(t *testing.T) {
	sourceContext := AppendJoinContext(
		map[string]any{},
		map[string][]int64{
			"cf":      {3, 4, 4},
			"content": {1, 2, 2},
		},
		[]int64{2, 3, 5},
		[]int64{3},
	)

	joinContext, ok := sourceContext["joinRecallContext"].(map[string]any)
	if !ok {
		t.Fatalf("joinRecallContext not found: %+v", sourceContext)
	}

	joinedSources, ok := joinContext["joinedRecallSources"].([]string)
	if !ok || len(joinedSources) != 2 || joinedSources[0] != "cf" || joinedSources[1] != "content" {
		t.Fatalf("unexpected joined sources: %+v", joinContext["joinedRecallSources"])
	}

	effectiveJoinGoodsIds, ok := joinContext["effectiveJoinRecallGoodsIds"].(map[string][]int64)
	if !ok {
		t.Fatalf("unexpected effectiveJoinRecallGoodsIds: %+v", joinContext["effectiveJoinRecallGoodsIds"])
	}
	if len(effectiveJoinGoodsIds["content"]) != 1 || effectiveJoinGoodsIds["content"][0] != 2 {
		t.Fatalf("unexpected effective content goods ids: %+v", effectiveJoinGoodsIds["content"])
	}
	if len(effectiveJoinGoodsIds["cf"]) != 1 || effectiveJoinGoodsIds["cf"][0] != 3 {
		t.Fatalf("unexpected effective cf goods ids: %+v", effectiveJoinGoodsIds["cf"])
	}

	returnedSources, ok := sourceContext["returnedJoinRecallSources"].([]string)
	if !ok || len(returnedSources) != 1 || returnedSources[0] != "cf" {
		t.Fatalf("unexpected returned join sources: %+v", sourceContext["returnedJoinRecallSources"])
	}
}

func TestAppendSimilarUserObservationContext(t *testing.T) {
	sourceContext := AppendSimilarUserObservationContext(
		map[string]any{},
		[]int64{101, 102, 101},
		[]int64{11, 12, 13, 13},
		map[string][]int64{
			"cf":      {11, 20},
			"content": {12, 30},
		},
		[]int64{12, 13, 20},
		[]int64{13},
	)

	observationContext, ok := sourceContext["similarUserObservationContext"].(map[string]any)
	if !ok {
		t.Fatalf("similarUserObservationContext not found: %+v", sourceContext)
	}

	candidateOverlapCount, ok := observationContext["candidateOverlapCount"].(int)
	if !ok || candidateOverlapCount != 2 {
		t.Fatalf("unexpected candidate overlap count: %+v", observationContext["candidateOverlapCount"])
	}

	returnedOverlapCount, ok := observationContext["returnedOverlapCount"].(int)
	if !ok || returnedOverlapCount != 1 {
		t.Fatalf("unexpected returned overlap count: %+v", observationContext["returnedOverlapCount"])
	}

	joinedRecallOverlap, ok := observationContext["joinedRecallOverlap"].(map[string]any)
	if !ok || len(joinedRecallOverlap) != 2 {
		t.Fatalf("unexpected joined recall overlap: %+v", observationContext["joinedRecallOverlap"])
	}
}
