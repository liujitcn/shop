package driver

// Scope 表示缓存库作用域。
type Scope string

const (
	ScopePool    Scope = "pool"
	ScopeRuntime Scope = "runtime"
	ScopeTrace   Scope = "trace"
)

// BinaryStore 定义二进制缓存存储的基础读写能力。
type BinaryStore interface {
	Put(key, value []byte) error
	Get(key []byte) ([]byte, error)
	Delete(key []byte) error
}

// BatchWriter 定义批量写入能力。
type BatchWriter interface {
	Put(key, value []byte)
	Delete(key []byte)
	Reset()
	Write() error
}

// Provider 定义推荐缓存驱动需要提供的能力。
type Provider interface {
	PoolStore() BinaryStore
	RuntimeStore() BinaryStore
	TraceStore() BinaryStore
	OpenBatch(scope Scope) (BatchWriter, error)
	Close() error
}
