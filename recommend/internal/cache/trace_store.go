package cache

import (
	"errors"

	recommendv1 "recommend/api/gen/go/recommend/v1"
	"recommend/internal/cache/driver"
	cacheleveldb "recommend/internal/cache/leveldb"

	"google.golang.org/protobuf/proto"
)

// TraceStore 负责管理推荐追踪缓存。
type TraceStore struct {
	Driver driver.Provider
}

// SaveTraceDetail 保存追踪详情，并按请求编号建立回查索引。
func (s *TraceStore) SaveTraceDetail(traceId, requestId string, detail *recommendv1.RecommendTraceDetail) error {
	rawValue, err := cacheleveldb.EncodeMessage(detail)
	if err != nil {
		return err
	}

	writer, err := s.batchWriter()
	if err != nil {
		return err
	}
	writer.Put(cacheleveldb.TraceDetailKey(traceId), rawValue)
	// 请求编号存在时，同时写入请求编号到追踪详情的索引。
	if requestId != "" {
		writer.Put(cacheleveldb.TraceByRequestKey(requestId), rawValue)
	}
	return writer.Write()
}

// GetTraceDetail 通过追踪编号读取追踪详情。
func (s *TraceStore) GetTraceDetail(traceId string) (*recommendv1.RecommendTraceDetail, error) {
	detail := &recommendv1.RecommendTraceDetail{}
	err := s.loadMessage(cacheleveldb.TraceDetailKey(traceId), detail)
	if err != nil {
		return nil, err
	}
	return detail, nil
}

// GetTraceDetailByRequestId 通过请求编号读取追踪详情。
func (s *TraceStore) GetTraceDetailByRequestId(requestId string) (*recommendv1.RecommendTraceDetail, error) {
	detail := &recommendv1.RecommendTraceDetail{}
	err := s.loadMessage(cacheleveldb.TraceByRequestKey(requestId), detail)
	if err != nil {
		return nil, err
	}
	return detail, nil
}

// DeleteTraceDetail 删除追踪详情以及请求编号索引。
func (s *TraceStore) DeleteTraceDetail(traceId, requestId string) error {
	writer, err := s.batchWriter()
	if err != nil {
		return err
	}
	writer.Delete(cacheleveldb.TraceDetailKey(traceId))
	// 请求编号存在时，需要同步删除回查索引。
	if requestId != "" {
		writer.Delete(cacheleveldb.TraceByRequestKey(requestId))
	}
	return writer.Write()
}

// traceDb 返回追踪数据库实例。
func (s *TraceStore) traceDb() driver.BinaryStore {
	if s != nil && s.Driver != nil {
		return s.Driver.TraceStore()
	}
	return nil
}

// batchWriter 返回追踪缓存使用的批量写入器。
func (s *TraceStore) batchWriter() (driver.BatchWriter, error) {
	if s == nil || s.Driver == nil {
		return nil, errors.New("recommend: 追踪数据库未初始化")
	}
	return s.Driver.OpenBatch(driver.ScopeTrace)
}

// loadMessage 从追踪数据库加载 proto 消息。
func (s *TraceStore) loadMessage(key []byte, message proto.Message) error {
	db := s.traceDb()
	// 追踪数据库未绑定时，说明 store 还没有正确初始化。
	if db == nil {
		return errors.New("recommend: 追踪数据库未初始化")
	}
	rawValue, err := db.Get(key)
	if err != nil {
		return err
	}
	return cacheleveldb.DecodeMessage(rawValue, message)
}
