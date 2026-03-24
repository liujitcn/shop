package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	_const "shop/pkg/const"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"

	"github.com/go-kratos/kratos/v2/log"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"github.com/liujitcn/kratos-kit/sdk"
	"gopkg.in/yaml.v3"
)

// BaseApiCase 接口业务实例
type BaseApiCase struct {
	*data.BaseApiRepo
}

// NewBaseApiCase 创建接口业务实例
func NewBaseApiCase(baseApiRepo *data.BaseApiRepo) *BaseApiCase {
	return &BaseApiCase{BaseApiRepo: baseApiRepo}
}

// batchCreateBaseApi 批量同步接口数据
func (c *BaseApiCase) batchCreateBaseApi(ctx context.Context, apis []*models.BaseApi) error {
	oldApiList, err := c.List(ctx)
	if err != nil {
		return err
	}

	oldApiIDMap := make(map[string]int64, len(oldApiList))
	for _, oldApi := range oldApiList {
		oldApiIDMap[fmt.Sprintf("%s_%s", oldApi.Method, oldApi.Path)] = oldApi.ID
	}

	apiList := make([]*models.BaseApi, 0)
	for _, item := range apis {
		key := fmt.Sprintf("%s_%s", item.Method, item.Path)
		if id, ok := oldApiIDMap[key]; ok {
			item.ID = id
			err = c.UpdateById(ctx, item)
			if err != nil {
				return err
			}
			delete(oldApiIDMap, key)
			continue
		}
		apiList = append(apiList, item)
	}

	if len(oldApiIDMap) > 0 {
		oldApiIds := make([]int64, 0, len(oldApiIDMap))
		for _, id := range oldApiIDMap {
			oldApiIds = append(oldApiIds, id)
		}
		err = c.DeleteByIds(ctx, oldApiIds)
		if err != nil {
			return err
		}
	}

	if len(apiList) == 0 {
		return nil
	}
	return c.BatchCreate(ctx, apiList)
}

// saveApi 保存队列中的接口同步数据
func (c *BaseApiCase) saveApi(message queueData.Message) error {
	rawBody, err := json.Marshal(message.Values)
	if err != nil {
		log.Errorf("序列化接口同步消息失败[%s]", err.Error())
		return err
	}

	var payload map[string][]*models.BaseApi
	err = json.Unmarshal(rawBody, &payload)
	if err != nil {
		log.Errorf("反序列化接口同步消息失败[%s]", err.Error())
		return err
	}

	if list, ok := payload["data"]; ok {
		return c.batchCreateBaseApi(context.TODO(), list)
	}
	return nil
}

// apiCheck 检查并同步 openapi 接口数据
func (c *BaseApiCase) apiCheck(openApiData []byte) error {
	baseApiList, err := c.openApiDataToBaseApi(openApiData)
	if err != nil {
		return err
	}

	queue := sdk.Runtime.GetQueue()
	if queue == nil {
		return c.batchCreateBaseApi(context.TODO(), baseApiList)
	}

	messageData := map[string]interface{}{
		"data": baseApiList,
	}

	message, err := sdk.Runtime.GetStreamMessage(string(_const.ApiCheck), messageData)
	if err != nil {
		log.Errorf("构建接口同步消息失败[%s]", err.Error())
		return err
	}
	return queue.Append(string(_const.ApiCheck), message)
}

// openApiDataToBaseApi 将 openapi 文档转换为接口模型
func (c *BaseApiCase) openApiDataToBaseApi(openApiData []byte) ([]*models.BaseApi, error) {
	var api OpenAPI
	err := yaml.Unmarshal(openApiData, &api)
	if err != nil {
		return nil, err
	}

	tagsMap := make(map[string]string, len(api.Tags))
	for _, item := range api.Tags {
		if strings.HasPrefix(item.Description, "Admin") {
			tagsMap[fmt.Sprintf("admin.%s", item.Name)] = item.Description
			continue
		}
		if strings.HasPrefix(item.Description, "App") {
			tagsMap[fmt.Sprintf("app.%s", item.Name)] = item.Description
			continue
		}
		if strings.HasPrefix(item.Description, "Base") {
			tagsMap[fmt.Sprintf("base.%s", item.Name)] = item.Description
			continue
		}
		tagsMap[item.Name] = item.Description
	}

	baseApiList := make([]*models.BaseApi, 0)
	for path, item := range api.Paths {
		getApi := parseOperation(path, "GET", item.Get, tagsMap)
		if getApi != nil {
			baseApiList = append(baseApiList, getApi)
		}

		postApi := parseOperation(path, "POST", item.Post, tagsMap)
		if postApi != nil {
			baseApiList = append(baseApiList, postApi)
		}

		putApi := parseOperation(path, "PUT", item.Put, tagsMap)
		if putApi != nil {
			baseApiList = append(baseApiList, putApi)
		}

		deleteApi := parseOperation(path, "DELETE", item.Delete, tagsMap)
		if deleteApi != nil {
			baseApiList = append(baseApiList, deleteApi)
		}
	}

	sort.Slice(baseApiList, func(i, j int) bool {
		return baseApiList[i].Operation < baseApiList[j].Operation
	})
	return baseApiList, nil
}

// parseOperation 解析单个 openapi 操作项
func parseOperation(path, method string, op *Operation, tagsMap map[string]string) *models.BaseApi {
	if op == nil {
		return nil
	}

	var pkgName string
	paths := strings.Split(path, "/")
	if len(paths) > 2 {
		pkgName = paths[2]
	}
	if pkgName != "admin" && pkgName != "app" {
		pkgName = "base"
	}

	var serviceName string
	var serviceDesc string
	if len(op.Tags) > 0 {
		serviceName = fmt.Sprintf("%s.%s", pkgName, op.Tags[0])
		if value, ok := tagsMap[serviceName]; ok {
			serviceDesc = value
		}
	}

	return &models.BaseApi{
		ServiceName: serviceName,
		ServiceDesc: serviceDesc,
		Desc:        op.Description,
		Operation:   fmt.Sprintf("/%s.%s", pkgName, strings.ReplaceAll(op.OperationID, "_", "/")),
		Method:      method,
		Path:        path,
	}
}

// OpenAPI 描述 openapi 文档结构
type OpenAPI struct {
	Paths map[string]PathItem `yaml:"paths"`
	Tags  []TagsItem          `yaml:"tags"`
}

// PathItem 描述单个路径的请求方法
type PathItem struct {
	Get    *Operation `yaml:"get,omitempty"`
	Post   *Operation `yaml:"post,omitempty"`
	Put    *Operation `yaml:"put,omitempty"`
	Delete *Operation `yaml:"delete,omitempty"`
}

// TagsItem 描述 openapi 标签信息
type TagsItem struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// Operation 描述单个接口操作项
type Operation struct {
	Tags        []string `yaml:"tags"`
	Description string   `yaml:"description"`
	OperationID string   `yaml:"operationId"`
}
