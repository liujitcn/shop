package rank

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	recommendDomain "shop/pkg/recommend/domain"
)

// BuildRerankRequestHash 根据请求、主体和候选商品快照生成稳定哈希。
func BuildRerankRequestHash(
	request *recommendDomain.GoodsRequest,
	actor *recommendDomain.Actor,
	strategy *recommendDomain.LlmRerankStrategy,
	candidateGoodsIds []int64,
	topN int64,
) string {
	hasher := sha1.New()
	normalizedRequest := recommendDomain.GoodsRequest{}
	// 请求对象存在时，继续复制请求快照参与哈希。
	if request != nil {
		normalizedRequest = *request
	}
	actorType := int32(0)
	actorId := int64(0)
	// 主体存在时，再补充主体信息，避免匿名与登录态哈希冲突。
	if actor != nil {
		actorType = actor.ActorType
		actorId = actor.ActorId
	}
	_, _ = hasher.Write([]byte(fmt.Sprintf(
		"scene=%d|order=%d|goods=%d|page_num=%d|page_size=%d|actor_type=%d|actor_id=%d|",
		normalizedRequest.Scene,
		normalizedRequest.OrderId,
		normalizedRequest.GoodsId,
		normalizedRequest.PageNum,
		normalizedRequest.PageSize,
		actorType,
		actorId,
	)))

	normalizedGoodsIds := trimCandidateGoodsIds(candidateGoodsIds, topN)
	for _, goodsId := range normalizedGoodsIds {
		_, _ = hasher.Write([]byte(fmt.Sprintf("%d|", goodsId)))
	}
	// 当前存在 LLM 配置快照时，再把会影响输出的关键参数并入哈希。
	if strategyHash := buildRerankStrategyHash(strategy); strategyHash != "" {
		_, _ = hasher.Write([]byte("strategy=" + strategyHash))
	}
	hashValue := hex.EncodeToString(hasher.Sum(nil))
	// 哈希结果为空时，回退到固定占位值，避免缓存键片段缺失。
	if strings.TrimSpace(hashValue) == "" {
		return "empty"
	}
	return hashValue
}

// trimCandidateGoodsIds 按 TopN 裁剪候选商品快照。
func trimCandidateGoodsIds(candidateGoodsIds []int64, topN int64) []int64 {
	if len(candidateGoodsIds) == 0 {
		return []int64{}
	}
	if topN <= 0 || topN > int64(len(candidateGoodsIds)) {
		topN = int64(len(candidateGoodsIds))
	}
	return append([]int64(nil), candidateGoodsIds[:topN]...)
}

// buildRerankStrategyHash 生成会影响 LLM 输出的配置摘要。
func buildRerankStrategyHash(strategy *recommendDomain.LlmRerankStrategy) string {
	// 当前没有配置快照时，不额外参与请求哈希。
	if strategy == nil {
		return ""
	}
	payload, err := json.Marshal(map[string]any{
		"model":               strings.TrimSpace(strings.ToLower(strategy.Model)),
		"systemPrompt":        strings.TrimSpace(strategy.SystemPrompt),
		"promptTemplate":      strings.TrimSpace(strategy.PromptTemplate),
		"candidateFilterExpr": strings.TrimSpace(strategy.CandidateFilterExpr),
		"scoreExpr":           strings.TrimSpace(strategy.ScoreExpr),
		"scoreScript":         strings.TrimSpace(strategy.ScoreScript),
		"maxCompletionTokens": strategy.MaxCompletionTokens,
		"temperature":         strategy.Temperature,
	})
	// 配置快照无法序列化时，回退为不参与哈希，避免影响主链路。
	if err != nil {
		return ""
	}
	return string(payload)
}
