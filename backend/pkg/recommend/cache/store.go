package cache

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	bootstrapConf "github.com/liujitcn/kratos-kit/api/gen/go/conf"
	kitcache "github.com/liujitcn/kratos-kit/cache"
)

const (
	// scoreSubsetIndexSuffix 表示排序集合子集合索引后缀。
	scoreSubsetIndexSuffix = "score_subset_index"
	// permanentDuration 表示推荐缓存默认的长效过期时间。
	permanentDuration = time.Hour * 24 * 365 * 100
)

// HashStore 表示基于 KV / Hash 能力封装的推荐缓存实现。
type HashStore struct {
	cache     kitcache.Cache // 基础缓存实现。
	cleanup   func()         // 底层缓存释放函数。
	closeOnce sync.Once      // 确保底层资源只释放一次。
}

// scoreDocument 表示排序型缓存的持久化文档结构。
type scoreDocument struct {
	Id         string    `json:"id"`         // 条目主键。
	Score      float64   `json:"score"`      // 排序分数。
	IsHidden   bool      `json:"is_hidden"`  // 是否隐藏。
	Categories []string  `json:"categories"` // 分类集合。
	Timestamp  time.Time `json:"timestamp"`  // 更新时间。
}

// NewStore 创建推荐缓存存储实现。
func NewStore(redisConfig *bootstrapConf.Data_Redis) (Store, func(), error) {
	baseCache, cleanup, err := kitcache.NewCache(redisConfig)
	if err != nil {
		return nil, cleanup, err
	}
	store := &HashStore{
		cache:   baseCache,
		cleanup: cleanup,
	}
	return store, func() {
		_ = store.Close()
	}, nil
}

// Close 关闭推荐缓存底层资源。
func (s *HashStore) Close() error {
	// 关闭操作只需要执行一次，避免重复释放底层资源。
	s.closeOnce.Do(func() {
		if s.cleanup != nil {
			s.cleanup()
		}
	})
	return nil
}

// Ping 检查推荐缓存是否已完成初始化。
func (s *HashStore) Ping() error {
	// 底层缓存未注入时，说明当前存储不可用。
	if s == nil || s.cache == nil {
		return ErrNoStore
	}
	return nil
}

// Get 读取普通缓存值。
func (s *HashStore) Get(key string) (string, error) {
	err := s.Ping()
	if err != nil {
		return "", err
	}

	var value string
	value, err = s.cache.Get(key)
	if err != nil {
		return "", normalizeObjectError(err)
	}
	return value, nil
}

// Set 写入普通缓存值。
func (s *HashStore) Set(key, value string, expire time.Duration) error {
	err := s.Ping()
	if err != nil {
		return err
	}
	return s.cache.Set(key, value, normalizeExpire(expire))
}

// Del 删除普通缓存值。
func (s *HashStore) Del(key string) error {
	err := s.Ping()
	if err != nil {
		return err
	}

	err = s.cache.Del(key)
	if isObjectNotExistError(err) {
		return nil
	}
	return err
}

// Expire 设置普通缓存过期时间。
func (s *HashStore) Expire(key string, dur time.Duration) error {
	err := s.Ping()
	if err != nil {
		return err
	}
	err = s.cache.Expire(key, normalizeExpire(dur))
	if err == nil {
		return nil
	}
	normalizedErr := normalizeObjectError(err)
	if normalizedErr != ErrObjectNotExist {
		return normalizedErr
	}

	// 内存缓存把 Hash 与 String 分开存储，哈希键没有独立的 Expire 能力。
	// 这里探测到哈希键实际存在时，按“底层不支持 TTL 但写入已成功”处理，避免本地链路被误判为失败。
	_, hashErr := s.cache.HGetAll(key)
	if hashErr == nil {
		return nil
	}
	return normalizeObjectError(hashErr)
}

// Exists 判断普通缓存值是否存在。
func (s *HashStore) Exists(key string) bool {
	// 底层缓存不可用时，统一视为不存在。
	if s == nil || s.cache == nil {
		return false
	}
	return s.cache.Exists(key)
}

// HGetAll 读取哈希缓存全部字段。
func (s *HashStore) HGetAll(key string) (map[string]string, error) {
	err := s.Ping()
	if err != nil {
		return nil, err
	}

	var value map[string]string
	value, err = s.cache.HGetAll(key)
	if err != nil {
		return nil, normalizeObjectError(err)
	}
	return value, nil
}

// HGet 读取哈希缓存字段值。
func (s *HashStore) HGet(key, field string) (string, error) {
	err := s.Ping()
	if err != nil {
		return "", err
	}

	var value string
	value, err = s.cache.HGet(key, field)
	if err != nil {
		return "", normalizeObjectError(err)
	}
	return value, nil
}

// HSet 写入哈希缓存字段值。
func (s *HashStore) HSet(key, field, value string) error {
	err := s.Ping()
	if err != nil {
		return err
	}
	return s.cache.HSet(key, field, value)
}

// HDel 删除哈希缓存字段值。
func (s *HashStore) HDel(key, field string) error {
	err := s.Ping()
	if err != nil {
		return err
	}

	err = s.cache.HDel(key, field)
	if isObjectNotExistError(err) {
		return nil
	}
	return err
}

// HExists 判断哈希缓存字段是否存在。
func (s *HashStore) HExists(key, field string) error {
	err := s.Ping()
	if err != nil {
		return err
	}

	err = s.cache.HExists(key, field)
	if err != nil {
		return normalizeObjectError(err)
	}
	return nil
}

// AddScores 用最新文档集合覆盖指定子集合内容。
func (s *HashStore) AddScores(_ context.Context, collection, subset string, documents []Score) error {
	err := s.Ping()
	if err != nil {
		return err
	}

	hashKey := scoreHashKey(collection, subset)
	existingMap := make(map[string]string)
	existingMap, err = s.cache.HGetAll(hashKey)
	if err != nil && !isObjectNotExistError(err) {
		return err
	}

	encodedMap := make(map[string]string, len(documents))
	for _, item := range documents {
		payload, marshalErr := marshalScoreDocument(item)
		if marshalErr != nil {
			return marshalErr
		}
		encodedMap[item.Id] = payload
	}
	for id := range existingMap {
		_, ok := encodedMap[id]
		// 本次发布未包含的旧条目需要先删除，避免残留脏数据。
		if ok {
			continue
		}
		err = s.cache.HDel(hashKey, id)
		if err != nil && !isObjectNotExistError(err) {
			return err
		}
	}
	for id, payload := range encodedMap {
		err = s.cache.HSet(hashKey, id, payload)
		if err != nil {
			return err
		}
	}
	return s.cache.HSet(ScoreSubsetIndexKey(collection), subset, time.Now().Format(time.RFC3339Nano))
}

// SearchScores 读取指定子集合的排序型缓存。
func (s *HashStore) SearchScores(_ context.Context, collection, subset string, begin, end int) ([]Score, error) {
	err := s.Ping()
	if err != nil {
		return nil, err
	}

	var documentMap map[string]string
	documentMap, err = s.cache.HGetAll(scoreHashKey(collection, subset))
	if err != nil {
		return nil, normalizeObjectError(err)
	}
	// 子集合为空时，统一按对象不存在处理。
	if len(documentMap) == 0 {
		return nil, ErrObjectNotExist
	}

	list := make([]Score, 0, len(documentMap))
	for _, payload := range documentMap {
		document, unmarshalErr := unmarshalScoreDocument(payload)
		if unmarshalErr != nil {
			return nil, unmarshalErr
		}
		// 隐藏条目不向读取侧返回，避免污染召回结果。
		if document.IsHidden {
			continue
		}
		list = append(list, document)
	}
	SortDocuments(list)
	return sliceScores(list, begin, end), nil
}

// DeleteScores 删除满足条件的排序型缓存文档。
func (s *HashStore) DeleteScores(_ context.Context, collections []string, condition ScoreCondition) error {
	err := s.Ping()
	if err != nil {
		return err
	}
	err = condition.Check()
	if err != nil {
		return err
	}

	for _, collection := range collections {
		subsetList, subsetErr := s.listScoreSubsets(collection, condition.Subset)
		if subsetErr != nil {
			return subsetErr
		}
		for _, subset := range subsetList {
			err = s.deleteSubsetScores(collection, subset, condition)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// UpdateScores 更新满足条件的排序型缓存文档。
func (s *HashStore) UpdateScores(_ context.Context, collections []string, subset *string, id string, patch ScorePatch) error {
	err := s.Ping()
	if err != nil {
		return err
	}

	for _, collection := range collections {
		subsetList, subsetErr := s.listScoreSubsets(collection, subset)
		if subsetErr != nil {
			return subsetErr
		}
		for _, currentSubset := range subsetList {
			err = s.updateSubsetScore(collection, currentSubset, id, patch)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// listScoreSubsets 返回当前集合下需要处理的子集合列表。
func (s *HashStore) listScoreSubsets(collection string, subset *string) ([]string, error) {
	// 已明确指定子集合时，不再读取索引。
	if subset != nil {
		return []string{*subset}, nil
	}

	subsetMap, err := s.cache.HGetAll(ScoreSubsetIndexKey(collection))
	if err != nil {
		if isObjectNotExistError(err) {
			return []string{}, nil
		}
		return nil, err
	}

	result := make([]string, 0, len(subsetMap))
	for item := range subsetMap {
		result = append(result, item)
	}
	return result, nil
}

// deleteSubsetScores 删除单个子集合下满足条件的文档。
func (s *HashStore) deleteSubsetScores(collection, subset string, condition ScoreCondition) error {
	hashKey := scoreHashKey(collection, subset)
	documentMap, err := s.cache.HGetAll(hashKey)
	if err != nil {
		if isObjectNotExistError(err) {
			return nil
		}
		return err
	}
	// 子集合为空时，顺带清理索引字段。
	if len(documentMap) == 0 {
		return s.removeSubsetIndex(collection, subset)
	}

	for docId, payload := range documentMap {
		document, unmarshalErr := unmarshalScoreDocument(payload)
		if unmarshalErr != nil {
			return unmarshalErr
		}
		// 不满足删除条件的文档保留。
		if !matchScoreCondition(document, condition) {
			continue
		}
		err = s.cache.HDel(hashKey, docId)
		if err != nil && !isObjectNotExistError(err) {
			return err
		}
	}

	remainingMap, err := s.cache.HGetAll(hashKey)
	if err != nil {
		if isObjectNotExistError(err) {
			return s.removeSubsetIndex(collection, subset)
		}
		return err
	}
	// 子集合已经删空时，移除对应索引。
	if len(remainingMap) == 0 {
		return s.removeSubsetIndex(collection, subset)
	}
	return nil
}

// updateSubsetScore 更新单个子集合中的指定文档。
func (s *HashStore) updateSubsetScore(collection, subset, id string, patch ScorePatch) error {
	hashKey := scoreHashKey(collection, subset)
	payload, err := s.cache.HGet(hashKey, id)
	if err != nil {
		if isObjectNotExistError(err) {
			return nil
		}
		return err
	}

	document, err := unmarshalScoreDocument(payload)
	if err != nil {
		return err
	}
	if patch.IsHidden != nil {
		document.IsHidden = *patch.IsHidden
	}
	if patch.Categories != nil {
		document.Categories = patch.Categories
	}
	if patch.Score != nil {
		document.Score = *patch.Score
	}

	payload, err = marshalScoreDocument(document)
	if err != nil {
		return err
	}
	return s.cache.HSet(hashKey, id, payload)
}

// removeSubsetIndex 清理子集合索引字段。
func (s *HashStore) removeSubsetIndex(collection, subset string) error {
	err := s.cache.HDel(ScoreSubsetIndexKey(collection), subset)
	if err != nil && !isObjectNotExistError(err) {
		return err
	}
	return nil
}

// matchScoreCondition 判断当前文档是否命中过滤条件。
func matchScoreCondition(document Score, condition ScoreCondition) bool {
	if condition.Id != nil && document.Id != *condition.Id {
		return false
	}
	// Before 仅命中更新时间早于阈值的旧文档。
	if condition.Before != nil && !document.Timestamp.Before(*condition.Before) {
		return false
	}
	return true
}

// scoreHashKey 返回排序型缓存文档的哈希键。
func scoreHashKey(collection, subset string) string {
	return Key(collection, subset)
}

// normalizeExpire 统一处理底层缓存对零过期时间的兼容问题。
func normalizeExpire(expire time.Duration) time.Duration {
	// 推荐缓存默认使用永久键，避免内存后端把零过期误判为立即过期。
	if expire <= 0 {
		return permanentDuration
	}
	return expire
}

// marshalScoreDocument 编码排序型缓存文档。
func marshalScoreDocument(document Score) (string, error) {
	payload, err := json.Marshal(scoreDocument{
		Id:         document.Id,
		Score:      document.Score,
		IsHidden:   document.IsHidden,
		Categories: document.Categories,
		Timestamp:  document.Timestamp,
	})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

// unmarshalScoreDocument 解码排序型缓存文档。
func unmarshalScoreDocument(payload string) (Score, error) {
	document := scoreDocument{}
	err := json.Unmarshal([]byte(payload), &document)
	if err != nil {
		return Score{}, err
	}
	return Score{
		Id:         document.Id,
		Score:      document.Score,
		IsHidden:   document.IsHidden,
		Categories: document.Categories,
		Timestamp:  document.Timestamp,
	}, nil
}

// sliceScores 按区间裁剪排序型缓存结果。
func sliceScores(list []Score, begin, end int) []Score {
	if begin < 0 {
		begin = 0
	}
	if begin >= len(list) {
		return []Score{}
	}
	if end <= 0 || end > len(list) {
		end = len(list)
	}
	if end < begin {
		end = begin
	}
	return append([]Score(nil), list[begin:end]...)
}

// normalizeObjectError 统一转换底层缓存的对象不存在错误。
func normalizeObjectError(err error) error {
	if isObjectNotExistError(err) {
		return ErrObjectNotExist
	}
	return err
}

// isObjectNotExistError 判断错误是否属于底层缓存的对象不存在语义。
func isObjectNotExistError(err error) bool {
	if err == nil {
		return false
	}
	if err == ErrObjectNotExist {
		return true
	}
	errText := strings.ToLower(err.Error())
	return strings.Contains(errText, "not found") ||
		strings.Contains(errText, "expired") ||
		strings.Contains(errText, "field not found") ||
		strings.Contains(errText, "redis: nil")
}
