package cache

import (
	"strconv"
	"strings"
)

const (
	// DefaultVersion 表示未显式指定版本时的默认缓存版本标识。
	DefaultVersion = "default"
	// defaultKeyPrefix 表示推荐缓存统一键前缀。
	defaultKeyPrefix = "recommend"
	// subsetKindHot 表示场景热门榜子集合。
	subsetKindHot = "hot"
	// subsetKindLatest 表示场景最新榜子集合。
	subsetKindLatest = "latest"
)

// CollectionKey 返回带统一前缀的推荐缓存集合名。
func CollectionKey(collection string) string {
	return Key(defaultKeyPrefix, sanitizeKeyPart(collection))
}

// SceneHotSubset 返回场景热门榜子集合键。
func SceneHotSubset(scene int32, version string) string {
	return sceneSubset(subsetKindHot, scene, version)
}

// SceneLatestSubset 返回场景最新榜子集合键。
func SceneLatestSubset(scene int32, version string) string {
	return sceneSubset(subsetKindLatest, scene, version)
}

// SimilarItemSubset 返回相似商品子集合键。
func SimilarItemSubset(goodsId int64, version string) string {
	return Key("goods", strconv.FormatInt(goodsId, 10), "version", NormalizeVersion(version))
}

// SimilarUserSubset 返回相似用户子集合键。
func SimilarUserSubset(userId int64, version string) string {
	return Key("user", strconv.FormatInt(userId, 10), "version", NormalizeVersion(version))
}

// CollaborativeFilteringSubset 返回协同过滤子集合键。
func CollaborativeFilteringSubset(userId int64, version string) string {
	return Key("user", strconv.FormatInt(userId, 10), "version", NormalizeVersion(version))
}

// ContentBasedSubset 返回内容相似子集合键。
func ContentBasedSubset(goodsId int64, version string) string {
	return Key("goods", strconv.FormatInt(goodsId, 10), "version", NormalizeVersion(version))
}

// DigestKey 返回当前集合子集合的摘要键。
func DigestKey(collection, subset string) string {
	return metadataKey(collection, subset, "digest")
}

// UpdateTimeKey 返回当前集合子集合的更新时间键。
func UpdateTimeKey(collection, subset string) string {
	return metadataKey(collection, subset, "update_time")
}

// DocumentCountKey 返回当前集合子集合的文档数量键。
func DocumentCountKey(collection, subset string) string {
	return metadataKey(collection, subset, "document_count")
}

// ScoreSubsetIndexKey 返回当前集合的子集合索引键。
func ScoreSubsetIndexKey(collection string) string {
	return Key(collection, GlobalMeta, scoreSubsetIndexSuffix)
}

// NormalizeVersion 统一归一化缓存版本号。
func NormalizeVersion(version string) string {
	normalized := strings.TrimSpace(version)
	if normalized == "" {
		return DefaultVersion
	}
	return sanitizeKeyPart(normalized)
}

// sceneSubset 返回场景级榜单子集合键。
func sceneSubset(kind string, scene int32, version string) string {
	return Key("scene", strconv.Itoa(int(scene)), "kind", sanitizeKeyPart(kind), "version", NormalizeVersion(version))
}

// metadataKey 返回当前集合子集合的元信息键。
func metadataKey(collection, subset, suffix string) string {
	return Key(CollectionKey(collection), subset, sanitizeKeyPart(suffix))
}

// sanitizeKeyPart 统一清洗缓存键片段，避免层级冲突。
func sanitizeKeyPart(part string) string {
	replaced := strings.TrimSpace(part)
	replaced = strings.ReplaceAll(replaced, "/", "_")
	replaced = strings.ReplaceAll(replaced, ":", "_")
	if replaced == "" {
		return DefaultVersion
	}
	return replaced
}
