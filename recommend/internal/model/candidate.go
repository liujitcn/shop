package model

import (
	"recommend/contract"
	"sort"
)

// Candidate 表示推荐链路内部流转的候选商品。
type Candidate struct {
	Goods         *contract.Goods
	Score         Score
	RecallSources map[string]struct{}
	TraceReasons  []string
}

// BuildCandidate 根据商品实体创建候选对象。
func BuildCandidate(goods *contract.Goods) *Candidate {
	return &Candidate{
		Goods:         goods,
		RecallSources: make(map[string]struct{}, 4),
		TraceReasons:  make([]string, 0, 4),
	}
}

// GoodsId 返回候选商品编号。
func (c *Candidate) GoodsId() int64 {
	if c == nil || c.Goods == nil {
		return 0
	}
	return c.Goods.Id
}

// CategoryId 返回候选商品分类编号。
func (c *Candidate) CategoryId() int64 {
	if c == nil || c.Goods == nil {
		return 0
	}
	return c.Goods.CategoryId
}

// AddRecallSource 添加召回来源。
func (c *Candidate) AddRecallSource(source string) {
	// 空召回来源没有业务意义，不需要写入候选解释。
	if source == "" || c == nil {
		return
	}
	if c.RecallSources == nil {
		c.RecallSources = make(map[string]struct{}, 4)
	}
	c.RecallSources[source] = struct{}{}
}

// AddTraceReason 添加候选解释原因。
func (c *Candidate) AddTraceReason(reason string) {
	// 空解释原因没有业务意义，不需要写入候选解释。
	if reason == "" || c == nil {
		return
	}
	c.TraceReasons = append(c.TraceReasons, reason)
}

// RecallSourceList 返回稳定排序后的召回来源。
func (c *Candidate) RecallSourceList() []string {
	if c == nil || len(c.RecallSources) == 0 {
		return nil
	}

	list := make([]string, 0, len(c.RecallSources))
	for source := range c.RecallSources {
		list = append(list, source)
	}
	sort.Strings(list)
	return list
}
