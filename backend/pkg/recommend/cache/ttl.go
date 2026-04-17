package cache

import "time"

// ExpireScoreSubset 为排序型缓存子集合及其元信息设置统一过期时间。
func ExpireScoreSubset(store Store, collection, subset string, ttl time.Duration) error {
	// 存储未初始化、集合为空、子集合为空或 TTL 非法时，不继续设置过期时间。
	if store == nil || collection == "" || subset == "" || ttl <= 0 {
		return nil
	}

	keyList := []string{
		ScoreHashKey(collection, subset),
		DigestKey(collection, subset),
		DocumentCountKey(collection, subset),
		UpdateTimeKey(collection, subset),
	}
	for _, key := range keyList {
		// 当前键为空时，直接跳过，避免向底层缓存写入非法过期操作。
		if key == "" {
			continue
		}
		err := store.Expire(key, ttl)
		if err != nil {
			return err
		}
	}
	return nil
}
