# frontend/app

`frontend/app` 是商城端应用，基于 `uni-app + Vue 3 + TypeScript + Vite + Pinia + Sass` 开发，当前主要覆盖 H5 与微信小程序，同时保留 App 和多小程序平台脚本。

## 目录职责

```text
frontend/app
├── public                 # H5 静态资源
├── src
│   ├── api                # app / base 接口 service 封装
│   ├── components         # Xtx 前缀通用组件与 SKU 等业务组件
│   ├── composables        # 组合式逻辑
│   ├── pages              # 主包页面
│   ├── pagesMember        # 会员分包
│   ├── pagesOrder         # 订单分包
│   ├── rpc                # 后端 proto 生成的 TypeScript RPC 类型与客户端
│   ├── static             # 小程序与 H5 静态资源
│   ├── stores             # Pinia 状态
│   ├── styles             # 全局样式
│   ├── types              # 手写类型
│   ├── utils              # 请求、鉴权、路由、格式化等工具
│   ├── manifest.json      # uni-app 应用配置
│   └── pages.json         # 页面、分包、tabBar、easycom 配置
├── package.json
└── vite.config.ts
```

## 页面结构

主包 `src/pages`：

- 首页：`pages/index/index`
- 分类：`pages/category/category`
- 购物车：`pages/cart/cart`
- 我的：`pages/my/my`
- 登录与协议：`pages/login`
- 商品详情与评价：`pages/goods`
- 热门推荐：`pages/hot/hot`
- 搜索：`pages/search/index`
- WebView：`pages/webview/webview`

会员分包 `src/pagesMember`：

- 设置、个人信息、收货地址、收藏、门店认证。

订单分包 `src/pagesOrder`：

- 填写订单、订单详情、支付结果、订单列表、退款/售后、我的评价、写评价。

## 环境要求

- `Node.js >= 16.18.0`
- `pnpm`
- 后端服务默认运行在 `http://localhost:7001`
- 微信开发者工具（调试微信小程序时需要）

安装依赖：

```bash
cd frontend/app
pnpm install
```

## 环境变量

| 文件 | 说明 |
| --- | --- |
| `.env.development` | 通用开发配置，默认接口地址指向 `http://localhost:7001`。 |
| `.env.development-h5` | H5 开发配置，默认端口 `5002`，代理 `/api`、`/shop` 到后端。 |
| `.env.production` | 通用生产配置。 |
| `.env.production-h5` | H5 生产配置，默认发布到 `/app/`。 |

H5 开发代理：

```text
/api  -> http://localhost:7001
/shop -> http://localhost:7001
```

## 启动 H5

```bash
cd frontend/app
pnpm dev:h5
```

默认地址：`http://localhost:5002`。

## 启动微信小程序

```bash
cd frontend/app
pnpm dev:mp-weixin
```

然后使用微信开发者工具导入：

```text
frontend/app/dist/dev/mp-weixin
```

也可以从仓库根目录执行：

```bash
make -C frontend run-app
```

该命令默认启动微信小程序开发构建。

## 构建

构建 H5：

```bash
cd frontend/app
pnpm build:h5
```

`build:h5` 会通过 `UNI_OUTPUT_DIR=../../backend/data/app` 将 H5 产物输出到：

```text
backend/data/app
```

后端启动后，构建产物默认可通过 `http://localhost:7001/app` 访问。

构建微信小程序：

```bash
cd frontend/app
pnpm build:mp-weixin
```

输出目录通常为：

```text
frontend/app/dist/build/mp-weixin
```

## 常用脚本

```bash
pnpm dev:h5
pnpm dev:mp-weixin
pnpm build:h5
pnpm build:mp-weixin
pnpm tsc
pnpm lint
```

`package.json` 中还保留了 App、支付宝、百度、QQ、抖音、快手、飞书、quickapp 等平台脚本，实际使用前需要结合对应平台配置检查兼容性。

## 接口、状态与生成代码

- 业务接口统一通过 `src/api` 下的 service 发起，不要在页面里直接手写 `uni.request`。
- 通用请求、鉴权、刷新 token、错误提示逻辑集中在 `src/utils/http.ts`。
- Token 读写统一走 `src/utils/auth.ts`。
- 全局共享状态放在 `src/stores/modules`，并通过 `src/stores/index.ts` 汇总。
- 后端 proto 生成的 RPC 代码位于 `src/rpc`，由后端 `make ts` 生成，不手工维护等价类型。

## 多端兼容

- 页面默认优先保证微信小程序端可用，同时兼顾 H5。
- 涉及登录、路由、存储、分享、支付、预览图片等平台敏感逻辑时，需要检查 `MP-WEIXIN`、`H5`、`APP-PLUS` 的差异。
- 平台差异优先使用 uni-app 条件编译标记，例如 `#ifdef MP-WEIXIN`、`#ifdef H5`。
- 新增页面必须同步更新 `src/pages.json`，保持页面、分包、导航和 tabBar 配置一致。

## 设计文档

| 文档 | 说明 |
| --- | --- |
| [商城端设计](../../docs/商城端设计.md) | 商城端页面结构、请求登录态、订单 / 推荐 / 评价链路和多端兼容。 |
| [订单数据流转设计](../../docs/订单数据流转设计.md) | 确认单、下单、支付、退款、收货、评价和删除流程。 |
| [推荐数据流转设计](../../docs/推荐数据流转设计.md) | 匿名主体、推荐请求、曝光点击、收藏加购和交易事件回写。 |
| [评价与审核数据流转设计](../../docs/评价与审核数据流转设计.md) | 前台评价展示、写评价、讨论、互动和审核可见性。 |

## 校验

默认检查命令：

```bash
cd frontend/app
pnpm lint
pnpm tsc
```

若全量检查因历史问题失败，需要在提交或交付说明中记录失败文件、失败原因，以及是否由本次改动引起。
