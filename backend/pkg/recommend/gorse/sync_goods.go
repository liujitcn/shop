package gorse

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	_const "shop/pkg/const"

	"shop/pkg/gen/models"

	client "github.com/gorse-io/gorse-go"
	_set "github.com/liujitcn/go-utils/set"
)

// GoodsSyncReceiver 表示商品主数据同步接收器。
type GoodsSyncReceiver struct {
	recommend *Recommend
}

// NewGoodsSyncReceiver 创建商品主数据同步接收器。
func NewGoodsSyncReceiver(recommend *Recommend) *GoodsSyncReceiver {
	return &GoodsSyncReceiver{recommend: recommend}
}

// Enabled 判断当前商品主数据同步接收器是否可用。
func (r *GoodsSyncReceiver) Enabled() bool {
	return r.recommend.Enabled()
}

// LoadIDs 加载推荐系统中已存在的商品主体编号集合。
func (r *GoodsSyncReceiver) LoadIDs(ctx context.Context, pageSize int) (_set.Set[string], error) {
	// 客户端未启用时，直接返回空商品集合。
	if !r.Enabled() {
		return _set.NewThreadUnsafeSet[string](), nil
	}
	// 分页大小非法时，回退到默认分页大小，避免Gorse 接口收到无效参数。
	if pageSize <= 0 {
		pageSize = 100
	}

	itemIDSet := _set.NewThreadUnsafeSetWithSize[string](pageSize)
	cursor := ""
	for {
		iterator, err := r.recommend.gorseClient.GetItems(ctx, pageSize, cursor)
		if err != nil {
			return nil, err
		}
		for _, item := range iterator.Items {
			// Gorse 返回空商品编号时，直接跳过当前无效数据。
			if item.ItemId == "" {
				continue
			}
			itemIDSet.Add(item.ItemId)
		}
		// 当前页没有更多游标或下一页游标未发生变化时，说明Gorse集合已经遍历完成。
		if iterator.Cursor == "" || iterator.Cursor == cursor {
			break
		}
		cursor = iterator.Cursor
	}
	return itemIDSet, nil
}

// SyncList 同步一批商品快照到推荐系统。
func (r *GoodsSyncReceiver) SyncList(ctx context.Context, goodsList []*models.GoodsInfo, existingItemIDs _set.Set[string], staleItemIDs _set.Set[string]) error {
	// 客户端未启用时，直接跳过当前商品同步批次。
	if !r.Enabled() {
		return nil
	}
	// 未传Gorse索引时，回退到单条 upsert 逻辑保证兼容性。
	if existingItemIDs == nil {
		for _, goods := range goodsList {
			// 无效商品快照不参与当前商品同步批次。
			if goods == nil || goods.ID <= 0 {
				continue
			}
			syncErr := r.sync(ctx, goods)
			if syncErr != nil {
				return syncErr
			}
		}
		return nil
	}

	insertItems := make([]client.Item, 0, len(goodsList))
	insertGoodsList := make([]*models.GoodsInfo, 0, len(goodsList))
	for _, goods := range goodsList {
		// 无效商品快照不参与当前商品同步批次。
		if goods == nil || goods.ID <= 0 {
			continue
		}
		recommendItem, itemPatch := r.buildPayload(goods)
		// 当前商品在本地仍然存在时，先从Gorse待删除集合中移除，避免后续误删有效主体。
		if staleItemIDs != nil {
			staleItemIDs.Remove(recommendItem.ItemId)
		}
		// Gorse已经存在时，直接走单条更新，避免重复插入失败后再回退。
		if existingItemIDs.ContainsOne(recommendItem.ItemId) {
			_, updateErr := r.recommend.gorseClient.UpdateItem(ctx, recommendItem.ItemId, itemPatch)
			if updateErr != nil {
				return updateErr
			}
			continue
		}
		insertItems = append(insertItems, recommendItem)
		insertGoodsList = append(insertGoodsList, goods)
	}
	// 当前批次没有新增商品时，说明本轮只命中了更新数据。
	if len(insertItems) == 0 {
		return nil
	}

	_, err := r.recommend.gorseClient.InsertItems(ctx, insertItems)
	// 批量插入失败时，回退到单条 upsert，避免因为索引陈旧或Gorse部分冲突导致整批失败。
	if err != nil {
		var fallbackErr error
		for _, goods := range insertGoodsList {
			syncErr := r.sync(ctx, goods)
			if syncErr != nil {
				fallbackErr = errors.Join(fallbackErr, syncErr)
			}
		}
		if fallbackErr != nil {
			return errors.Join(err, fallbackErr)
		}
		return nil
	}

	for _, item := range insertItems {
		existingItemIDs.Add(item.ItemId)
	}
	return nil
}

// DeleteIDs 删除推荐系统中多余的商品主体。
func (r *GoodsSyncReceiver) DeleteIDs(ctx context.Context, staleItemIDs _set.Set[string]) error {
	// 客户端未启用或没有待删除商品时，直接跳过当前清理批次。
	if !r.Enabled() || staleItemIDs == nil || staleItemIDs.IsEmpty() {
		return nil
	}
	var deleteErr error
	for itemID := range staleItemIDs.Iter() {
		// 待删除编号为空时，直接跳过当前无效主体。
		if itemID == "" {
			continue
		}
		// 推荐系统接口会在删除商品主体时一并级联删除该商品下的反馈数据。
		_, err := r.recommend.gorseClient.DeleteItem(ctx, itemID)
		if err != nil {
			deleteErr = errors.Join(deleteErr, err)
		}
	}
	return deleteErr
}

// sync 将单个商品快照同步到推荐系统。
func (r *GoodsSyncReceiver) sync(ctx context.Context, goods *models.GoodsInfo) error {
	// 客户端未启用或商品快照无效时，无需继续同步。
	if !r.Enabled() || goods == nil || goods.ID <= 0 {
		return nil
	}
	item, itemPatch := r.buildPayload(goods)
	_, err := r.recommend.gorseClient.InsertItem(ctx, item)
	if err == nil {
		return nil
	}

	_, updateErr := r.recommend.gorseClient.UpdateItem(ctx, item.ItemId, itemPatch)
	if updateErr == nil {
		return nil
	}
	return errors.Join(err, updateErr)
}

// buildPayload 构建推荐系统商品写入载荷。
func (r *GoodsSyncReceiver) buildPayload(goods *models.GoodsInfo) (client.Item, client.ItemPatch) {
	categoryIDs := make([]int64, 0)
	// 分类字段非空时，尝试解析为推荐系统分类维度。
	if strings.TrimSpace(goods.CategoryID) != "" {
		parseErr := json.Unmarshal([]byte(goods.CategoryID), &categoryIDs)
		// 分类 JSON 解析失败时，回退为空列表，避免单条商品脏数据阻塞整批推荐同步。
		if parseErr != nil {
			categoryIDs = []int64{}
		}
	}
	categories := make([]string, 0, len(categoryIDs))
	for _, categoryID := range categoryIDs {
		// 商品存在分类时，把分类编号作为推荐系统分类维度同步。
		if categoryID > 0 {
			categories = append(categories, strconv.FormatInt(categoryID, 10))
		}
	}

	timestamp := goods.UpdatedAt
	// 商品更新时间为空时，回退到创建时间，再不满足时使用当前时间。
	if timestamp.IsZero() {
		timestamp = goods.CreatedAt
	}
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	item := client.Item{
		ItemId:     strconv.FormatInt(goods.ID, 10),
		IsHidden:   goods.Status != _const.GOODS_STATUS_PUT_ON,
		Categories: categories,
		Timestamp:  timestamp,
		Comment:    goods.Name,
		Labels: map[string]interface{}{
			"desc":           goods.Desc,
			"status":         goods.Status,
			"price":          goods.Price,
			"discount_price": goods.DiscountPrice,
			"inventory":      goods.Inventory,
		},
	}
	return item, client.ItemPatch{
		IsHidden:   &item.IsHidden,
		Categories: item.Categories,
		Timestamp:  &item.Timestamp,
		Labels:     item.Labels,
		Comment:    &item.Comment,
	}
}
