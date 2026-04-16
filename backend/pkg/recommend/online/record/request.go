package record

import (
	"encoding/json"
	"time"

	"shop/pkg/gen/models"
	recommendDomain "shop/pkg/recommend/domain"
)

// MarshalPersistedSourceContext 序列化推荐请求主表要持久化的来源上下文。
func MarshalPersistedSourceContext(sourceContext map[string]any) (string, error) {
	persistedSourceContext := BuildPersistedSourceContext(sourceContext)
	sourceContextJson, err := json.Marshal(persistedSourceContext)
	if err != nil {
		return "", err
	}
	return string(sourceContextJson), nil
}

// BuildRecommendRequestEntity 构建推荐请求主表模型。
func BuildRecommendRequestEntity(requestId string, actor *recommendDomain.Actor, request *recommendDomain.GoodsRequest, sourceContext map[string]any, createdAt time.Time) (*models.RecommendRequest, error) {
	sourceContextJson, err := MarshalPersistedSourceContext(sourceContext)
	if err != nil {
		return nil, err
	}

	entity := &models.RecommendRequest{
		RequestID:     requestId,
		SourceContext: sourceContextJson,
		CreatedAt:     createdAt,
	}
	// 当前存在推荐主体时，继续回写主体类型和主体编号。
	if actor != nil {
		entity.ActorType = actor.ActorType
		entity.ActorID = actor.ActorId
	}
	// 当前存在推荐请求时，继续回写场景和分页信息。
	if request != nil {
		entity.Scene = int32(request.Scene)
		entity.PageNum = int32(request.PageNum)
		entity.PageSize = int32(request.PageSize)
	}
	return entity, nil
}
