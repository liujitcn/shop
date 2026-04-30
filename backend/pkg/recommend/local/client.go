package local

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	_const "shop/pkg/const"

	"shop/pkg/gen/data"

	_slice "github.com/liujitcn/go-utils/slice"
	"github.com/liujitcn/gorm-kit/repository"
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

type rankedGoodsCandidate struct {
	goodsID     int64
	realSaleNum int64
	createdAt   time.Time
	score       float64
}

type goodsStatSummaryRow struct {
	GoodsID      int64 `gorm:"column:goods_id"`
	ViewCount    int64 `gorm:"column:view_count"`
	CollectCount int64 `gorm:"column:collect_count"`
	CartCount    int64 `gorm:"column:cart_count"`
	OrderCount   int64 `gorm:"column:order_count"`
	PayCount     int64 `gorm:"column:pay_count"`
	PayGoodsNum  int64 `gorm:"column:pay_goods_num"`
}

// Recommend 表示本地推荐基础客户端。
type Recommend struct {
	goodsInfoRepo    *data.GoodsInfoRepository
	goodsStatDayRepo *data.GoodsStatDayRepository
}

// NewRecommend 创建本地推荐基础客户端。
func NewRecommend(goodsInfoRepo *data.GoodsInfoRepository, goodsStatDayRepo *data.GoodsStatDayRepository) *Recommend {
	return &Recommend{
		goodsInfoRepo:    goodsInfoRepo,
		goodsStatDayRepo: goodsStatDayRepo,
	}
}

// Enabled 判断当前本地推荐基础客户端是否可用。
func (r *Recommend) Enabled() bool {
	return r != nil && r.goodsInfoRepo != nil && r.goodsStatDayRepo != nil
}

// ListCategoryIDsByGoodsIDs 查询商品编号对应的分类编号列表。
func (r *Recommend) ListCategoryIDsByGoodsIDs(ctx context.Context, goodsIDs []int64) ([]int64, error) {
	categoryJSONList := make([]string, 0)
	// 没有上下文商品时，无需继续查询分类。
	if len(goodsIDs) == 0 {
		return []int64{}, nil
	}

	query := r.goodsInfoRepo.Query(ctx).GoodsInfo
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.DeletedAt.IsNull()))
	opts = append(opts, repository.Where(query.ID.In(goodsIDs...)))
	queryDo := query.WithContext(ctx)
	err := repository.ApplyQueryOptions(&queryDo.DO, opts...).Pluck(query.CategoryID, &categoryJSONList)
	if err != nil {
		return nil, err
	}

	categoryIDs := make([]int64, 0)
	for _, rawCategoryIDs := range categoryJSONList {
		categoryIDs = append(categoryIDs, r.parseCategoryIDs(rawCategoryIDs)...)
	}
	return _slice.Unique(categoryIDs), nil
}

// ListRankedGoodsPage 按热度权重查询本地推荐商品分页结果。
func (r *Recommend) ListRankedGoodsPage(
	ctx context.Context,
	categoryIDs []int64,
	excludedGoodsIDs []int64,
	statDays int,
	scoreWeight localScoreWeight,
	pageNum, pageSize int64,
) ([]int64, int64, error) {
	goodsIDs := make([]int64, 0)
	// 页码或每页条数非法时，不再继续查询热度结果。
	if pageNum <= 0 || pageSize <= 0 {
		return []int64{}, 0, nil
	}

	// 统计窗口非法时，统一回退到 30 天窗口。
	if statDays <= 0 {
		statDays = 30
	}
	now := time.Now()
	endAt := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	startAt := endAt.AddDate(0, 0, -statDays)
	goodsQuery := r.goodsInfoRepo.Query(ctx).GoodsInfo
	dao := goodsQuery.WithContext(ctx).
		Where(
			goodsQuery.DeletedAt.IsNull(),
			goodsQuery.Status.Eq(_const.GOODS_STATUS_PUT_ON),
		)
	// 存在上下文商品时，从候选池里排除当前上下文商品。
	if len(excludedGoodsIDs) > 0 {
		dao = dao.Where(goodsQuery.ID.NotIn(excludedGoodsIDs...))
	}

	goodsList, err := dao.Find()
	if err != nil {
		return nil, 0, err
	}
	categoryIDSet := make(map[int64]struct{}, len(categoryIDs))
	for _, categoryID := range categoryIDs {
		categoryIDSet[categoryID] = struct{}{}
	}

	candidates := make([]*rankedGoodsCandidate, 0, len(goodsList))
	for _, goods := range goodsList {
		// 存在分类约束时，商品分类必须命中任一上下文分类。
		if len(categoryIDSet) > 0 {
			matchedCategory := false
			for _, categoryID := range r.parseCategoryIDs(goods.CategoryID) {
				// 任一分类命中时即可纳入推荐候选池。
				if _, ok := categoryIDSet[categoryID]; ok {
					matchedCategory = true
					break
				}
			}
			if !matchedCategory {
				continue
			}
		}
		candidates = append(candidates, &rankedGoodsCandidate{
			goodsID:     goods.ID,
			realSaleNum: goods.RealSaleNum,
			createdAt:   goods.CreatedAt,
		})
	}
	// 候选商品为空时，直接返回空分页。
	if len(candidates) == 0 {
		return []int64{}, 0, nil
	}

	statSummaryMap, err := r.loadGoodsStatSummaryMap(ctx, candidates, startAt, endAt)
	if err != nil {
		return nil, 0, err
	}
	r.sortRankedGoodsCandidates(candidates, statSummaryMap, scoreWeight)

	total := int64(len(candidates))
	pageOffset := int((pageNum - 1) * pageSize)
	// 当前页已经超出候选池时，返回空列表并保留总数。
	if pageOffset >= len(candidates) {
		return []int64{}, total, nil
	}
	pageEnd := pageOffset + int(pageSize)
	// 最后一页不足一页时，截断到候选池末尾。
	if pageEnd > len(candidates) {
		pageEnd = len(candidates)
	}
	for _, item := range candidates[pageOffset:pageEnd] {
		goodsIDs = append(goodsIDs, item.goodsID)
	}
	return goodsIDs, total, nil
}

// ListExploreGoodsPage 查询探索曝光候选池商品分页结果。
func (r *Recommend) ListExploreGoodsPage(
	ctx context.Context,
	excludedGoodsIDs []int64,
	seed, pageNum, pageSize int64,
) ([]int64, int64, error) {
	goodsIDs := make([]int64, 0)
	// 页码或每页条数非法时，不再继续查询探索结果。
	if pageNum <= 0 || pageSize <= 0 {
		return []int64{}, 0, nil
	}

	query := r.goodsInfoRepo.Query(ctx).GoodsInfo
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Where(query.DeletedAt.IsNull()))
	opts = append(opts, repository.Where(query.Status.Eq(_const.GOODS_STATUS_PUT_ON)))
	// 存在上下文商品时，从探索池里排除当前上下文商品。
	if len(excludedGoodsIDs) > 0 {
		opts = append(opts, repository.Where(query.ID.NotIn(excludedGoodsIDs...)))
	}
	opts = append(opts, repository.Order(
		query.ID.Mul(131).Add(seed).Mod(1000003).Asc(),
		query.CreatedAt.Desc(),
		query.ID.Desc(),
	))
	queryDo := query.WithContext(ctx)
	dao := repository.ApplyQueryOptions(&queryDo.DO, opts...)
	pageOffset := int((pageNum - 1) * pageSize)
	total, err := dao.Count()
	if err != nil {
		return nil, 0, err
	}
	goodsIDs, err = r.scanSelectedGoodsIDs(dao, query.ID, pageOffset, int(pageSize))
	if err != nil {
		return nil, 0, err
	}
	return goodsIDs, total, nil
}

// loadGoodsStatSummaryMap 查询候选商品在统计窗口内的行为聚合。
func (r *Recommend) loadGoodsStatSummaryMap(
	ctx context.Context,
	candidates []*rankedGoodsCandidate,
	startAt, endAt time.Time,
) (map[int64]*goodsStatSummaryRow, error) {
	goodsIDs := make([]int64, 0, len(candidates))
	for _, item := range candidates {
		goodsIDs = append(goodsIDs, item.goodsID)
	}

	rows := make([]*goodsStatSummaryRow, 0)
	query := r.goodsStatDayRepo.Query(ctx).GoodsStatDay
	err := query.WithContext(ctx).
		Select(
			query.GoodsID,
			query.ViewCount.Sum().IfNull(0).As("view_count"),
			query.CollectCount.Sum().IfNull(0).As("collect_count"),
			query.CartCount.Sum().IfNull(0).As("cart_count"),
			query.OrderCount.Sum().IfNull(0).As("order_count"),
			query.PayCount.Sum().IfNull(0).As("pay_count"),
			query.PayGoodsNum.Sum().IfNull(0).As("pay_goods_num"),
		).
		Where(
			query.GoodsID.In(goodsIDs...),
			query.StatDate.Gte(startAt),
			query.StatDate.Lt(endAt),
		).
		Group(query.GoodsID).
		Scan(&rows)
	if err != nil {
		return nil, err
	}

	res := make(map[int64]*goodsStatSummaryRow, len(rows))
	for _, row := range rows {
		res[row.GoodsID] = row
	}
	return res, nil
}

// sortRankedGoodsCandidates 按热度分和稳定兜底字段排序候选商品。
func (r *Recommend) sortRankedGoodsCandidates(
	candidates []*rankedGoodsCandidate,
	statSummaryMap map[int64]*goodsStatSummaryRow,
	scoreWeight localScoreWeight,
) {
	for _, item := range candidates {
		summary := statSummaryMap[item.goodsID]
		// 没有统计数据的商品按 0 分参与后续兜底排序。
		if summary == nil {
			continue
		}
		item.score = float64(summary.ViewCount)*scoreWeight.viewWeight +
			float64(summary.CollectCount)*scoreWeight.collectWeight +
			float64(summary.CartCount)*scoreWeight.cartWeight +
			float64(summary.OrderCount)*scoreWeight.orderWeight +
			float64(summary.PayCount)*scoreWeight.payWeight +
			float64(summary.PayGoodsNum)*scoreWeight.payGoodsWeight
	}

	sort.Slice(candidates, func(i, j int) bool {
		// 热度分不同时，优先按热度分降序。
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		// 热度分相同时，按真实销量降序兜底。
		if candidates[i].realSaleNum != candidates[j].realSaleNum {
			return candidates[i].realSaleNum > candidates[j].realSaleNum
		}
		// 销量相同时，按创建时间倒序兜底。
		if !candidates[i].createdAt.Equal(candidates[j].createdAt) {
			return candidates[i].createdAt.After(candidates[j].createdAt)
		}
		return candidates[i].goodsID > candidates[j].goodsID
	})
}

// scanSelectedGoodsIDs 扫描单列商品编号查询结果。
func (r *Recommend) scanSelectedGoodsIDs(dao gen.Dao, goodsIDField field.Int64, pageOffset, pageSize int) ([]int64, error) {
	goodsIDs := make([]int64, 0)
	rows, err := dao.Select(goodsIDField).Offset(pageOffset).Limit(pageSize).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		goodsID := int64(0)
		err = rows.Scan(&goodsID)
		if err != nil {
			return nil, err
		}
		goodsIDs = append(goodsIDs, goodsID)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return goodsIDs, nil
}

// parseCategoryIDs 解析商品分类编号列表。
func (r *Recommend) parseCategoryIDs(rawCategoryIDs string) []int64 {
	categoryIDs := make([]int64, 0)
	// 分类字段为空时，直接返回空分类列表。
	if strings.TrimSpace(rawCategoryIDs) == "" {
		return categoryIDs
	}

	err := json.Unmarshal([]byte(rawCategoryIDs), &categoryIDs)
	// 分类 JSON 解析失败时，回退为空分类列表，避免单条脏数据影响推荐链路。
	if err != nil {
		return []int64{}
	}
	return categoryIDs
}
