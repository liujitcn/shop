package scene

import (
	"errors"
	cachex "recommend/internal/cache"
	"recommend/internal/model"

	goleveldb "github.com/syndtr/goleveldb/leveldb"
)

// loadRuntimePenaltyState 加载当前主体在指定场景下的运行态惩罚。
func loadRuntimePenaltyState(request model.Request, runtimeStore *cachex.RuntimeStore) (map[int64]float64, map[int64]float64, error) {
	// 运行态缓存未启用时，当前请求回退为无惩罚模式。
	if runtimeStore == nil {
		return nil, nil, nil
	}
	// 主体编号非法时，无法构造稳定的运行态键。
	if request.Actor.Id <= 0 {
		return nil, nil, nil
	}

	state, err := runtimeStore.GetPenaltyState(request.Scene.String(), int32(request.Actor.Type), request.Actor.Id)
	if err != nil {
		// 当前主体还没有建立运行态惩罚缓存时，直接按空惩罚处理。
		if errors.Is(err, goleveldb.ErrNotFound) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	return clonePenaltyMap(state.GetExposurePenalty()), clonePenaltyMap(state.GetRepeatPenalty()), nil
}

// clonePenaltyMap 复制惩罚 map，避免后续排序链路修改缓存对象原始数据。
func clonePenaltyMap(source map[int64]float64) map[int64]float64 {
	if len(source) == 0 {
		return nil
	}

	result := make(map[int64]float64, len(source))
	for goodsId, penalty := range source {
		// 非法商品编号或非法惩罚值不需要进入在线排序阶段。
		if goodsId <= 0 || penalty <= 0 {
			continue
		}
		result[goodsId] = penalty
	}
	return result
}
