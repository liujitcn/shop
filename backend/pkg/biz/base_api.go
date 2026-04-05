package biz

import (
	"context"
	"fmt"
	"shop/pkg/gen/data"
	"shop/pkg/gen/models"
	"sort"
	"strings"

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

// apiCheck 检查并同步 openapi 接口数据
func (c *BaseApiCase) apiCheck(openApiData []byte) error {
	baseApiList, err := c.openApiDataToBaseApi(openApiData)
	if err != nil {
		return err
	}
	// API 检查改为同步执行，启动时直接根据 openapi 文档落库，避免排队导致接口权限数据滞后。
	return c.batchCreateBaseApi(context.TODO(), baseApiList)
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

// batchCreateBaseApi 批量同步接口数据
func (c *BaseApiCase) batchCreateBaseApi(ctx context.Context, apis []*models.BaseApi) error {
	oldApiList, err := c.List(ctx)
	if err != nil {
		return err
	}

	oldApiIdMap := make(map[string]int64, len(oldApiList))
	for _, oldApi := range oldApiList {
		oldApiIdMap[oldApi.Operation] = oldApi.ID
	}

	apiList := make([]*models.BaseApi, 0)
	for _, item := range apis {
		if id, ok := oldApiIdMap[item.Operation]; ok {
			item.ID = id
			err = c.UpdateById(ctx, item)
			if err != nil {
				return err
			}
			delete(oldApiIdMap, item.Operation)
			continue
		}
		apiList = append(apiList, item)
	}

	if len(oldApiIdMap) > 0 {
		oldApiIds := make([]int64, 0, len(oldApiIdMap))
		for _, id := range oldApiIdMap {
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
