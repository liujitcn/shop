package cache

import (
	"errors"

	recommendv1 "recommend/api/gen/go/recommend/v1"
	"recommend/internal/cache/driver"
	cacheleveldb "recommend/internal/cache/leveldb"

	"google.golang.org/protobuf/proto"
)

// RuntimeStore 负责管理运行态相关缓存。
type RuntimeStore struct {
	Driver driver.Provider
}

// SaveSessionState 保存会话态。
func (s *RuntimeStore) SaveSessionState(actorType int32, actorId int64, sessionId string, state *recommendv1.RecommendSessionState) error {
	return s.putMessage(cacheleveldb.SessionStateKey(actorType, actorId, sessionId), state)
}

// GetSessionState 读取会话态。
func (s *RuntimeStore) GetSessionState(actorType int32, actorId int64, sessionId string) (*recommendv1.RecommendSessionState, error) {
	state := &recommendv1.RecommendSessionState{}
	err := s.loadMessage(cacheleveldb.SessionStateKey(actorType, actorId, sessionId), state)
	if err != nil {
		return nil, err
	}
	return state, nil
}

// DeleteSessionState 删除会话态。
func (s *RuntimeStore) DeleteSessionState(actorType int32, actorId int64, sessionId string) error {
	return s.deleteKey(cacheleveldb.SessionStateKey(actorType, actorId, sessionId))
}

// SavePenaltyState 保存惩罚态。
func (s *RuntimeStore) SavePenaltyState(scene string, actorType int32, actorId int64, state *recommendv1.RecommendPenaltyState) error {
	return s.putMessage(cacheleveldb.PenaltyStateKey(scene, actorType, actorId), state)
}

// GetPenaltyState 读取惩罚态。
func (s *RuntimeStore) GetPenaltyState(scene string, actorType int32, actorId int64) (*recommendv1.RecommendPenaltyState, error) {
	state := &recommendv1.RecommendPenaltyState{}
	err := s.loadMessage(cacheleveldb.PenaltyStateKey(scene, actorType, actorId), state)
	if err != nil {
		return nil, err
	}
	return state, nil
}

// DeletePenaltyState 删除惩罚态。
func (s *RuntimeStore) DeletePenaltyState(scene string, actorType int32, actorId int64) error {
	return s.deleteKey(cacheleveldb.PenaltyStateKey(scene, actorType, actorId))
}

// SaveRankingModelState 保存学习排序模型状态。
func (s *RuntimeStore) SaveRankingModelState(scene, modelName string, state *recommendv1.RecommendRankingModelState) error {
	return s.putMessage(cacheleveldb.RankingModelStateKey(scene, modelName), state)
}

// GetRankingModelState 读取学习排序模型状态。
func (s *RuntimeStore) GetRankingModelState(scene, modelName string) (*recommendv1.RecommendRankingModelState, error) {
	state := &recommendv1.RecommendRankingModelState{}
	err := s.loadMessage(cacheleveldb.RankingModelStateKey(scene, modelName), state)
	if err != nil {
		return nil, err
	}
	return state, nil
}

// DeleteRankingModelState 删除学习排序模型状态。
func (s *RuntimeStore) DeleteRankingModelState(scene, modelName string) error {
	return s.deleteKey(cacheleveldb.RankingModelStateKey(scene, modelName))
}

// runtimeDb 返回运行态数据库实例。
func (s *RuntimeStore) runtimeDb() driver.BinaryStore {
	if s != nil && s.Driver != nil {
		return s.Driver.RuntimeStore()
	}
	return nil
}

// putMessage 向运行态数据库写入一条 proto 消息。
func (s *RuntimeStore) putMessage(key []byte, message proto.Message) error {
	db := s.runtimeDb()
	// 运行态数据库未绑定时，说明 store 还没有正确初始化。
	if db == nil {
		return errors.New("recommend: 运行态数据库未初始化")
	}
	rawValue, err := cacheleveldb.EncodeMessage(message)
	if err != nil {
		return err
	}
	return db.Put(key, rawValue)
}

// loadMessage 从运行态数据库加载 proto 消息。
func (s *RuntimeStore) loadMessage(key []byte, message proto.Message) error {
	db := s.runtimeDb()
	// 运行态数据库未绑定时，说明 store 还没有正确初始化。
	if db == nil {
		return errors.New("recommend: 运行态数据库未初始化")
	}
	rawValue, err := db.Get(key)
	if err != nil {
		return err
	}
	return cacheleveldb.DecodeMessage(rawValue, message)
}

// deleteKey 从运行态数据库删除一条记录。
func (s *RuntimeStore) deleteKey(key []byte) error {
	db := s.runtimeDb()
	// 运行态数据库未绑定时，说明 store 还没有正确初始化。
	if db == nil {
		return errors.New("recommend: 运行态数据库未初始化")
	}
	return db.Delete(key)
}
