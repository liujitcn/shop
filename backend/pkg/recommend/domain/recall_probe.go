package domain

// RecallProbeItem 表示单条召回探针配置。
type RecallProbeItem struct {
	Enabled       bool  `json:"enabled"`        // 是否启用当前探针。
	JoinCandidate bool  `json:"join_candidate"` // 是否允许把探针结果并入默认候选池。
	Limit         int64 `json:"limit"`          // 当前探针的读取数量上限。
}

// ResolveLimit 返回探针最终使用的读取数量。
func (c *RecallProbeItem) ResolveLimit(defaultLimit int64) int64 {
	// 配置为空或数量非法时，回退到调用方给出的默认值。
	if c == nil || c.Limit <= 0 {
		return defaultLimit
	}
	return c.Limit
}

// ShouldJoinCandidate 判断当前探针结果是否允许并入候选池。
func (c *RecallProbeItem) ShouldJoinCandidate() bool {
	// 只有探针启用且显式允许并入候选池时，才参与默认候选构建。
	if c == nil {
		return false
	}
	return c.Enabled && c.JoinCandidate
}

// RecallProbeStrategy 表示阶段 4 的召回探针配置。
type RecallProbeStrategy struct {
	SimilarUser            *RecallProbeItem `json:"similar_user"`            // 相似用户召回探针。
	CollaborativeFiltering *RecallProbeItem `json:"collaborative_filtering"` // 协同过滤召回探针。
	ContentBased           *RecallProbeItem `json:"content_based"`           // 内容相似召回探针。
}

// HasEnabledProbe 判断当前是否存在已启用的召回探针。
func (c *RecallProbeStrategy) HasEnabledProbe() bool {
	// 探针配置为空时，说明当前版本没有开启任何探针。
	if c == nil {
		return false
	}
	return c.IsSimilarUserEnabled() || c.IsCollaborativeFilteringEnabled() || c.IsContentBasedEnabled()
}

// IsSimilarUserEnabled 判断是否启用了相似用户探针。
func (c *RecallProbeStrategy) IsSimilarUserEnabled() bool {
	return c != nil && c.SimilarUser != nil && c.SimilarUser.Enabled
}

// IsCollaborativeFilteringEnabled 判断是否启用了协同过滤探针。
func (c *RecallProbeStrategy) IsCollaborativeFilteringEnabled() bool {
	return c != nil && c.CollaborativeFiltering != nil && c.CollaborativeFiltering.Enabled
}

// IsContentBasedEnabled 判断是否启用了内容相似探针。
func (c *RecallProbeStrategy) IsContentBasedEnabled() bool {
	return c != nil && c.ContentBased != nil && c.ContentBased.Enabled
}
