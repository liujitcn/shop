package cache

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	kitcache "github.com/liujitcn/kratos-kit/cache"
)

const (
	// NonPersonalized 表示非个性化推荐集合。
	NonPersonalized = "non-personalized"
	// NonPersonalizedDigest 表示非个性化推荐摘要键前缀。
	NonPersonalizedDigest = "non-personalized_digest"
	// NonPersonalizedUpdateTime 表示非个性化推荐更新时间键前缀。
	NonPersonalizedUpdateTime = "non-personalized_update_time"
	// ItemToItem 表示商品相似推荐集合。
	ItemToItem = "item-to-item"
	// ItemToItemDigest 表示商品相似推荐摘要键前缀。
	ItemToItemDigest = "item-to-item_digest"
	// ItemToItemUpdateTime 表示商品相似推荐更新时间键前缀。
	ItemToItemUpdateTime = "item-to-item_update_time"
	// UserToUser 表示相似用户推荐集合。
	UserToUser = "user-to-user"
	// UserToUserDigest 表示相似用户推荐摘要键前缀。
	UserToUserDigest = "user-to-user_digest"
	// UserToUserUpdateTime 表示相似用户推荐更新时间键前缀。
	UserToUserUpdateTime = "user-to-user_update_time"
	// CollaborativeFiltering 表示协同过滤推荐集合。
	CollaborativeFiltering = "collaborative-filtering"
	// CollaborativeFilteringDigest 表示协同过滤推荐摘要键前缀。
	CollaborativeFilteringDigest = "collaborative-filtering_digest"
	// CollaborativeFilteringUpdateTime 表示协同过滤推荐更新时间键前缀。
	CollaborativeFilteringUpdateTime = "collaborative-filtering_update_time"
	// ContentBased 表示内容相似推荐集合。
	ContentBased = "content-based"
	// ContentBasedDigest 表示内容相似推荐摘要键前缀。
	ContentBasedDigest = "content-based_digest"
	// ContentBasedUpdateTime 表示内容相似推荐更新时间键前缀。
	ContentBasedUpdateTime = "content-based_update_time"
	// Recommend 表示最终推荐结果集合。
	Recommend = "recommend"
	// RecommendDigest 表示最终推荐结果摘要键前缀。
	RecommendDigest = "recommend_digest"
	// RecommendUpdateTime 表示最终推荐结果更新时间键前缀。
	RecommendUpdateTime = "recommend_update_time"
	// GlobalMeta 表示全局元信息集合。
	GlobalMeta = "global_meta"
)

var (
	// ErrObjectNotExist 表示缓存对象不存在。
	ErrObjectNotExist = errors.New("recommend cache object not exist")
	// ErrNoStore 表示当前未配置推荐缓存实现。
	ErrNoStore = errors.New("recommend cache store not configured")
)

// Key 按推荐缓存约定拼接层级键，空片段会被自动跳过。
func Key(keys ...string) string {
	if len(keys) == 0 {
		return ""
	}
	var builder strings.Builder
	firstWritten := false
	for _, key := range keys {
		if key == "" {
			continue
		}
		if firstWritten {
			builder.WriteRune('/')
		}
		builder.WriteString(key)
		firstWritten = true
	}
	return builder.String()
}

// Value 表示一个普通 KV 值。
type Value struct {
	Name  string // 缓存键名
	Value string // 缓存值
}

// String 创建字符串值对象。
func String(name, value string) Value {
	return Value{Name: name, Value: value}
}

// Integer 创建整数值对象。
func Integer(name string, value int) Value {
	return Value{Name: name, Value: strconv.Itoa(value)}
}

// Time 创建时间值对象。
func Time(name string, value time.Time) Value {
	return Value{Name: name, Value: value.Format(time.RFC3339Nano)}
}

// ReturnValue 表示缓存读取结果。
type ReturnValue struct {
	Value  string // 原始缓存值
	Err    error  // 读取过程中的异常
	Exists bool   // 缓存是否存在
}

// String 读取字符串返回值。
func (r *ReturnValue) String() (string, error) {
	// 返回对象为空时，统一返回空值，避免调用方继续判空。
	if r == nil {
		return "", nil
	}
	// 读取阶段出现异常时，优先返回原始错误。
	if r.Err != nil {
		return "", r.Err
	}
	// 缓存不存在时，统一返回空字符串。
	if !r.Exists {
		return "", nil
	}
	return r.Value, nil
}

// Integer 读取整数返回值。
func (r *ReturnValue) Integer() (int, error) {
	// 返回对象为空时，统一回退到零值。
	if r == nil {
		return 0, nil
	}
	// 读取阶段出现异常时，优先返回原始错误。
	if r.Err != nil {
		return 0, r.Err
	}
	// 缓存不存在时，统一回退到零值。
	if !r.Exists {
		return 0, nil
	}
	return strconv.Atoi(r.Value)
}

// Time 读取时间返回值。
func (r *ReturnValue) Time() (time.Time, error) {
	// 返回对象为空时，统一回退到零时间。
	if r == nil {
		return time.Time{}, nil
	}
	// 读取阶段出现异常时，优先返回原始错误。
	if r.Err != nil {
		return time.Time{}, r.Err
	}
	// 缓存不存在或值为空时，统一回退到零时间。
	if !r.Exists || r.Value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, r.Value)
}

// Score 表示排序型推荐缓存中的单个元素。
type Score struct {
	Id         string    // 排序条目主键
	Score      float64   // 排序分数
	IsHidden   bool      // 是否对外隐藏
	Categories []string  // 条目所属分类集合
	Timestamp  time.Time // 条目更新时间
}

// SortDocuments 按分数从高到低稳定排序缓存条目。
func SortDocuments(documents []Score) {
	sort.SliceStable(documents, func(i, j int) bool {
		return documents[i].Score > documents[j].Score
	})
}

// ConvertDocumentsToValues 提取缓存条目的主键列表。
func ConvertDocumentsToValues(documents []Score) []string {
	values := make([]string, 0, len(documents))
	for _, item := range documents {
		values = append(values, item.Id)
	}
	return values
}

// ScoreCondition 表示批量删除或更新时的过滤条件。
type ScoreCondition struct {
	Subset *string    // 子集合过滤条件
	Id     *string    // 主键过滤条件
	Before *time.Time // 时间上界过滤条件
}

// Check 校验当前分数过滤条件是否有效。
func (c *ScoreCondition) Check() error {
	// 过滤条件为空时，无法安全执行批量删除或更新。
	if c == nil {
		return errors.New("recommend cache score condition is nil")
	}
	// 三类过滤条件都为空时，拒绝继续执行，避免误删全量数据。
	if c.Subset == nil && c.Id == nil && c.Before == nil {
		return errors.New("recommend cache score condition is empty")
	}
	return nil
}

// ScorePatch 表示排序型推荐缓存的增量更新字段。
type ScorePatch struct {
	IsHidden   *bool    // 是否隐藏更新项
	Categories []string // 分类集合更新项
	Score      *float64 // 分数更新项
}

// Store 定义推荐专用缓存的通用接口。
// 普通 KV / Hash 能力直接复用 kratos-kit/cache，推荐层只额外补排序集合能力。
type Store interface {
	kitcache.Cache
	Close() error
	Ping() error
	AddScores(ctx context.Context, collection, subset string, documents []Score) error
	SearchScores(ctx context.Context, collection, subset string, begin, end int) ([]Score, error)
	DeleteScores(ctx context.Context, collections []string, condition ScoreCondition) error
	UpdateScores(ctx context.Context, collections []string, subset *string, id string, patch ScorePatch) error
}
