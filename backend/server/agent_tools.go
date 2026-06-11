package server

import (
	appv1 "shop/api/gen/go/app/v1"
	einoTool "shop/pkg/agent/eino/tool"
)

// newAdminFlowAgentTools 创建管理端 AI 助手流程工具列表。
func newAdminFlowAgentTools(_ *ServerServices) ([]einoTool.Invokable, error) {
	return nil, nil
}

// newAppFlowAgentTools 创建商城端 AI 助手流程工具列表。
func newAppFlowAgentTools(services *ServerServices) ([]einoTool.Invokable, error) {
	var err error
	var builder agentToolBuilder
	if err = builder.appendTool(appv1.NewRecommendServiceRecommendGoodsAgentTool(services.appRecommend)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(appv1.NewGoodsInfoServicePageGoodsInfoAgentTool(services.appGoods)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(appv1.NewGoodsInfoServiceGetGoodsInfoAgentTool(services.appGoods)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(appv1.NewOrderInfoServiceBuyNowOrderInfoAgentTool(services.appOrder)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(appv1.NewOrderInfoServiceCreateOrderInfoAgentTool(services.appOrder)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(appv1.NewOrderInfoServicePageOrderInfoAgentTool(services.appOrder)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(appv1.NewOrderInfoServiceGetOrderInfoByIdAgentTool(services.appOrder)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(appv1.NewOrderInfoServiceReceiveOrderInfoAgentTool(services.appOrder)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(appv1.NewUserAddressServiceListUserAddressesAgentTool(services.appUserAddress)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(appv1.NewUserAddressServiceCreateUserAddressAgentTool(services.appUserAddress)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(appv1.NewCommentServicePagePendingCommentGoodsAgentTool(services.appComment)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(appv1.NewCommentServiceCreateCommentAgentTool(services.appComment)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(appv1.NewPayServiceJsapiPayAgentTool(services.appPay)); err != nil {
		return nil, err
	}
	if err = builder.appendTool(appv1.NewPayServiceH5PayAgentTool(services.appPay)); err != nil {
		return nil, err
	}
	return builder.tools, nil
}

// agentToolBuilder 负责合并各服务生成的 Agent 工具。
type agentToolBuilder struct {
	tools []einoTool.Invokable
}

// appendTool 合并单个生成的 Agent 工具。
func (b *agentToolBuilder) appendTool(value einoTool.Invokable, err error) error {
	if err != nil {
		return err
	}
	b.tools = append(b.tools, value)
	return nil
}
