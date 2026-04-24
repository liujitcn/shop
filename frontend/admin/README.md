# shop-admin

`shop-admin` 是 `shop` 项目的管理后台，基于 `Vue 3 + Vite + TypeScript + Element Plus + Pinia`。

项目采用后端动态菜单驱动路由，认证链路使用 `access token + refresh token`，管理商品、订单、门店、运营内容和系统基础数据。

## 技术栈

- Vue 3
- Vite 5
- TypeScript 5
- Element Plus
- Pinia
- Vue Router
- Axios
- ECharts
- WangEditor

## 环境要求

- Node.js `>= 16.18.0`
- `pnpm`

## 快速开始

### 安装依赖

```bash
pnpm install
```

### 启动开发环境

```bash
pnpm dev
```

默认行为：

- 本地端口：`8848`
- 自动打开浏览器：`true`
- 路由模式：`hash`
- 开发接口前缀：`/api`

开发环境会把以下请求代理到本地后端 `http://localhost:7001`：

- `/api`
- `/shop`

## 常用命令

```bash
pnpm dev
pnpm build
pnpm build:dev
pnpm build:test
pnpm build:pro
pnpm preview
pnpm type:check
pnpm lint:eslint
pnpm lint:prettier
pnpm lint:stylelint
```

## 构建产物

`vite.config.ts` 当前将构建输出目录固定为：

```text
../../backend/data/shop
```

这意味着：

- 构建结果不会输出到当前目录下的 `dist`
- 产物会直接写入后端静态目录
- 后端运行后可直接通过 `/shop` 访问管理后台

生产环境公共路径当前为：

```text
/shop/
```

## 环境变量

项目使用 `.env`、`.env.development`、`.env.production` 管理配置。

关键变量如下：

| 变量名 | 说明 | 当前值 |
| --- | --- | --- |
| `VITE_GLOB_APP_TITLE` | 应用标题 | `Shop Admin` |
| `VITE_PORT` | 开发端口 | `8848` |
| `VITE_OPEN` | 自动打开浏览器 | `true` |
| `VITE_PUBLIC_PATH` | 生产公共路径 | `/shop/` |
| `VITE_API_URL` | 接口前缀 | `/api` |
| `VITE_PROXY` | 开发代理 | `[["/api","http://localhost:7001"],["/shop","http://localhost:7001"]]` |

## 目录结构

```text
shop-admin
├── build                  # Vite 构建扩展配置
├── docs                   # 项目文档
├── public                 # 静态资源
├── src
│   ├── api                # 业务接口封装
│   ├── assets             # 图片、字体、svg 等资源
│   ├── components         # 通用组件
│   ├── config             # 全局配置
│   ├── directives         # 自定义指令
│   ├── enums              # 枚举定义
│   ├── hooks              # 组合式 hooks
│   ├── layouts            # 布局组件
│   ├── routers            # 静态路由与动态路由初始化
│   ├── rpc                # 生成的 TypeScript RPC 代码
│   ├── stores             # Pinia 状态管理
│   ├── styles             # 全局样式与主题变量
│   ├── utils              # 工具函数与请求封装
│   └── views              # 页面模块
├── .env*                  # 环境变量
├── package.json
└── vite.config.ts
```

## 项目文档

- `docs/gorse-remote-recommend-design.md`：远程推荐 Gorse 管理平台功能设计文档，梳理后台可承接的运行总览、场景策略、同步、事件、调试、发布与告警能力。

## 通用展示组件

- `src/components/Card/DataPanelCard.vue`：通用数据展示卡片，用于图表、表格摘要等带标题的内容块。
- 数据分析页、订单月报页、商品月报页统一复用该组件，避免将通用卡片继续放在具体业务目录下维护。

## 已落地页面模块

- 登录与个人中心
- 工作台、数据分析
- 订单月报、订单日报、商品月报、商品日报
- 用户、角色、菜单、部门、岗位、日志、字典、系统配置
- 门店管理
- 商品分类、商品信息、规格、属性、SKU
- 订单管理、发货处理、订单详情
- 轮播图、商城服务
- 推荐管理（热门推荐、推荐请求、远程推荐）
- 支付管理（支付账单）

商品管理当前补充了统一的页面交互约定：

- 商品信息编辑页采用“商品信息 -> 商品属性 -> 规格项 -> 商品规格”四步流程
- 商品信息编辑页支持多分类勾选，列表与详情页统一展示组合分类名称
- 商品详情页与编辑页的摘要区、属性/规格项/商品规格分区保持统一展示风格
- 商品列表、编辑页、详情页统一展示按 SKU 汇总后的商品总库存
- 商品详情页支持商城 H5 商品详情预览入口，便于核对轮播图、详情图与规格展示效果

其中推荐相关后台入口当前已覆盖：

- 推荐请求分页列表与详情页查看
- 推荐请求详情页改为页面跳转，并通过 tab 展示请求信息、链路与上下文、推荐商品
- 推荐请求链路轨迹、最终推荐器与上下文查看
- 推荐请求中的推荐器已接入 `recommend_provider` 字典，默认展示英文标识，悬浮显示中文名称
- 推荐商品关联的 recommend_event 数据通过独立弹窗查看，默认展示当前商品事件明细
- 远程推荐概览、任务状态查看
- 远程推荐概览、任务、用户、商品、配置与编排页面已参照 Gorse 管理端改为后台可视化展示，避免直接把接口 JSON 作为主内容输出
- 远程推荐用户、商品列表与详情查看
- 远程推荐用户、商品删除
- 远程推荐数据导入导出
- 推荐编排配置查看、保存与重置
- 远程推荐配置查看
- 推荐管理页面目录统一收敛在 `src/views/recommend`
- 推荐请求页面目录收敛在 `src/views/recommend/request`，详情页位于 `src/views/recommend/request/detail`
- 推荐请求详情页路由由菜单数据动态下发，旧库可通过 `sql/patch-recommend-request-detail-menu.sql` 补齐
- 远程推荐页面目录统一收敛在 `src/views/recommend/remote`

## 上传类型约定

- 商品分类图片上传使用 `category`
- 商品管理相关图片上传使用 `goods`
- 轮播图图片上传使用 `banner`
- 热门推荐图片上传使用 `hot`
- 未纳入以上业务分类的通用图片上传，默认仍使用 `image`

首页默认路由：

```text
/dashboard/workspace
```

## 权限与路由

- 静态路由只保留登录页、布局页和错误页。
- 业务路由主要由后端接口动态返回。
- 前端启动后会拉取菜单和按钮权限并注册路由。
- 页面组件路径基于 `src/views` 自动匹配。

相关接口：

- `GET /admin/auth/userInfo`
- `GET /admin/auth/menu`
- `GET /admin/auth/button`

## 认证机制

登录相关接口分为验证码与认证资源两组：

- `GET /login/captcha`
- `POST /auth`
- `POST /auth/token`
- `DELETE /auth`

当前处理策略：

- 登录成功后缓存 `access token`、`refresh token`、`tokenType` 和过期时间。
- 请求前会在令牌快过期时自动刷新。
- 遇到 `401` 或 `403` 会清理本地认证信息并要求重新登录。
