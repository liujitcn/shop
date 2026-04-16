package task

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// recommendMaterializeStats 表示写缓存任务的最小可观测摘要。
type recommendMaterializeStats struct {
	taskName               string
	limit                  int64
	startedAt              time.Time
	currentStage           string
	versionSet             map[string]struct{}
	inputCountMap          map[string]int
	publishedSubsetCount   int
	publishedDocumentCount int
	clearedSubsetCount     int
}

// newRecommendMaterializeStats 创建写缓存任务摘要统计器。
func newRecommendMaterializeStats(taskName string, limit int64) *recommendMaterializeStats {
	return &recommendMaterializeStats{
		taskName:      taskName,
		limit:         limit,
		startedAt:     time.Now(),
		versionSet:    make(map[string]struct{}),
		inputCountMap: make(map[string]int),
	}
}

// SetStage 记录任务当前执行阶段。
func (s *recommendMaterializeStats) SetStage(stage string) {
	// 统计器为空或阶段名为空时，不更新当前阶段。
	if s == nil || stage == "" {
		return
	}
	s.currentStage = stage
}

// AddVersion 记录任务本次触达的缓存版本。
func (s *recommendMaterializeStats) AddVersion(version string) {
	// 版本为空时，不记录到摘要统计中。
	if s == nil || version == "" {
		return
	}
	s.versionSet[version] = struct{}{}
}

// AddInputCount 记录训练或写缓存任务的输入规模。
func (s *recommendMaterializeStats) AddInputCount(name string, count int) {
	// 统计器为空、指标名为空或数量非法时，不继续累计输入规模。
	if s == nil || name == "" || count <= 0 {
		return
	}
	s.inputCountMap[name] += count
}

// AddPublishedSubset 记录一个已发布子集合及其文档数量。
func (s *recommendMaterializeStats) AddPublishedSubset(documentCount int) {
	// 统计器为空时，不继续累计发布结果。
	if s == nil {
		return
	}
	s.publishedSubsetCount++
	// 负数文档数量没有业务意义，按 0 处理。
	if documentCount <= 0 {
		return
	}
	s.publishedDocumentCount += documentCount
}

// AddClearedSubsets 记录清理失效子集合的数量。
func (s *recommendMaterializeStats) AddClearedSubsets(count int) {
	// 统计器为空或清理数量非法时，不继续累计。
	if s == nil || count <= 0 {
		return
	}
	s.clearedSubsetCount += count
}

// BuildSummary 构建当前任务的摘要字符串。
func (s *recommendMaterializeStats) BuildSummary() string {
	// 统计器为空时，直接返回空摘要，避免调用方判空。
	if s == nil {
		return ""
	}
	durationMs := time.Since(s.startedAt).Milliseconds()
	summary := fmt.Sprintf(
		"task=%s limit=%d versions=%d published_subsets=%d published_documents=%d cleared_subsets=%d duration_ms=%d",
		s.taskName,
		s.limit,
		len(s.versionSet),
		s.publishedSubsetCount,
		s.publishedDocumentCount,
		s.clearedSubsetCount,
		durationMs,
	)
	inputSummary := s.buildInputSummary()
	// 存在输入规模统计时，追加到统一摘要末尾。
	if inputSummary != "" {
		summary = summary + " " + inputSummary
	}
	return summary
}

// BuildFailureSummary 构建当前任务的失败摘要字符串。
func (s *recommendMaterializeStats) BuildFailureSummary(err error) string {
	// 统计器为空或错误为空时，不构建失败摘要。
	if s == nil || err == nil {
		return ""
	}
	summary := s.BuildSummary()
	if s.currentStage == "" {
		return summary + " status=failed error=" + err.Error()
	}
	return summary + " status=failed stage=" + s.currentStage + " error=" + err.Error()
}

// buildInputSummary 构建稳定排序的输入规模摘要。
func (s *recommendMaterializeStats) buildInputSummary() string {
	if s == nil || len(s.inputCountMap) == 0 {
		return ""
	}
	keyList := make([]string, 0, len(s.inputCountMap))
	for key := range s.inputCountMap {
		keyList = append(keyList, key)
	}
	sort.Strings(keyList)
	partList := make([]string, 0, len(keyList))
	for _, key := range keyList {
		partList = append(partList, fmt.Sprintf("%s=%d", key, s.inputCountMap[key]))
	}
	return "inputs:" + strings.Join(partList, ",")
}

// LogSummary 输出当前任务的摘要日志。
func (s *recommendMaterializeStats) LogSummary() {
	summary := s.BuildSummary()
	// 摘要为空时，不额外输出日志。
	if summary == "" {
		return
	}
	log.Infof("Job %s", summary)
}

// LogFailure 输出当前任务的失败日志。
func (s *recommendMaterializeStats) LogFailure(err error) {
	summary := s.BuildFailureSummary(err)
	// 失败摘要为空时，不额外输出日志。
	if summary == "" {
		return
	}
	log.Errorf("Job %s", summary)
}

// returnRecommendMaterializeFailure 统一输出失败摘要并返回任务错误。
func returnRecommendMaterializeFailure(stats *recommendMaterializeStats, err error) ([]string, error) {
	// 错误为空时，直接返回空结果。
	if err == nil {
		return nil, nil
	}
	if stats == nil {
		return []string{err.Error()}, err
	}
	stats.LogFailure(err)
	return []string{stats.BuildFailureSummary(err)}, err
}
