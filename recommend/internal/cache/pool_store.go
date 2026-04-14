package cache

import (
	"errors"

	recommendv1 "recommend/api/gen/go/recommend/v1"
	"recommend/internal/cache/driver"
	cacheleveldb "recommend/internal/cache/leveldb"

	"google.golang.org/protobuf/proto"
)

// PoolStore 负责管理候选池相关缓存。
type PoolStore struct {
	Driver driver.Provider
}

// SaveCandidatePool 保存通用候选池。
func (s *PoolStore) SaveCandidatePool(scene string, actorType int32, actorId int64, pool *recommendv1.RecommendCandidatePool) error {
	return s.putMessage(cacheleveldb.CandidatePoolKey(scene, actorType, actorId), pool)
}

// GetCandidatePool 读取通用候选池。
func (s *PoolStore) GetCandidatePool(scene string, actorType int32, actorId int64) (*recommendv1.RecommendCandidatePool, error) {
	pool := &recommendv1.RecommendCandidatePool{}
	err := s.loadMessage(cacheleveldb.CandidatePoolKey(scene, actorType, actorId), pool)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

// DeleteCandidatePool 删除通用候选池。
func (s *PoolStore) DeleteCandidatePool(scene string, actorType int32, actorId int64) error {
	return s.deleteKey(cacheleveldb.CandidatePoolKey(scene, actorType, actorId))
}

// SaveRelatedGoodsPool 保存商品关联池。
func (s *PoolStore) SaveRelatedGoodsPool(scene string, sourceGoodsId int64, pool *recommendv1.RecommendRelatedGoodsPool) error {
	return s.putMessage(cacheleveldb.RelatedGoodsPoolKey(scene, sourceGoodsId), pool)
}

// GetRelatedGoodsPool 读取商品关联池。
func (s *PoolStore) GetRelatedGoodsPool(scene string, sourceGoodsId int64) (*recommendv1.RecommendRelatedGoodsPool, error) {
	pool := &recommendv1.RecommendRelatedGoodsPool{}
	err := s.loadMessage(cacheleveldb.RelatedGoodsPoolKey(scene, sourceGoodsId), pool)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

// DeleteRelatedGoodsPool 删除商品关联池。
func (s *PoolStore) DeleteRelatedGoodsPool(scene string, sourceGoodsId int64) error {
	return s.deleteKey(cacheleveldb.RelatedGoodsPoolKey(scene, sourceGoodsId))
}

// SaveUserCandidatePool 保存用户候选池。
func (s *PoolStore) SaveUserCandidatePool(scene string, userId int64, pool *recommendv1.RecommendUserCandidatePool) error {
	return s.putMessage(cacheleveldb.UserCandidatePoolKey(scene, userId), pool)
}

// GetUserCandidatePool 读取用户候选池。
func (s *PoolStore) GetUserCandidatePool(scene string, userId int64) (*recommendv1.RecommendUserCandidatePool, error) {
	pool := &recommendv1.RecommendUserCandidatePool{}
	err := s.loadMessage(cacheleveldb.UserCandidatePoolKey(scene, userId), pool)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

// DeleteUserCandidatePool 删除用户候选池。
func (s *PoolStore) DeleteUserCandidatePool(scene string, userId int64) error {
	return s.deleteKey(cacheleveldb.UserCandidatePoolKey(scene, userId))
}

// SaveUserNeighborPool 保存相似用户池。
func (s *PoolStore) SaveUserNeighborPool(userId int64, pool *recommendv1.RecommendUserNeighborPool) error {
	return s.putMessage(cacheleveldb.UserNeighborPoolKey(userId), pool)
}

// GetUserNeighborPool 读取相似用户池。
func (s *PoolStore) GetUserNeighborPool(userId int64) (*recommendv1.RecommendUserNeighborPool, error) {
	pool := &recommendv1.RecommendUserNeighborPool{}
	err := s.loadMessage(cacheleveldb.UserNeighborPoolKey(userId), pool)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

// DeleteUserNeighborPool 删除相似用户池。
func (s *PoolStore) DeleteUserNeighborPool(userId int64) error {
	return s.deleteKey(cacheleveldb.UserNeighborPoolKey(userId))
}

// SaveCollaborativePool 保存协同过滤池。
func (s *PoolStore) SaveCollaborativePool(scene string, userId int64, pool *recommendv1.RecommendCollaborativePool) error {
	return s.putMessage(cacheleveldb.CollaborativePoolKey(scene, userId), pool)
}

// GetCollaborativePool 读取协同过滤池。
func (s *PoolStore) GetCollaborativePool(scene string, userId int64) (*recommendv1.RecommendCollaborativePool, error) {
	pool := &recommendv1.RecommendCollaborativePool{}
	err := s.loadMessage(cacheleveldb.CollaborativePoolKey(scene, userId), pool)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

// DeleteCollaborativePool 删除协同过滤池。
func (s *PoolStore) DeleteCollaborativePool(scene string, userId int64) error {
	return s.deleteKey(cacheleveldb.CollaborativePoolKey(scene, userId))
}

// SaveExternalPool 保存外部推荐池。
func (s *PoolStore) SaveExternalPool(scene, strategy string, actorType int32, actorId int64, pool *recommendv1.RecommendExternalPool) error {
	return s.putMessage(cacheleveldb.ExternalPoolKey(scene, strategy, actorType, actorId), pool)
}

// GetExternalPool 读取外部推荐池。
func (s *PoolStore) GetExternalPool(scene, strategy string, actorType int32, actorId int64) (*recommendv1.RecommendExternalPool, error) {
	pool := &recommendv1.RecommendExternalPool{}
	err := s.loadMessage(cacheleveldb.ExternalPoolKey(scene, strategy, actorType, actorId), pool)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

// DeleteExternalPool 删除外部推荐池。
func (s *PoolStore) DeleteExternalPool(scene, strategy string, actorType int32, actorId int64) error {
	return s.deleteKey(cacheleveldb.ExternalPoolKey(scene, strategy, actorType, actorId))
}

// poolDb 返回候选池数据库实例。
func (s *PoolStore) poolDb() driver.BinaryStore {
	if s != nil && s.Driver != nil {
		return s.Driver.PoolStore()
	}
	return nil
}

// putMessage 向候选池数据库写入一条 proto 消息。
func (s *PoolStore) putMessage(key []byte, message proto.Message) error {
	db := s.poolDb()
	// 候选池数据库未绑定时，说明 store 还没有正确初始化。
	if db == nil {
		return errors.New("recommend: 候选池数据库未初始化")
	}
	rawValue, err := cacheleveldb.EncodeMessage(message)
	if err != nil {
		return err
	}
	return db.Put(key, rawValue)
}

// loadMessage 从候选池数据库加载 proto 消息。
func (s *PoolStore) loadMessage(key []byte, message proto.Message) error {
	db := s.poolDb()
	// 候选池数据库未绑定时，说明 store 还没有正确初始化。
	if db == nil {
		return errors.New("recommend: 候选池数据库未初始化")
	}
	rawValue, err := db.Get(key)
	if err != nil {
		return err
	}
	return cacheleveldb.DecodeMessage(rawValue, message)
}

// deleteKey 从候选池数据库删除一条记录。
func (s *PoolStore) deleteKey(key []byte) error {
	db := s.poolDb()
	// 候选池数据库未绑定时，说明 store 还没有正确初始化。
	if db == nil {
		return errors.New("recommend: 候选池数据库未初始化")
	}
	return db.Delete(key)
}
