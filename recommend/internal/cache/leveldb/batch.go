package leveldb

import (
	"errors"

	goleveldb "github.com/syndtr/goleveldb/leveldb"
)

// BatchWriter 封装 LevelDB 批量写入操作。
type BatchWriter struct {
	db    *goleveldb.DB
	batch *goleveldb.Batch
}

// OpenBatchWriter 创建一个批量写入器。
func OpenBatchWriter(db *goleveldb.DB) *BatchWriter {
	return &BatchWriter{
		db:    db,
		batch: &goleveldb.Batch{},
	}
}

// Put 向批量写入器加入一条写入记录。
func (w *BatchWriter) Put(key, value []byte) {
	if w == nil || w.batch == nil {
		return
	}
	w.batch.Put(key, value)
}

// Delete 向批量写入器加入一条删除记录。
func (w *BatchWriter) Delete(key []byte) {
	if w == nil || w.batch == nil {
		return
	}
	w.batch.Delete(key)
}

// Reset 清空当前批量写入内容。
func (w *BatchWriter) Reset() {
	if w == nil || w.batch == nil {
		return
	}
	w.batch.Reset()
}

// Write 将当前批量操作一次性提交到数据库。
func (w *BatchWriter) Write() error {
	// 批量写入器未绑定数据库时，不能执行提交。
	if w == nil || w.db == nil {
		return errors.New("recommend: 批量写入器未绑定数据库")
	}
	// 批量对象缺失时，说明写入器状态异常。
	if w.batch == nil {
		return errors.New("recommend: 批量写入器缺少批量对象")
	}
	return w.db.Write(w.batch, nil)
}
