package cache

import (
	"path/filepath"
	"testing"

	recommendv1 "recommend/api/gen/go/recommend/v1"
	"recommend/contract"
	cacheleveldb "recommend/internal/cache/leveldb"

	goleveldb "github.com/syndtr/goleveldb/leveldb"
)

func TestPoolStore(t *testing.T) {
	manager := openTestManager(t)
	defer func() {
		err := manager.Close()
		if err != nil {
			t.Fatalf("关闭 LevelDB 失败: %v", err)
		}
	}()

	store := &PoolStore{Driver: manager}
	expected := &recommendv1.RecommendCandidatePool{
		Items: []*recommendv1.RecommendCandidateItem{
			{
				GoodsId:       1001,
				Score:         9.9,
				RecallSources: []string{"scene_hot"},
			},
		},
	}

	err := store.SaveCandidatePool("home", 1, 2001, expected)
	if err != nil {
		t.Fatalf("保存通用候选池失败: %v", err)
	}

	pool, err := store.GetCandidatePool("home", 1, 2001)
	if err != nil {
		t.Fatalf("读取通用候选池失败: %v", err)
	}
	if len(pool.GetItems()) != 1 || pool.GetItems()[0].GetGoodsId() != 1001 {
		t.Fatalf("候选池数据不符合预期: %+v", pool)
	}

	err = store.DeleteCandidatePool("home", 1, 2001)
	if err != nil {
		t.Fatalf("删除通用候选池失败: %v", err)
	}

	_, err = store.GetCandidatePool("home", 1, 2001)
	if err == nil {
		t.Fatal("删除后读取通用候选池应返回错误")
	}
	if err != goleveldb.ErrNotFound {
		t.Fatalf("删除后读取通用候选池错误不符合预期: %v", err)
	}
}

func TestRuntimeStore(t *testing.T) {
	manager := openTestManager(t)
	defer func() {
		err := manager.Close()
		if err != nil {
			t.Fatalf("关闭 LevelDB 失败: %v", err)
		}
	}()

	store := &RuntimeStore{Driver: manager}
	expected := &recommendv1.RecommendSessionState{
		RecentViewGoodsIds: []int64{11, 12},
		RecentCartGoodsIds: []int64{21},
	}

	err := store.SaveSessionState(1, 3001, "session-1", expected)
	if err != nil {
		t.Fatalf("保存会话态失败: %v", err)
	}

	state, err := store.GetSessionState(1, 3001, "session-1")
	if err != nil {
		t.Fatalf("读取会话态失败: %v", err)
	}
	if len(state.GetRecentViewGoodsIds()) != 2 || state.GetRecentCartGoodsIds()[0] != 21 {
		t.Fatalf("会话态数据不符合预期: %+v", state)
	}

	penaltyState := &recommendv1.RecommendPenaltyState{
		ExposurePenalty: map[int64]float64{11: 0.3},
	}
	err = store.SavePenaltyState("home", 1, 3001, penaltyState)
	if err != nil {
		t.Fatalf("保存惩罚态失败: %v", err)
	}

	loadedPenaltyState, err := store.GetPenaltyState("home", 1, 3001)
	if err != nil {
		t.Fatalf("读取惩罚态失败: %v", err)
	}
	if loadedPenaltyState.GetExposurePenalty()[11] != 0.3 {
		t.Fatalf("惩罚态数据不符合预期: %+v", loadedPenaltyState)
	}
}

func TestTraceStore(t *testing.T) {
	manager := openTestManager(t)
	defer func() {
		err := manager.Close()
		if err != nil {
			t.Fatalf("关闭 LevelDB 失败: %v", err)
		}
	}()

	store := &TraceStore{Driver: manager}
	expected := &recommendv1.RecommendTraceDetail{
		RequestId:      "request-1",
		Scene:          "goods_detail",
		ResultGoodsIds: []int64{501, 502},
	}

	err := store.SaveTraceDetail("trace-1", "request-1", expected)
	if err != nil {
		t.Fatalf("保存追踪详情失败: %v", err)
	}

	detail, err := store.GetTraceDetail("trace-1")
	if err != nil {
		t.Fatalf("按追踪编号读取追踪详情失败: %v", err)
	}
	if detail.GetRequestId() != "request-1" || len(detail.GetResultGoodsIds()) != 2 {
		t.Fatalf("追踪详情不符合预期: %+v", detail)
	}

	detail, err = store.GetTraceDetailByRequestId("request-1")
	if err != nil {
		t.Fatalf("按请求编号读取追踪详情失败: %v", err)
	}
	if detail.GetScene() != "goods_detail" {
		t.Fatalf("按请求编号回查结果不符合预期: %+v", detail)
	}

	err = store.DeleteTraceDetail("trace-1", "request-1")
	if err != nil {
		t.Fatalf("删除追踪详情失败: %v", err)
	}

	_, err = store.GetTraceDetail("trace-1")
	if err == nil {
		t.Fatal("删除后读取追踪详情应返回错误")
	}
	if err != goleveldb.ErrNotFound {
		t.Fatalf("删除后读取追踪详情错误不符合预期: %v", err)
	}
}

func openTestManager(t *testing.T) *cacheleveldb.Manager {
	t.Helper()

	rootPath := t.TempDir()
	layout := contract.LevelDbLayout{
		PoolPath:    filepath.Join(rootPath, "pool.db"),
		RuntimePath: filepath.Join(rootPath, "runtime.db"),
		TracePath:   filepath.Join(rootPath, "trace.db"),
	}

	manager, err := cacheleveldb.OpenManagerByLayout(layout)
	if err != nil {
		t.Fatalf("打开 LevelDB 管理器失败: %v", err)
	}
	return manager
}
