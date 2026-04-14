package leveldb

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"recommend/contract"
	"recommend/internal/cache/driver"

	goleveldb "github.com/syndtr/goleveldb/leveldb"
)

// Manager 管理推荐模块使用的三个 LevelDB 实例。
type Manager struct {
	layout       contract.LevelDbLayout
	poolDb       *goleveldb.DB
	runtimeDb    *goleveldb.DB
	traceDb      *goleveldb.DB
	poolStore    *binaryStore
	runtimeStore *binaryStore
	traceStore   *binaryStore
}

// binaryStore 封装单个 LevelDB 实例的基础读写能力。
type binaryStore struct {
	db *goleveldb.DB
}

// OpenManager 根据缓存配置打开推荐模块使用的三个 LevelDB 实例。
func OpenManager(ctx context.Context, cacheSource contract.CacheSource) (*Manager, error) {
	layout, err := cacheSource.RecommendLevelDb(ctx)
	if err != nil {
		return nil, err
	}
	return OpenManagerByLayout(layout)
}

// OpenManagerByLayout 根据文件路径直接打开推荐模块使用的三个 LevelDB 实例。
func OpenManagerByLayout(layout contract.LevelDbLayout) (*Manager, error) {
	err := validateLayout(layout)
	if err != nil {
		return nil, err
	}

	manager := &Manager{layout: layout}
	manager.poolDb, err = openDbFile(layout.PoolPath)
	if err != nil {
		return nil, err
	}

	manager.runtimeDb, err = openDbFile(layout.RuntimePath)
	if err != nil {
		closeErr := manager.Close()
		if closeErr != nil {
			return nil, fmt.Errorf("打开 runtime.db 失败: %w；关闭已打开实例失败: %v", err, closeErr)
		}
		return nil, err
	}

	manager.traceDb, err = openDbFile(layout.TracePath)
	if err != nil {
		closeErr := manager.Close()
		if closeErr != nil {
			return nil, fmt.Errorf("打开 trace.db 失败: %w；关闭已打开实例失败: %v", err, closeErr)
		}
		return nil, err
	}

	manager.poolStore = &binaryStore{db: manager.poolDb}
	manager.runtimeStore = &binaryStore{db: manager.runtimeDb}
	manager.traceStore = &binaryStore{db: manager.traceDb}

	return manager, nil
}

// Layout 返回当前管理器使用的文件布局。
func (m *Manager) Layout() contract.LevelDbLayout {
	if m == nil {
		return contract.LevelDbLayout{}
	}
	return m.layout
}

// PoolDb 返回候选池数据库实例。
func (m *Manager) PoolDb() *goleveldb.DB {
	if m == nil {
		return nil
	}
	return m.poolDb
}

// RuntimeDb 返回运行态数据库实例。
func (m *Manager) RuntimeDb() *goleveldb.DB {
	if m == nil {
		return nil
	}
	return m.runtimeDb
}

// TraceDb 返回追踪数据库实例。
func (m *Manager) TraceDb() *goleveldb.DB {
	if m == nil {
		return nil
	}
	return m.traceDb
}

// PoolStore 返回候选池缓存驱动。
func (m *Manager) PoolStore() driver.BinaryStore {
	if m == nil {
		return nil
	}
	return m.poolStore
}

// RuntimeStore 返回运行态缓存驱动。
func (m *Manager) RuntimeStore() driver.BinaryStore {
	if m == nil {
		return nil
	}
	return m.runtimeStore
}

// TraceStore 返回追踪缓存驱动。
func (m *Manager) TraceStore() driver.BinaryStore {
	if m == nil {
		return nil
	}
	return m.traceStore
}

// OpenBatch 根据缓存作用域创建批量写入器。
func (m *Manager) OpenBatch(scope driver.Scope) (driver.BatchWriter, error) {
	if m == nil {
		return nil, errors.New("recommend: LevelDB 管理器未初始化")
	}

	switch scope {
	// 候选池批量写入用于离线构建批量落缓存。
	case driver.ScopePool:
		return OpenBatchWriter(m.poolDb), nil
	// 运行态批量写入用于行为回传后同步更新状态。
	case driver.ScopeRuntime:
		return OpenBatchWriter(m.runtimeDb), nil
	// 追踪批量写入用于 explain 明细和回查索引同时写入。
	case driver.ScopeTrace:
		return OpenBatchWriter(m.traceDb), nil
	default:
		return nil, fmt.Errorf("recommend: 不支持的缓存作用域 %q", scope)
	}
}

// Close 关闭管理器持有的全部数据库实例。
func (m *Manager) Close() error {
	if m == nil {
		return nil
	}

	var err error
	err = closeDb(m.poolDb, err)
	err = closeDb(m.runtimeDb, err)
	err = closeDb(m.traceDb, err)
	return err
}

// validateLayout 校验推荐缓存路径配置。
func validateLayout(layout contract.LevelDbLayout) error {
	// 三个数据库路径缺一不可，否则后续缓存能力无法完整工作。
	if layout.PoolPath == "" {
		return errors.New("recommend: pool.db 路径不能为空")
	}
	// 运行态库缺失时，曝光惩罚和会话态无法持久化。
	if layout.RuntimePath == "" {
		return errors.New("recommend: runtime.db 路径不能为空")
	}
	// 追踪库缺失时，explain 无法回查。
	if layout.TracePath == "" {
		return errors.New("recommend: trace.db 路径不能为空")
	}
	return nil
}

// openDbFile 打开单个 LevelDB 文件。
func openDbFile(path string) (*goleveldb.DB, error) {
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		return nil, err
	}
	return goleveldb.OpenFile(path, nil)
}

// closeDb 按顺序关闭数据库，并保留第一个关闭失败的错误。
func closeDb(db *goleveldb.DB, firstErr error) error {
	if db == nil {
		return firstErr
	}

	err := db.Close()
	if err != nil && firstErr == nil {
		return err
	}
	return firstErr
}

// Put 写入一条缓存记录。
func (s *binaryStore) Put(key, value []byte) error {
	if s == nil || s.db == nil {
		return errors.New("recommend: LevelDB 实例未初始化")
	}
	return s.db.Put(key, value, nil)
}

// Get 读取一条缓存记录。
func (s *binaryStore) Get(key []byte) ([]byte, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("recommend: LevelDB 实例未初始化")
	}
	return s.db.Get(key, nil)
}

// Delete 删除一条缓存记录。
func (s *binaryStore) Delete(key []byte) error {
	if s == nil || s.db == nil {
		return errors.New("recommend: LevelDB 实例未初始化")
	}
	return s.db.Delete(key, nil)
}
