package domain

import (
	"fmt"
	"hash/fnv"
	"math"
	"strings"
)

const publishGrayBucketScale = 10000

// PublishVersionResolution 表示一次发布配置在运行时解析出的版本决策结果。
type PublishVersionResolution struct {
	BaselineVersion    string  // 当前流量未命中灰度时使用的基线版本。
	GrayVersion        string  // 当前流量命中灰度时使用的灰度版本。
	EffectiveVersion   string  // 当前请求最终命中的实际版本。
	GrayRatio          float64 // 当前运行时实际采用的灰度比例。
	GrayEnabled        bool    // 当前是否启用了真实灰度放量。
	GrayHit            bool    // 当前请求是否命中灰度版本。
	GrayBucket         int     // 当前主体命中的稳定分桶编号。
	GrayBucketResolved bool    // 当前是否成功解析出稳定分桶编号。
}

// ResolveVersionResolution 基于主体信息解析当前请求的实际生效版本。
func (c *PublishStrategy) ResolveVersionResolution(scene int32, defaultVersion string, actor *Actor) PublishVersionResolution {
	resolution := PublishVersionResolution{
		BaselineVersion:  strings.TrimSpace(defaultVersion),
		GrayVersion:      strings.TrimSpace(defaultVersion),
		EffectiveVersion: strings.TrimSpace(defaultVersion),
		GrayRatio:        normalizePublishGrayRatio(0),
	}
	if c == nil {
		return resolution
	}

	grayRatio := normalizePublishGrayRatio(c.GrayRatio)
	baselineVersion := strings.TrimSpace(defaultVersion)
	// 当前显式配置了回滚版本时，把它作为灰度未命中时的基线版本。
	if strings.TrimSpace(c.RollbackVersion) != "" {
		baselineVersion = strings.TrimSpace(c.RollbackVersion)
	}
	grayVersion := strings.TrimSpace(defaultVersion)
	// 当前显式配置了缓存版本时，把它作为灰度命中时的目标版本。
	if strings.TrimSpace(c.CacheVersion) != "" {
		grayVersion = strings.TrimSpace(c.CacheVersion)
	}

	resolution.BaselineVersion = baselineVersion
	resolution.GrayVersion = grayVersion
	resolution.EffectiveVersion = baselineVersion
	resolution.GrayRatio = grayRatio

	// 灰度比例未启用或两套版本相同时，直接回退到基线版本即可。
	if grayRatio <= 0 || baselineVersion == "" || grayVersion == "" || baselineVersion == grayVersion {
		return resolution
	}
	resolution.GrayEnabled = true

	bucket, ok := resolvePublishGrayBucket(scene, actor)
	// 当前主体没有稳定标识时，不参与灰度放量，避免同一用户跨请求漂移。
	if !ok {
		return resolution
	}
	resolution.GrayBucket = bucket
	resolution.GrayBucketResolved = true

	if float64(bucket) < grayRatio*publishGrayBucketScale {
		resolution.GrayHit = true
		resolution.EffectiveVersion = grayVersion
	}
	return resolution
}

// normalizePublishGrayRatio 统一收口灰度比例，避免脏数据进入运行时分桶。
func normalizePublishGrayRatio(grayRatio float64) float64 {
	if grayRatio < 0 {
		return 0
	}
	if grayRatio > 1 {
		return 1
	}
	return math.Round(grayRatio*10000) / 10000
}

// resolvePublishGrayBucket 计算当前主体在某个场景下命中的稳定分桶编号。
func resolvePublishGrayBucket(scene int32, actor *Actor) (int, bool) {
	// 缺少稳定主体编号时，不参与灰度分桶。
	if actor == nil || actor.ActorId <= 0 {
		return 0, false
	}
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(fmt.Sprintf("scene:%d:actor_type:%d:actor_id:%d", scene, actor.ActorType, actor.ActorId)))
	return int(hasher.Sum32() % publishGrayBucketScale), true
}
