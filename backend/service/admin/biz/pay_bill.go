package biz

import (
	"context"

	"shop/api/gen/go/admin"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repo"
)

// PayBillCase 支付账单业务实例
type PayBillCase struct {
	*biz.BaseCase
	*data.PayBillRepo
	mapper *mapper.CopierMapper[admin.PayBill, models.PayBill]
}

// NewPayBillCase 创建支付账单业务实例
func NewPayBillCase(baseCase *biz.BaseCase, payBillRepo *data.PayBillRepo) *PayBillCase {
	return &PayBillCase{
		BaseCase:    baseCase,
		PayBillRepo: payBillRepo,
		mapper:      mapper.NewCopierMapper[admin.PayBill, models.PayBill](),
	}
}

// PagePayBill 分页查询支付账单
func (c *PayBillCase) PagePayBill(ctx context.Context, req *admin.PagePayBillRequest) (*admin.PagePayBillResponse, error) {
	query := c.Query(ctx).PayBill
	opts := make([]repo.QueryOption, 0, 1)
	if req.GetBillDate() != "" {
		opts = append(opts, repo.Where(query.BillDate.Eq(req.GetBillDate())))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*admin.PayBill, 0, len(list))
	for _, item := range list {
		payBill := c.mapper.ToDTO(item)
		resList = append(resList, payBill)
	}
	return &admin.PagePayBillResponse{List: resList, Total: int32(total)}, nil
}
