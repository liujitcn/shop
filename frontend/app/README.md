# shop-app

`frontend/app` 是 `shop` 项目的商城端，基于 `uni-app + Vue 3 + TypeScript + Pinia`。当前仓库主要用于微信小程序与 H5，构建脚本同时保留 App 端能力。

## 当前实现范围

- 首页、分类、搜索、热门推荐
- 商品详情、SKU 选择
- 微信登录
- 购物车、收藏
- 收货地址
- 下单、支付结果、订单列表、订单详情
- 个人中心、设置、门店认证
- WebView 页面承载协议等内容
- 推荐链路已覆盖首页猜你喜欢、购物车、个人中心、商品详情、订单确认、订单详情和支付成功页。
- 前端推荐能力当前集中在 `src/api/app/recommend.ts`、`src/stores/modules/recommend.ts`、`src/utils/navigation.ts` 与 `src/components/XtxGuess.vue`，统一处理匿名主体、推荐上下文透传和曝光/点击埋点。

## 技术栈

- uni-app
- Vue 3
- TypeScript
- Pinia
- uni-ui
- Vite

## 环境要求

- Node.js `16.18+`
- `pnpm`
- 微信开发者工具（调试小程序时）
- HBuilderX（如需调试 App 端）

## 快速开始

### 安装依赖

```bash
pnpm install
```

### 启动 H5

```bash
pnpm dev:h5
```

默认地址：`http://localhost:5002`

### 启动微信小程序

```bash
pnpm dev:mp-weixin
```

然后使用微信开发者工具导入：

```text
dist/dev/mp-weixin
```

### 构建 H5

```bash
pnpm build:h5
```

H5 产物会输出到：

```text
../../backend/data/app
```

后端运行后会自动把该目录挂载到 `/app`。

### 构建微信小程序

```bash
pnpm build:mp-weixin
```

## 环境变量

开发与生产环境使用以下文件：

- `.env.development`
- `.env.development-h5`
- `.env.production`
- `.env.production-h5`

关键变量如下：

| 变量名 | 说明 | 开发默认值 |
| --- | --- | --- |
| `VITE_APP_PORT` | H5 开发端口 | `5002` |
| `VITE_APP_BASE_API` | 接口前缀 | `/api` |
| `VITE_APP_API_URL` | 后端地址 | `http://localhost:7001` |
| `VITE_APP_STATIC_API` | 静态资源前缀 | `/shop` |
| `VITE_APP_STATIC_URL` | 静态资源代理目标 | `http://localhost:7001` |
| `VITE_APP_BASE_PATH` | H5 根路径 | 开发 `/`，生产 `/app/` |

说明：

- `development-h5` 会合并 `.env.development` 与 `.env.development-h5`。
- `production-h5` 会合并 `.env.production` 与 `.env.production-h5`。
- H5 开发环境默认走同源代理，请求 `/api` 和 `/shop` 时由 Vite 转发到本地后端。

## 目录结构

```text
frontend/app
├── src
│   ├── api                 # 业务接口封装
│   ├── components          # 通用组件
│   ├── composables         # 组合式函数
│   ├── pages               # 主包页面
│   ├── pagesMember         # 会员相关分包
│   ├── pagesOrder          # 订单相关分包
│   ├── rpc                 # 生成的 TypeScript RPC 代码
│   ├── static              # 静态资源
│   ├── stores              # Pinia 状态管理
│   ├── styles              # 全局样式
│   ├── types               # 类型声明
│   ├── utils               # 请求与工具函数
│   ├── manifest.json       # uni-app 构建配置
│   └── pages.json          # 页面与路由配置
├── dist                    # 部分平台构建输出
├── unpackage               # uni-app 构建中间产物
├── package.json
└── vite.config.ts
```

## 页面结构

主包页面：

- `pages/index/index`
- `pages/category/category`
- `pages/cart/cart`
- `pages/my/my`
- `pages/login/login`
- `pages/hot/hot`
- `pages/goods/goods`
- `pages/search/index`
- `pages/webview/webview`

分包页面：

- `pagesMember/address/*`
- `pagesMember/collect/*`
- `pagesMember/profile/*`
- `pagesMember/settings/*`
- `pagesMember/store/*`
- `pagesOrder/create/*`
- `pagesOrder/detail/*`
- `pagesOrder/list/*`
- `pagesOrder/payment/*`

## 常用命令

```bash
pnpm dev:h5
pnpm dev:mp-weixin
pnpm build:h5
pnpm build:mp-weixin
pnpm tsc
pnpm lint
```

## 说明

- 仓库中仍保留 uni-app 默认的多平台脚本，但当前项目文档只覆盖实际在本仓库联调过的 H5 与微信小程序流程。
- `src/manifest.json` 中仍保留 App 与多个小程序平台配置；如果要正式发布对应平台，还需要按实际应用信息补全配置。
- 当前商品分类、搜索和详情链路已兼容多分类商品，商品总库存统一读取 `goods_info.inventory` 聚合字段。
