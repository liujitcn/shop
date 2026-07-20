package biz

import (
	"context"
	"errors"
	"strings"

	shopappv1 "shop/api/gen/go/shop/app/v1"
	"shop/pkg/biz"
	"shop/pkg/errorsx"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gorm"
)

const commentTagDefaultLimit = 20

// CommentTagCase 评价标签业务处理对象。
type CommentTagCase struct {
	*biz.BaseCase
	*data.CommentTagRepository
}

// NewCommentTagCase 创建评价标签业务处理对象。
func NewCommentTagCase(baseCase *biz.BaseCase, commentTagRepo *data.CommentTagRepository) *CommentTagCase {
	return &CommentTagCase{
		BaseCase:             baseCase,
		CommentTagRepository: commentTagRepo,
	}
}

// ListTags 查询商品评价标签列表。
func (c *CommentTagCase) ListTags(ctx context.Context, goodsID int64, limit int32) ([]*shopappv1.CommentTagItem, error) {
	recordList, err := c.listVisibleByGoodsID(ctx, goodsID)
	if err != nil {
		return nil, err
	}
	tagLimit := int(limit)
	// 商品评价标签最多返回前 20 个，两个前端页面按各自展示容量传入更小数量。
	if tagLimit <= 0 || tagLimit > commentTagDefaultLimit {
		tagLimit = commentTagDefaultLimit
	}
	// 当前商品标签超过展示数量时，只返回排序靠前的标签。
	if tagLimit < len(recordList) {
		recordList = recordList[:tagLimit]
	}

	tagList := make([]*shopappv1.CommentTagItem, 0, len(recordList))
	for _, record := range recordList {
		tagList = append(tagList, &shopappv1.CommentTagItem{
			TagId:        record.ID,
			Label:        record.Name,
			MentionCount: record.MentionCount,
		})
	}
	return tagList, nil
}

// MatchTagIDs 根据评价正文匹配商品下的标签编号。
func (c *CommentTagCase) MatchTagIDs(ctx context.Context, goodsID int64, content string) ([]int64, error) {
	// 评价正文为空时，不进行标签命中计算。
	if content == "" {
		return []int64{}, nil
	}

	recordList, err := c.listVisibleByGoodsID(ctx, goodsID)
	if err != nil {
		return nil, err
	}

	tagIDs := make([]int64, 0)
	for _, record := range recordList {
		// 标签名称为空或正文未命中时，不写入当前标签编号。
		if record.Name == "" || !strings.Contains(content, record.Name) {
			continue
		}
		tagIDs = append(tagIDs, record.ID)
	}
	return tagIDs, nil
}

// ExistingTagNames 查询商品已有评价标签名称。
func (c *CommentTagCase) ExistingTagNames(ctx context.Context, goodsID int64) ([]string, error) {
	recordList, err := c.listVisibleByGoodsID(ctx, goodsID)
	if err != nil {
		return nil, err
	}
	tagNames := make([]string, 0, len(recordList))
	for _, record := range recordList {
		// 标签名称为空时无法作为模型复用候选项。
		if record.Name == "" {
			continue
		}
		tagNames = append(tagNames, record.Name)
	}
	return tagNames, nil
}

// IncreaseMentionCount 批量增加标签提及次数。
func (c *CommentTagCase) IncreaseMentionCount(ctx context.Context, tagIDs []int64) error {
	// 没有命中的标签编号时，无需执行计数更新。
	if len(tagIDs) == 0 {
		return nil
	}

	query := c.Query(ctx).CommentTag
	// 标签提及次数需要数据库原子递增，避免并发评价覆盖计数。
	_, err := query.WithContext(ctx).
		Where(query.ID.In(tagIDs...)).
		UpdateSimple(query.MentionCount.Add(1))
	return err
}

// UpsertTagsByNames 根据标签名称创建或复用商品标签，并增加提及次数。
func (c *CommentTagCase) UpsertTagsByNames(ctx context.Context, tenantID, tenantStoreID, goodsID int64, tagNames []string) ([]int64, []string, error) {
	cleanNames := cleanCommentTagNames(tagNames)
	// 没有可用标签名称时，直接返回空结果。
	if len(cleanNames) == 0 {
		return []int64{}, []string{}, nil
	}

	tagIDs := make([]int64, 0, len(cleanNames))
	for _, tagName := range cleanNames {
		tagID, err := c.upsertTagByName(ctx, tenantID, tenantStoreID, goodsID, tagName)
		if err != nil {
			return nil, nil, err
		}
		tagIDs = append(tagIDs, tagID)
	}
	return tagIDs, cleanNames, nil
}

// DecreaseMentionCount 批量减少标签提及次数。
func (c *CommentTagCase) DecreaseMentionCount(ctx context.Context, tagIDs []int64) error {
	// 没有命中的标签编号时，无需执行计数更新。
	if len(tagIDs) == 0 {
		return nil
	}

	query := c.Query(ctx).CommentTag
	// 仅对大于 0 的记录执行原子递减，避免提及次数出现负数。
	_, err := query.WithContext(ctx).
		Where(
			query.ID.In(tagIDs...),
			query.MentionCount.Gt(0),
		).
		UpdateSimple(query.MentionCount.Sub(1))
	return err
}

// listVisibleByGoodsID 查询商品下的全部可展示标签。
func (c *CommentTagCase) listVisibleByGoodsID(ctx context.Context, goodsID int64) ([]*models.CommentTag, error) {
	query := c.Query(ctx).CommentTag
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.MentionCount.Desc()))
	return c.List(ctx, opts...)
}

// cleanCommentTagNames 清理 LLM 返回的标签名称。
func cleanCommentTagNames(tagNames []string) []string {
	cleanNames := make([]string, 0, len(tagNames))
	seen := make(map[string]struct{}, len(tagNames))
	for _, tagName := range tagNames {
		cleanName := tagName
		// 空标签不写入标签库。
		if cleanName == "" {
			continue
		}
		runes := []rune(cleanName)
		// 标签最长保留 8 个字符，避免模型输出过长短语污染筛选项。
		if len(runes) > 8 {
			cleanName = string(runes[:8])
		}
		// 重复标签只保留一次。
		if _, ok := seen[cleanName]; ok {
			continue
		}
		seen[cleanName] = struct{}{}
		cleanNames = append(cleanNames, cleanName)
		if len(cleanNames) >= 5 {
			break
		}
	}
	return cleanNames
}

// jsonStringTagNames 将标签名称转为 JSON 数组字符串。
func jsonStringTagNames(tagNames []string) string {
	return _string.ConvertAnyToJsonString(cleanCommentTagNames(tagNames))
}

// upsertTagByName 根据标签名称创建或复用商品标签。
func (c *CommentTagCase) upsertTagByName(ctx context.Context, tenantID, tenantStoreID, goodsID int64, tagName string) (int64, error) {
	query := c.Query(ctx).CommentTag
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.GoodsID.Eq(goodsID)))
	opts = append(opts, repository.Where(query.Name.Eq(tagName)))
	record, err := c.Find(ctx, opts...)
	if err == nil {
		_, err = query.WithContext(ctx).
			Where(query.ID.Eq(record.ID)).
			UpdateSimple(query.MentionCount.Add(1))
		if err != nil {
			return 0, err
		}
		return record.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	record = &models.CommentTag{
		TenantID:      tenantID,
		TenantStoreID: tenantStoreID,
		GoodsID:       goodsID,
		Name:          tagName,
		MentionCount:  1,
		Sort:          0,
	}
	err = c.Create(ctx, record)
	if err != nil {
		// 并发写入相同标签时，复用已存在记录并补加提及次数。
		if errorsx.IsMySQLDuplicateKey(err) {
			var retryRecord *models.CommentTag
			retryRecord, err = c.Find(ctx, opts...)
			if err != nil {
				return 0, err
			}
			_, err = query.WithContext(ctx).
				Where(query.ID.Eq(retryRecord.ID)).
				UpdateSimple(query.MentionCount.Add(1))
			if err != nil {
				return 0, err
			}
			return retryRecord.ID, nil
		}
		return 0, err
	}
	return record.ID, nil
}
