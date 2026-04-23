package local

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"shop/api/gen/go/common"
	"shop/pkg/gen/data"

	_slice "github.com/liujitcn/go-utils/slice"
	"github.com/liujitcn/gorm-kit/repo"
	"gorm.io/datatypes"
	"gorm.io/gen"
	"gorm.io/gen/field"
)

type localScoreWeight struct {
	viewWeight     float64
	collectWeight  float64
	cartWeight     float64
	orderWeight    float64
	payWeight      float64
	payGoodsWeight float64
}

// Recommend 表示本地推荐基础客户端。
type Recommend struct {
	goodsInfoRepo    *data.GoodsInfoRepo
	goodsStatDayRepo *data.GoodsStatDayRepo
}

// NewRecommend 创建本地推荐基础客户端。
func NewRecommend(goodsInfoRepo *data.GoodsInfoRepo, goodsStatDayRepo *data.GoodsStatDayRepo) *Recommend {
	return &Recommend{
		goodsInfoRepo:    goodsInfoRepo,
		goodsStatDayRepo: goodsStatDayRepo,
	}
}

// Enabled 判断当前本地推荐基础客户端是否可用。
func (r *Recommend) Enabled() bool {
	return r != nil && r.goodsInfoRepo != nil && r.goodsStatDayRepo != nil
}

// ListCategoryIdsByGoodsIds 查询商品编号对应的分类编号列表。
func (r *Recommend) ListCategoryIdsByGoodsIds(ctx context.Context, goodsIds []int64) ([]int64, error) {
	categoryJsonList := make([]string, 0)
	// 没有上下文商品时，无需继续查询分类。
	if len(goodsIds) == 0 {
		return []int64{}, nil
	}

	query := r.goodsInfoRepo.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, 2)
	opts = append(opts, repo.Where(query.DeletedAt.IsNull()))
	opts = append(opts, repo.Where(query.ID.In(goodsIds...)))
	queryDo := query.WithContext(ctx)
	err := repo.ApplyQueryOptions(&queryDo.DO, opts...).Pluck(query.CategoryID, &categoryJsonList)
	if err != nil {
		return nil, err
	}

	categoryIds := make([]int64, 0)
	for _, rawCategoryIds := range categoryJsonList {
		categoryIds = append(categoryIds, r.parseCategoryIds(rawCategoryIds)...)
	}
	return _slice.Unique(categoryIds), nil
}

// ListRankedGoodsPage 按热度权重查询本地推荐商品分页结果。
func (r *Recommend) ListRankedGoodsPage(
	ctx context.Context,
	categoryIds []int64,
	excludedGoodsIds []int64,
	statDays int,
	scoreWeight localScoreWeight,
	pageNum, pageSize int64,
) ([]int64, int64, error) {
	goodsIds := make([]int64, 0)
	// 页码或每页条数非法时，不再继续查询热度结果。
	if pageNum <= 0 || pageSize <= 0 {
		return []int64{}, 0, nil
	}

	startAt, endAt := r.buildStatRange(statDays)
	goodsQuery := r.goodsInfoRepo.Query(ctx).GoodsInfo
	statQuery := r.goodsStatDayRepo.Query(ctx).GoodsStatDay
	opts := make([]repo.QueryOption, 0, 8)
	opts = append(opts, repo.LeftJoin(
		statQuery,
		statQuery.GoodsID.EqCol(goodsQuery.ID),
		statQuery.DeletedAt.IsNull(),
		statQuery.StatDate.Gte(startAt),
		statQuery.StatDate.Lt(endAt),
	))
	opts = append(opts, repo.Where(goodsQuery.DeletedAt.IsNull()))
	opts = append(opts, repo.Where(goodsQuery.Status.Eq(int32(common.GoodsStatus_PUT_ON))))

	// 存在分类约束时，只在相关类目范围内挑选商品。
	if len(categoryIds) > 0 {
		categoryIdsJson, marshalErr := json.Marshal(categoryIds)
		if marshalErr != nil {
			return nil, 0, marshalErr
		}
		opts = append(opts, repo.Where(gen.Cond(datatypes.JSONOverlaps(goodsQuery.CategoryID, string(categoryIdsJson)))...))
	}
	// 存在上下文商品时，从候选池里排除当前上下文商品。
	if len(excludedGoodsIds) > 0 {
		opts = append(opts, repo.Where(goodsQuery.ID.NotIn(excludedGoodsIds...)))
	}
	opts = append(opts, repo.Group(goodsQuery.ID, goodsQuery.RealSaleNum, goodsQuery.CreatedAt))
	opts = append(opts, repo.Order(
		r.buildScoreExpr(
			statQuery.ViewCount,
			statQuery.CollectCount,
			statQuery.CartCount,
			statQuery.OrderCount,
			statQuery.PayCount,
			statQuery.PayGoodsNum,
			scoreWeight,
		).Desc(),
		goodsQuery.RealSaleNum.Desc(),
		goodsQuery.CreatedAt.Desc(),
		goodsQuery.ID.Desc(),
	))
	goodsDo := goodsQuery.WithContext(ctx)
	dao := repo.ApplyQueryOptions(&goodsDo.DO, opts...)
	pageOffset := int((pageNum - 1) * pageSize)
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	goodsIds, err = r.scanSelectedGoodsIds(dao, goodsQuery.ID, pageOffset, int(pageSize))
	if err != nil {
		return nil, 0, err
	}
	return goodsIds, total, nil
}

// ListExploreGoodsPage 查询探索曝光候选池商品分页结果。
func (r *Recommend) ListExploreGoodsPage(
	ctx context.Context,
	excludedGoodsIds []int64,
	seed, pageNum, pageSize int64,
) ([]int64, int64, error) {
	goodsIds := make([]int64, 0)
	// 页码或每页条数非法时，不再继续查询探索结果。
	if pageNum <= 0 || pageSize <= 0 {
		return []int64{}, 0, nil
	}

	query := r.goodsInfoRepo.Query(ctx).GoodsInfo
	opts := make([]repo.QueryOption, 0, 5)
	opts = append(opts, repo.Where(query.DeletedAt.IsNull()))
	opts = append(opts, repo.Where(query.Status.Eq(int32(common.GoodsStatus_PUT_ON))))
	// 存在上下文商品时，从探索池里排除当前上下文商品。
	if len(excludedGoodsIds) > 0 {
		opts = append(opts, repo.Where(query.ID.NotIn(excludedGoodsIds...)))
	}
	opts = append(opts, repo.Order(
		r.buildExploreExpr(query.ID, seed).Asc(),
		query.CreatedAt.Desc(),
		query.ID.Desc(),
	))
	queryDo := query.WithContext(ctx)
	dao := repo.ApplyQueryOptions(&queryDo.DO, opts...)
	pageOffset := int((pageNum - 1) * pageSize)
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	goodsIds, err = r.scanSelectedGoodsIds(dao, query.ID, pageOffset, int(pageSize))
	if err != nil {
		return nil, 0, err
	}
	return goodsIds, total, nil
}

// buildStatRange 构建本地热度统计时间窗口。
func (r *Recommend) buildStatRange(statDays int) (time.Time, time.Time) {
	// 统计窗口非法时，统一回退到 30 天窗口。
	if statDays <= 0 {
		statDays = 30
	}
	now := time.Now()
	endAt := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	startAt := endAt.AddDate(0, 0, -statDays)
	return startAt, endAt
}

// buildRotationSeed 构建探索候选池轮转种子。
func (r *Recommend) buildRotationSeed(scene common.RecommendScene, requestId int64) int64 {
	seed := requestId*131 + int64(scene)*17
	// 轮转种子需要保持非负，避免取模后出现负序。
	if seed < 0 {
		seed = -seed
	}
	return seed % 1000003
}

// buildScoreExpr 构建热度排序表达式。
func (r *Recommend) buildScoreExpr(
	viewCount field.Int64,
	collectCount field.Int64,
	cartCount field.Int64,
	orderCount field.Int64,
	payCount field.Int64,
	payGoodsNum field.Int64,
	scoreWeight localScoreWeight,
) field.Field {
	// 热度得分依赖聚合加权表达式，这里仅保留必要的原生函数组合，不再手写字段名字符串。
	return field.NewUnsafeFieldRaw(
		"(COALESCE(SUM(?), 0) * ? + COALESCE(SUM(?), 0) * ? + COALESCE(SUM(?), 0) * ? + COALESCE(SUM(?), 0) * ? + COALESCE(SUM(?), 0) * ? + COALESCE(SUM(?), 0) * ?)",
		viewCount,
		scoreWeight.viewWeight,
		collectCount,
		scoreWeight.collectWeight,
		cartCount,
		scoreWeight.cartWeight,
		orderCount,
		scoreWeight.orderWeight,
		payCount,
		scoreWeight.payWeight,
		payGoodsNum,
		scoreWeight.payGoodsWeight,
	)
}

// buildExploreExpr 构建探索候选池排序表达式。
func (r *Recommend) buildExploreExpr(goodsId field.Int64, seed int64) field.Int64 {
	return goodsId.Mul(131).Add(seed).Mod(1000003)
}

// scanSelectedGoodsIds 扫描单列商品编号查询结果。
func (r *Recommend) scanSelectedGoodsIds(dao gen.Dao, goodsIdField field.Int64, pageOffset, pageSize int) ([]int64, error) {
	goodsIds := make([]int64, 0)
	rows, err := dao.Select(goodsIdField).Offset(pageOffset).Limit(pageSize).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		goodsId := int64(0)
		err = rows.Scan(&goodsId)
		if err != nil {
			return nil, err
		}
		goodsIds = append(goodsIds, goodsId)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return goodsIds, nil
}

// parseCategoryIds 解析商品分类编号列表。
func (r *Recommend) parseCategoryIds(rawCategoryIds string) []int64 {
	categoryIds := make([]int64, 0)
	// 分类字段为空时，直接返回空分类列表。
	if strings.TrimSpace(rawCategoryIds) == "" {
		return categoryIds
	}

	err := json.Unmarshal([]byte(rawCategoryIds), &categoryIds)
	// 分类 JSON 解析失败时，回退为空分类列表，避免单条脏数据影响推荐链路。
	if err != nil {
		return []int64{}
	}
	return categoryIds
}
