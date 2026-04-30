package biz

import (
	"context"

	adminv1 "shop/api/gen/go/admin/v1"
	"shop/pkg/biz"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
)

// PayBillCase 支付账单业务实例
type PayBillCase struct {
	*biz.BaseCase
	*data.PayBillRepository
	mapper *mapper.CopierMapper[adminv1.PayBill, models.PayBill]
}

// NewPayBillCase 创建支付账单业务实例
func NewPayBillCase(baseCase *biz.BaseCase, payBillRepo *data.PayBillRepository) *PayBillCase {
	return &PayBillCase{
		BaseCase:          baseCase,
		PayBillRepository: payBillRepo,
		mapper:            mapper.NewCopierMapper[adminv1.PayBill, models.PayBill](),
	}
}

// PagePayBills 查询支付账单列表
func (c *PayBillCase) PagePayBills(ctx context.Context, req *adminv1.PagePayBillsRequest) (*adminv1.PagePayBillsResponse, error) {
	query := c.Query(ctx).PayBill
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Order(query.BillDate.Asc()))
	// 传入账单日期时，仅查询对应日期的支付账单。
	if req.GetBillDate() != "" {
		opts = append(opts, repository.Where(query.BillDate.Eq(req.GetBillDate())))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}

	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*adminv1.PayBill, 0, len(list))
	for _, item := range list {
		payBill := c.mapper.ToDTO(item)
		resList = append(resList, payBill)
	}
	return &adminv1.PagePayBillsResponse{PayBills: resList, Total: int32(total)}, nil
}
