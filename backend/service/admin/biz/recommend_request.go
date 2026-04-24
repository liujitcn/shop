package biz

import (
	"context"
	"encoding/json"
	"strconv"

	"shop/api/gen/go/admin"
	"shop/api/gen/go/common"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	recommendDto "shop/pkg/recommend/dto"

	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repo"
)

// RecommendRequestCase 推荐请求管理业务实例。
type RecommendRequestCase struct {
	*biz.BaseCase
	*data.RecommendRequestRepo
	baseUserRepo             *data.BaseUserRepo
	recommendRequestItemCase *RecommendRequestItemCase
	recommendEventCase       *RecommendEventCase
}

// NewRecommendRequestCase 创建推荐请求管理业务实例。
func NewRecommendRequestCase(
	baseCase *biz.BaseCase,
	recommendRequestRepo *data.RecommendRequestRepo,
	baseUserRepo *data.BaseUserRepo,
	recommendRequestItemCase *RecommendRequestItemCase,
	recommendEventCase *RecommendEventCase,
) *RecommendRequestCase {
	return &RecommendRequestCase{
		BaseCase:                 baseCase,
		RecommendRequestRepo:     recommendRequestRepo,
		baseUserRepo:             baseUserRepo,
		recommendRequestItemCase: recommendRequestItemCase,
		recommendEventCase:       recommendEventCase,
	}
}

// PageRecommendRequest 分页查询推荐请求。
func (c *RecommendRequestCase) PageRecommendRequest(ctx context.Context, req *admin.PageRecommendRequestRequest) (*admin.PageRecommendRequestResponse, error) {
	// 请求为空时，回退到默认查询条件，避免分页逻辑读取空指针。
	if req == nil {
		req = &admin.PageRecommendRequestRequest{}
	}

	query := c.RecommendRequestRepo.Query(ctx).RecommendRequest
	opts := make([]repo.QueryOption, 0, 8)
	opts = append(opts, repo.Order(query.RequestAt.Desc()))
	opts = append(opts, repo.Order(query.ID.Desc()))

	// 传入推荐请求编号时，仅查询指定推荐会话的请求记录。
	if req.RequestId != nil && req.GetRequestId() != "" {
		requestId, parseErr := strconv.ParseInt(req.GetRequestId(), 10, 64)
		// 推荐请求编号解析成功时，才继续按推荐会话筛选请求记录。
		if parseErr == nil && requestId > 0 {
			opts = append(opts, repo.Where(query.RequestID.Eq(requestId)))
		}
	}
	// 传入主体类型时，仅查询指定主体类型的请求记录。
	if req.ActorType != nil && req.GetActorType() > common.RecommendActorType_UNKNOWN_RAT {
		opts = append(opts, repo.Where(query.ActorType.Eq(int32(req.GetActorType()))))
	}
	// 传入主体编号时，仅查询指定主体的请求记录。
	if req.ActorId != nil && req.GetActorId() > 0 {
		opts = append(opts, repo.Where(query.ActorID.Eq(req.GetActorId())))
	}
	// 传入推荐场景时，仅查询指定场景的请求记录。
	if req.Scene != nil && req.GetScene() != common.RecommendScene_UNKNOWN_RS {
		opts = append(opts, repo.Where(query.Scene.Eq(int32(req.GetScene()))))
	}

	requestTime := req.GetRequestAt()
	// 仅在传入完整请求时间区间时，按时间范围筛选请求记录。
	if len(requestTime) == 2 {
		startTime := _time.StringTimeToTime(requestTime[0])
		endTime := _time.StringTimeToTime(requestTime[1])
		// 开始时间解析成功时，补充请求时间下界。
		if startTime != nil {
			opts = append(opts, repo.Where(query.RequestAt.Gte(*startTime)))
		}
		// 结束时间解析成功时，补充请求时间上界。
		if endTime != nil {
			opts = append(opts, repo.Where(query.RequestAt.Lt(endTime.AddDate(0, 0, 1))))
		}
	}

	list, total, err := c.RecommendRequestRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	actorNameMap, err := c.getRecommendActorNameMap(ctx, list)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.RecommendRequest, 0, len(list))
	for _, item := range list {
		resList = append(resList, c.toRecommendRequest(item, actorNameMap[item.ActorID]))
	}
	return &admin.PageRecommendRequestResponse{
		List:  resList,
		Total: int32(total),
	}, nil
}

// GetRecommendRequest 查询推荐请求详情。
func (c *RecommendRequestCase) GetRecommendRequest(ctx context.Context, id int64) (*admin.RecommendRequestDetailResponse, error) {
	// 请求记录编号非法时，无法定位推荐请求详情。
	if id <= 0 {
		return nil, errorsx.InvalidArgument("推荐请求记录编号不能为空")
	}

	requestModel, err := c.RecommendRequestRepo.FindById(ctx, id)
	if err != nil {
		return nil, err
	}

	contextRecord := c.parseRecommendContext(requestModel.ContextJSON)
	actorNameMap, err := c.getRecommendActorNameMap(ctx, []*models.RecommendRequest{requestModel})
	if err != nil {
		return nil, err
	}
	itemList, err := c.recommendRequestItemCase.ListRecommendRequestItems(ctx, requestModel)
	if err != nil {
		return nil, err
	}

	return &admin.RecommendRequestDetailResponse{
		Request:  c.toRecommendRequest(requestModel, actorNameMap[requestModel.ActorID]),
		Context:  c.toRecommendRequestContext(contextRecord, requestModel.ContextJSON),
		ItemList: itemList,
	}, nil
}

// GetRecommendRequestEvent 查询推荐请求商品关联事件。
func (c *RecommendRequestCase) GetRecommendRequestEvent(
	ctx context.Context,
	requestRecordId int64,
	goodsId int64,
	position int32,
) (*admin.GetRecommendRequestEventResponse, error) {
	// 请求记录编号非法时，无法定位推荐请求事件范围。
	if requestRecordId <= 0 {
		return nil, errorsx.InvalidArgument("推荐请求记录编号不能为空")
	}
	// 商品编号非法时，无法定位推荐请求事件范围。
	if goodsId <= 0 {
		return nil, errorsx.InvalidArgument("商品编号不能为空")
	}

	requestModel, err := c.RecommendRequestRepo.FindById(ctx, requestRecordId)
	if err != nil {
		return nil, err
	}

	return c.recommendEventCase.GetRecommendRequestEvent(ctx, requestModel.RequestID, goodsId, position)
}

// toRecommendRequest 转换推荐请求分页响应数据。
func (c *RecommendRequestCase) toRecommendRequest(item *models.RecommendRequest, actorName string) *admin.RecommendRequest {
	// 请求实体为空时，回退到空响应结构，避免列表渲染空指针。
	if item == nil {
		return &admin.RecommendRequest{}
	}

	contextRecord := c.parseRecommendContext(item.ContextJSON)

	return &admin.RecommendRequest{
		Id:           item.ID,
		RequestId:    strconv.FormatInt(item.RequestID, 10),
		ActorType:    common.RecommendActorType(item.ActorType),
		ActorId:      item.ActorID,
		Scene:        common.RecommendScene(item.Scene),
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
	rawJson string,
) *admin.RecommendRequestContext {
	// 上下文为空时，回退到空结构，避免详情页读取空指针。
	if contextRecord == nil {
		contextRecord = &recommendDto.RecommendContext{}
	}

	finalProviderName := c.resolveFinalProviderName(contextRecord)
	traceList := make([]*admin.RecommendRequestTrace, 0, len(contextRecord.Trace))
	for _, item := range contextRecord.Trace {
		// 链路节点为空时，直接跳过无效节点。
		if item == nil {
			continue
		}
		traceList = append(traceList, &admin.RecommendRequestTrace{
			ProviderName: item.ProviderName,
			ResultCount:  int32(item.ResultCount),
			Hit:          item.Hit,
			ErrorMsg:     item.ErrorMsg,
			IsFinal:      finalProviderName != "" && item.ProviderName == finalProviderName,
		})
	}

	return &admin.RecommendRequestContext{
		GoodsId:           contextRecord.GoodsId,
		OrderId:           contextRecord.OrderId,
		ContextGoodsIds:   append([]int64(nil), contextRecord.ContextGoodsIds...),
		Strategy:          contextRecord.Strategy,
		ProviderName:      contextRecord.ProviderName,
		FinalProviderName: finalProviderName,
		Trace:             traceList,
		RawJson:           rawJson,
	}
}

// parseRecommendContext 解析推荐上下文JSON。
func (c *RecommendRequestCase) parseRecommendContext(rawJson string) *recommendDto.RecommendContext {
	contextRecord := &recommendDto.RecommendContext{
		Trace: make([]*recommendDto.GoodsTrace, 0),
	}
	// 原始上下文为空时，直接返回空结构，兼容历史空数据。
	if rawJson == "" {
		return contextRecord
	}
	// 上下文解析失败时，保留空结构回退，避免旧数据阻塞管理端查看。
	if json.Unmarshal([]byte(rawJson), contextRecord) != nil {
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

	userIdSet := make(map[int64]struct{}, len(requestList))
	userIds := make([]int64, 0, len(requestList))
	for _, item := range requestList {
		// 仅登录用户主体需要回查主体信息，匿名主体统一走固定文案。
		if item == nil || item.ActorType != int32(common.RecommendActorType_USER_ACTOR) || item.ActorID <= 0 {
			continue
		}
		if _, ok := userIdSet[item.ActorID]; ok {
			continue
		}
		userIdSet[item.ActorID] = struct{}{}
		userIds = append(userIds, item.ActorID)
	}
	// 当前页没有登录用户主体时，无需继续查询主体信息。
	if len(userIds) == 0 {
		return actorNameMap, nil
	}

	query := c.baseUserRepo.Query(ctx).BaseUser
	list, err := c.baseUserRepo.List(ctx, repo.Where(query.ID.In(userIds...)))
	if err != nil {
		return nil, err
	}

	for _, item := range list {
		// 用户为空或编号非法时，直接跳过无效记录。
		if item == nil || item.ID <= 0 {
			continue
		}
		actorNameMap[item.ID] = c.resolveRecommendActorName(item)
	}
	return actorNameMap, nil
}

// resolveRecommendActorName 解析推荐主体名称。
func (c *RecommendRequestCase) resolveRecommendActorName(user *models.BaseUser) string {
	// 用户为空时，不存在可展示的姓名。
	if user == nil {
		return ""
	}
	// 优先使用用户昵称，保证后台列表展示更贴近业务侧认知。
	if user.NickName != "" {
		return user.NickName
	}
	return user.UserName
}
