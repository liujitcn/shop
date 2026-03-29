# View 页面组件化改造记录

## 1. 背景

本次改造目标是将 `src/views` 下原先大量使用 `el-form + el-table + pagination` 手写结构的页面，统一替换为当前项目已经接入的组件化方案：

- 列表页统一使用 `ProTable`
- 树筛选列表页统一使用 `TreeFilter + ProTable`
- 树形数据页统一使用树表格风格的 `ProTable`

本次改造时，主要参考了原版 Geeker Admin 中的以下页面：

- 原版参考目录：`/Users/liujun/Downloads/Geeker-Admin-master`
- 树筛选列表参考：`src/views/proTable/useTreeFilter/index.vue`
- 树形表格参考：`src/views/proTable/treeProTable/index.vue`

## 2. 统一改造规则

### 2.1 列表页

普通列表页统一改为：

- 通过 `columns` 配置搜索项、字段、枚举项
- 通过 `requestApi` 统一接管分页、查询、刷新
- 使用插槽保留状态开关、图片、金额、操作按钮等业务定制内容

### 2.2 树筛选页

存在左侧分类/部门筛选的页面统一改为：

- 左侧 `TreeFilter`
- 右侧 `ProTable`
- `TreeFilter` 通过 `initParam` 或搜索参数驱动 `ProTable` 数据刷新

这类页面直接对齐原版 `useTreeFilter` 的实现思路。

### 2.3 树形数据页

分类、部门、菜单等树结构页面统一改为：

- `ProTable` 直接承载树数据
- 配置 `row-key`
- 关闭分页
- 使用 `default-expand-all`、`tree-props`

这类页面直接对齐原版 `treeProTable` 的实现思路。

## 3. 新增的通用能力

### 3.1 通用表格工具

新增文件：`src/utils/proTable.ts`

包含以下公共方法：

- `buildDictEnum`
  - 将字典缓存转换为 `ProTable` 可用的枚举数据
- `buildPageRequest`
  - 统一规范 `pageNum/pageSize`
- `normalizeSelectedIds`
  - 统一处理单条删除、批量删除产生的 ID 集合

### 3.2 通用展示工具

新增文件：`src/utils/utils.ts`

包含以下公共方法：

- `formatJson`
- `formatPrice`
- `formatSrc`

## 4. 已完成的页面改造

### 4.1 树筛选 + 表格

参考原版 `useTreeFilter/index.vue` 改造：

- `src/views/base/user/index.vue`
  - 左侧部门树，右侧用户表格
- `src/views/goods/info/index.vue`
  - 左侧分类树，右侧商品表格

### 4.2 树形表格

参考原版 `treeProTable/index.vue` 改造：

- `src/views/base/dept/index.vue`
- `src/views/goods/category/index.vue`

### 4.3 普通 ProTable 列表页

以下页面已改为 `ProTable` 风格：

- `src/views/pay/bill/index.vue`
- `src/views/shop/service/index.vue`
- `src/views/goods/prop/index.vue`
- `src/views/shop/hot/index.vue`
- `src/views/base/dict/index.vue`
- `src/views/base/config/index.vue`
- `src/views/shop/banner/index.vue`
- `src/views/user/store/index.vue`
- `src/views/goods/sku/index.vue`
- `src/views/base/log/index.vue`
- `src/views/base/role/index.vue`
- `src/views/base/job/index.vue`
- `src/views/base/dict/item.vue`
- `src/views/base/job/log.vue`
- `src/views/order/info/index.vue`

### 4.4 已保留现有组件化实现

- `src/views/base/menu/index.vue`

该页原本已经是 `ProTable` 风格，本轮未重复重构。

## 5. 本轮顺手修复的关联问题

为了让本次改造链路可正常工作，本轮还补了部分直接关联文件的缺失导入问题：

- `src/views/base/user/components/dept-tree.vue`
- `src/views/goods/info/components/category-tree.vue`
- `src/views/order/info/detail/index.vue`
- `src/views/goods/info/detail/index.vue`

这些调整属于最小修复，目的是保证当前列表页跳转或树筛选依赖的页面不再因为基础导入缺失而报错。

## 6. 这次实际保留的业务能力

虽然页面结构改成了组件化表格，但以下业务没有被移除：

- 新增、编辑、删除
- 启用/禁用状态切换
- 权限控制按钮显示
- 角色权限分配
- 定时任务启动、停止、执行一次、查看日志
- 字典项编辑与状态切换
- 订单发货、发货详情、退款、退款详情
- 商品属性、规格、详情跳转

原则是：

- 只改列表容器和查询结构
- 尽量不改原业务接口和弹窗行为
- 先统一模式，再逐页细化

## 7. 与原版 Geeker Admin 的对应关系

### 7.1 借鉴点

从原版借鉴的核心不是业务代码，而是页面组织方式：

- `TreeFilter` 负责树筛选
- `ProTable` 负责查询、分页、列配置、插槽定制
- 列表页尽量通过 `columns` 驱动
- 树页尽量通过树表格表达，而不是手工递归 `el-table`

### 7.2 本项目的落地差异

由于当前项目是商城后台，不是原版示例项目，所以落地时做了以下适配：

- 保留了商城业务接口和字段结构
- 接入当前项目已有的权限体系
- 接入当前项目已有的字典组件 `Dict/DictLabel`
- 对金额、图片、JSON 内容补了项目内公共格式化工具

## 8. 当前状态判断

### 8.1 已完成

当前主干的基础管理、商品管理、订单列表相关页面，已经基本形成统一的组件化结构。

### 8.2 仍待继续整理的页面

当前仓库里还有部分旧页面仍未统一到本次模式，例如：

- `src/views/goods/info/edit/index.vue`
- `src/views/order/info/shipped/index.vue`
- `src/views/shop/hot/item.vue`
- `src/views/profile/index.vue`

这些页面更多属于编辑页、详情页或表单页，不完全是列表页组件化问题，可在下一阶段继续清理。

## 9. 校验说明

本次改造后，已对当前改动链路做过定向类型检查，关注文件包括：

- `base/user`
- `base/role`
- `base/job`
- `base/dict/item`
- `base/job/log`
- `goods/info`
- `goods/info/detail`
- `order/info`
- `order/info/detail`

结论：

- 当前这批已改文件未再出现新的定向类型错误
- 全仓库 `pnpm type:check` 仍会被仓库内其他未处理旧页面阻塞

## 10. 建议的后续推进顺序

如果继续按同一套规范推进，建议顺序如下：

1. 优先清理订单详情链路和商品编辑链路
2. 再处理 `shop/hot/item` 这类旧表单页
3. 最后统一处理 `profile` 等非后台资源管理类页面

这样可以优先保证后台核心业务页的风格统一。
