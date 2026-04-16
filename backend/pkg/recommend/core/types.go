package core

import recommendDomain "shop/pkg/recommend/domain"

// Candidate 兼容旧引用，实际类型已下沉到领域层维护。
// 阶段 6 在线主链路完成切换后，可统一搜索 “阶段 6 后删除：兼容旧引用” 清理本别名。
// 阶段 6 后删除：兼容旧引用。
type Candidate = recommendDomain.Candidate

// ScoreDetail 兼容旧引用，实际类型已下沉到领域层维护。
// 阶段 6 在线主链路完成切换后，可统一搜索 “阶段 6 后删除：兼容旧引用” 清理本别名。
// 阶段 6 后删除：兼容旧引用。
type ScoreDetail = recommendDomain.ScoreDetail
