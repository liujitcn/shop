package biz

import (
	"context"
	"encoding/json"
	"strconv"

	_const "shop/pkg/const"

	adminv1 "shop/api/gen/go/admin/v1"
	commonv1 "shop/api/gen/go/common/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendDto "shop/pkg/recommend/dto"

	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repository"
)

// RecommendRequestCase 推荐请求管理业务实例。
type RecommendRequestCase struct {
	*biz.BaseCase
	*data.RecommendRequestRepository
	baseUserRepo             *data.BaseUserRepository
	recommendRequestItemCase *RecommendRequestItemCase
	recommendEventCase       *RecommendEventCase
}

// NewRecommendRequestCase 创建推荐请求管理业务实例。
func NewRecommendRequestCase(
	baseCase *biz.BaseCase,
	recommendRequestRepo *data.RecommendRequestRepository,
	baseUserRepo *data.BaseUserRepository,
	recommendRequestItemCase *RecommendRequestItemCase,
	recommendEventCase *RecommendEventCase,
) *RecommendRequestCase {
	return &RecommendRequestCase{
		BaseCase:                   baseCase,
		RecommendRequestRepository: recommendRequestRepo,
		baseUserRepo:               baseUserRepo,
		recommendRequestItemCase:   recommendRequestItemCase,
		recommendEventCase:         recommendEventCase,
	}
}

// PageRecommendRequests 分页查询推荐请求。
func (c *RecommendRequestCase) PageRecommendRequests(ctx context.Context, req *adminv1.PageRecommendRequestsRequest) (*adminv1.PageRecommendRequestsResponse, error) {
	// 请求为空时，回退到默认查询条件，避免分页逻辑读取空指针。
	if req == nil {
		req = &adminv1.PageRecommendRequestsRequest{}
	}

	query := c.RecommendRequestRepository.Query(ctx).RecommendRequest
	opts := make([]repository.QueryOption, 0, 8)
	opts = append(opts, repository.Order(query.RequestAt.Desc()))
	opts = append(opts, repository.Order(query.ID.Desc()))

	// 传入推荐请求编号时，仅查询指定推荐会话的请求记录。
	if req.RequestId != nil && req.GetRequestId() != "" {
		requestID, parseErr := strconv.ParseInt(req.GetRequestId(), 10, 64)
		// 推荐请求编号解析成功时，才继续按推荐会话筛选请求记录。
		if parseErr == nil && requestID > 0 {
			opts = append(opts, repository.Where(query.RequestID.Eq(requestID)))
		}
	}
	// 传入主体类型时，仅查询指定主体类型的请求记录。
	if req.ActorType != nil && req.GetActorType() > commonv1.RecommendActorType(_const.RECOMMEND_ACTOR_TYPE_UNKNOWN) {
		opts = append(opts, repository.Where(query.ActorType.Eq(int32(req.GetActorType()))))
	}
	// 传入主体编号时，仅查询指定主体的请求记录。
	if req.ActorId != nil && req.GetActorId() > 0 {
		opts = append(opts, repository.Where(query.ActorID.Eq(req.GetActorId())))
	}
	// 传入推荐场景时，仅查询指定场景的请求记录。
	if req.Scene != nil && req.GetScene() != commonv1.RecommendScene(_const.RECOMMEND_SCENE_UNKNOWN) {
		opts = append(opts, repository.Where(query.Scene.Eq(int32(req.GetScene()))))
	}

	requestTime := req.GetRequestAt()
	// 仅在传入完整请求时间区间时，按时间范围筛选请求记录。
	if len(requestTime) == 2 {
		startTime := _time.StringTimeToTime(requestTime[0])
		endTime := _time.StringTimeToTime(requestTime[1])
		// 开始时间解析成功时，补充请求时间下界。
		if startTime != nil {
			opts = append(opts, repository.Where(query.RequestAt.Gte(*startTime)))
		}
		// 结束时间解析成功时，补充请求时间上界。
		if endTime != nil {
			opts = append(opts, repository.Where(query.RequestAt.Lt(endTime.AddDate(0, 0, 1))))
		}
	}

	list, total, err := c.RecommendRequestRepository.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	actorNameMap := make(map[int64]string)
	actorNameMap, err = c.getRecommendActorNameMap(ctx, list)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.RecommendRequest, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.toRecommendRequest(item, actorNameMap[item.ActorID]))
	}
	return &adminv1.PageRecommendRequestsResponse{
		RecommendRequests: resList,
		Total:             int32(total),
	}, nil
}

// GetRecommendRequest 查询推荐请求详情。
func (c *RecommendRequestCase) GetRecommendRequest(ctx context.Context, id int64) (*adminv1.RecommendRequestDetailResponse, error) {
	// 请求记录编号非法时，无法定位推荐请求详情。
	if id <= 0 {
		return nil, errorsx.InvalidArgument("推荐请求记录编号不能为空")
	}

	requestModel, err := c.RecommendRequestRepository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	contextRecord := c.parseRecommendContext(requestModel.ContextJSON)
	actorNameMap := make(map[int64]string)
	actorNameMap, err = c.getRecommendActorNameMap(ctx, []*models.RecommendRequest{requestModel})
	if err != nil {
		return nil, err
	}
	itemList := make([]*adminv1.RecommendRequestItem, 0)
	itemList, err = c.recommendRequestItemCase.ListRecommendRequestItems(ctx, requestModel)
	if err != nil {
		return nil, err
	}

	return &adminv1.RecommendRequestDetailResponse{
		Request:  c.toRecommendRequest(requestModel, actorNameMap[requestModel.ActorID]),
		Context:  c.toRecommendRequestContext(contextRecord, requestModel.ContextJSON),
		ItemList: itemList,
	}, nil
}

// ListRecommendRequestEvents 查询推荐请求商品关联事件列表。
func (c *RecommendRequestCase) ListRecommendRequestEvents(
	ctx context.Context,
	requestRecordID int64,
	goodsID int64,
	position int32,
) (*adminv1.ListRecommendRequestEventsResponse, error) {
	// 请求记录编号非法时，无法定位推荐请求事件范围。
	if requestRecordID <= 0 {
		return nil, errorsx.InvalidArgument("推荐请求记录编号不能为空")
	}
	// 商品编号非法时，无法定位推荐请求事件范围。
	if goodsID <= 0 {
		return nil, errorsx.InvalidArgument("商品编号不能为空")
	}

	requestModel, err := c.RecommendRequestRepository.FindByID(ctx, requestRecordID)
	if err != nil {
		return nil, err
	}

	return c.recommendEventCase.ListRecommendRequestEvents(ctx, requestModel.RequestID, goodsID, position)
}

// toRecommendRequest 转换推荐请求分页响应数据。
func (c *RecommendRequestCase) toRecommendRequest(item *models.RecommendRequest, actorName string) *adminv1.RecommendRequest {
	// 请求实体为空时，回退到空响应结构，避免列表渲染空指针。
	if item == nil {
		return &adminv1.RecommendRequest{}
	}

	contextRecord := c.parseRecommendContext(item.ContextJSON)

	return &adminv1.RecommendRequest{
		Id:           item.ID,
		RequestId:    strconv.FormatInt(item.RequestID, 10),
		ActorType:    commonv1.RecommendActorType(item.ActorType),
		ActorId:      item.ActorID,
		Scene:        commonv1.RecommendScene(item.Scene),
		PageNum:      item.PageNum,
		PageSize:     item.PageSize,
		Total:        item.Total,
		Strategy:     contextRecord.Strategy,
		ProviderName: c.resolveFinalProviderName(contextRecord),
		RequestAt:    _time.TimeToTimeString(item.RequestAt),
		ActorName:    actorName,
	}
}

// toRecommendRequestContext 转换推荐上下文响应数据。
func (c *RecommendRequestCase) toRecommendRequestContext(
	contextRecord *recommendDto.RecommendContext,
	rawJSON string,
) *adminv1.RecommendRequestContext {
	// 上下文为空时，回退到空结构，避免详情页读取空指针。
	if contextRecord == nil {
		contextRecord = &recommendDto.RecommendContext{}
	}

	finalProviderName := c.resolveFinalProviderName(contextRecord)
	traceList := make([]*adminv1.RecommendRequestTrace, 0, len(contextRecord.Trace))
	for _, item := range contextRecord.Trace {
		// 链路节点为空时，直接跳过无效节点。
		if item == nil {
			continue
		}
		traceList = append(traceList, &adminv1.RecommendRequestTrace{
			ProviderName: item.ProviderName,
			ResultCount:  int32(item.ResultCount),
			Hit:          item.Hit,
			ErrorMsg:     item.ErrorMsg,
			IsFinal:      finalProviderName != "" && item.ProviderName == finalProviderName,
		})
	}

	return &adminv1.RecommendRequestContext{
		GoodsId:           contextRecord.GoodsID,
		OrderId:           contextRecord.OrderID,
		ContextGoodsIds:   append([]int64(nil), contextRecord.ContextGoodsIDs...),
		Strategy:          contextRecord.Strategy,
		ProviderName:      contextRecord.ProviderName,
		FinalProviderName: finalProviderName,
		Trace:             traceList,
		RawJson:           rawJSON,
	}
}

// parseRecommendContext 解析推荐上下文JSON。
func (c *RecommendRequestCase) parseRecommendContext(rawJSON string) *recommendDto.RecommendContext {
	contextRecord := &recommendDto.RecommendContext{
		Trace: make([]*recommendDto.GoodsTrace, 0),
	}
	// 原始上下文为空时，直接返回空结构，兼容历史空数据。
	if rawJSON == "" {
		return contextRecord
	}
	// 上下文解析失败时，保留空结构回退，避免旧数据阻塞管理端查看。
	if json.Unmarshal([]byte(rawJSON), contextRecord) != nil {
		return contextRecord
	}
	// 解析后的轨迹为空时，统一补齐空切片，避免前端收到 null。
	if contextRecord.Trace == nil {
		contextRecord.Trace = make([]*recommendDto.GoodsTrace, 0)
	}
	return contextRecord
}

// resolveFinalProviderName 解析最终命中的推荐器名称。
func (c *RecommendRequestCase) resolveFinalProviderName(contextRecord *recommendDto.RecommendContext) string {
	// 上下文为空时，不存在可解析的最终推荐器。
	if contextRecord == nil {
		return ""
	}
	// 上下文已显式记录推荐器时，优先使用该值作为最终推荐器。
	if contextRecord.ProviderName != "" {
		return contextRecord.ProviderName
	}
	for _, item := range contextRecord.Trace {
		// 链路节点为空时，直接跳过无效节点。
		if item == nil {
			continue
		}
		// 链路命中结果时，将当前节点视为最终命中的推荐器。
		if item.Hit && item.ProviderName != "" {
			return item.ProviderName
		}
	}
	return ""
}

// getRecommendActorNameMap 构建推荐主体名称映射。
func (c *RecommendRequestCase) getRecommendActorNameMap(
	ctx context.Context,
	requestList []*models.RecommendRequest,
) (map[int64]string, error) {
	actorNameMap := make(map[int64]string)
	// 请求列表为空时，无需继续查询主体信息。
	if len(requestList) == 0 {
		return actorNameMap, nil
	}

	userIDSet := make(map[int64]struct{}, len(requestList))
	userIDs := make([]int64, 0, len(requestList))
	for _, item := range requestList {
		// 仅登录用户主体需要回查主体信息，匿名主体统一走固定文案。
		if item == nil || item.ActorType != _const.RECOMMEND_ACTOR_TYPE_USER || item.ActorID <= 0 {
			continue
		}
		if _, ok := userIDSet[item.ActorID]; ok {
			continue
		}
		userIDSet[item.ActorID] = struct{}{}
		userIDs = append(userIDs, item.ActorID)
	}
	// 当前页没有登录用户主体时，无需继续查询主体信息。
	if len(userIDs) == 0 {
		return actorNameMap, nil
	}

	query := c.baseUserRepo.Query(ctx).BaseUser
	list, err := c.baseUserRepo.List(ctx, repository.Where(query.ID.In(userIDs...)))
	if err != nil {
		return nil, err
	}

	for _, item := range list {
		// 用户为空或编号非法时，直接跳过无效记录。
		if item == nil || item.ID <= 0 {
			continue
		}
		actorName := item.UserName
		// 优先使用用户昵称，保证后台列表展示更贴近业务侧认知。
		if item.NickName != "" {
			actorName = item.NickName
		}
		actorNameMap[item.ID] = actorName
	}
	return actorNameMap, nil
}
