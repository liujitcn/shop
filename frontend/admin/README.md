# frontend/admin

`frontend/admin` 是商城管理后台，基于 `Vue 3 + Vite + TypeScript + Element Plus + Pinia` 开发，负责系统管理、商品、订单、评价、报表、支付和推荐相关后台页面。

## 目录职责

```text
frontend/admin
├── build              # Vite 环境变量、插件、代理封装
├── docs               # 管理后台相关设计文档
├── public             # 静态资源
├── src
│   ├── api            # 手写接口封装
│   ├── components     # ProTable、ProForm、Dialog、Upload 等通用组件
│   ├── layouts        # 后台布局
│   ├── routers        # 路由与动态路由模块
│   ├── rpc            # 后端 proto 生成的 TypeScript RPC 类型与客户端
│   ├── stores         # Pinia 状态
│   ├── styles         # 全局样式、主题、暗黑模式
│   ├── typings        # 手写类型声明
│   ├── utils          # 请求、权限、格式化等工具
│   └── views          # 业务页面
├── types              # 自动导入等生成类型
├── package.json
└── vite.config.ts
```

## 页面模块

- `views/base`：用户、角色、部门、菜单、字典、配置、日志、定时任务、API 管理等系统管理。
- `views/dashboard`：工作台与分析页。
- `views/goods`：商品分类、商品信息、属性、SKU。
- `views/shop`：轮播图、商城服务、热门推荐。
- `views/order`：订单管理。
- `views/comment`：评价审核、评价详情、讨论审核。
- `views/report`：商品月报、订单月报。
- `views/pay`：交易账单。
- `views/recommend`：推荐请求、热门推荐、Gorse 推荐概览、任务、用户、商品、相似内容、反馈、高级调试、推荐编排、推荐配置。
- `views/user`：门店管理。

## 环境要求

- `Node.js >= 16.18.0`
- `pnpm`
- 后端服务默认运行在 `http://localhost:7001`

安装依赖：

```bash
cd frontend/admin
pnpm install
```

## 环境变量

| 文件 | 说明 |
| --- | --- |
| `.env` | 通用配置，默认端口 `8848`、应用标题、是否自动打开浏览器等。 |
| `.env.development` | 开发环境配置，公共路径 `/`，代理 `/api` 与 `/shop` 到后端。 |
| `.env.production` | 生产环境配置，公共路径 `/admin/`，接口前缀 `/api`。 |

开发代理：

```text
/api  -> http://localhost:7001
/shop -> http://localhost:7001
```

生产构建公共路径为 `/admin/`，构建产物会输出到 `../../backend/data/admin`。

## 开发启动

```bash
cd frontend/admin
pnpm dev
```

默认地址：`http://localhost:8848`。

也可以从仓库根目录执行：

```bash
make -C frontend run-admin
```

## 构建

```bash
cd frontend/admin
pnpm build
```

等价于 `pnpm build:pro`，会先执行 `vue-tsc`，再按 production 模式打包。

输出目录：

```text
backend/data/admin
```

后端启动后，构建产物默认可通过 `http://localhost:7001/admin` 访问。

## 常用脚本

```bash
pnpm dev
pnpm build
pnpm build:dev
pnpm build:test
pnpm type:check
pnpm lint:eslint
pnpm lint:prettier
pnpm lint:stylelint
pnpm preview
```

说明：

- `pnpm type:check`：执行 `vue-tsc --noEmit --skipLibCheck`。
- `pnpm lint:eslint`：对 `src` 下 `.js`、`.ts`、`.vue` 执行 ESLint 自动修复。
- `pnpm lint:prettier`：格式化 `src` 下常见前端文件。
- `pnpm lint:stylelint`：执行样式检查与自动修复。
- `pnpm preview`：先执行开发模式构建，再启动 Vite 预览。

## 接口与生成代码

后端 proto 生成的前端 RPC 代码位于：

```text
src/rpc
```

这些文件由后端目录的 `make ts` 生成。新增或调整接口时，应优先修改后端 `backend/api/protos`，再执行生成命令，不要在 `src/rpc` 中手写等价类型。

业务页面可以按现有模式在 `src/api` 中封装接口调用，也可以直接复用生成的 RPC 客户端，具体以当前页面风格为准。

## 开发约定

- 后台列表页优先延续 `ProTable + FormDialog + ProForm` 的页面结构。
- 图片列、状态列、顶部按钮、行内按钮优先使用 `ProTable` 现有配置能力。
- 业务页样式优先复用 `src/styles/common.scss`、`src/styles/element-dark.scss` 与主题变量。
- 新增页面后需要同步检查菜单初始化数据和后端接口权限初始化数据。
- 修改前端代码后，至少执行 `pnpm lint:eslint`；涉及类型或构建链路时补充 `pnpm type:check` 或构建命令。

## 设计文档

| 文档 | 说明 |
| --- | --- |
| [管理后台设计](../../docs/管理后台设计.md) | 后台页面组织、接口调用、权限菜单、通用页面模式和主题样式。 |
| [推荐系统设计](../../docs/推荐系统设计.md) | 推荐场景、Gorse 集成、本地兜底和后台管理能力。 |
| [推荐数据流转设计](../../docs/推荐数据流转设计.md) | 推荐请求、推荐事件、同步任务和后台排查链路。 |
| [统计数据流转设计](../../docs/统计数据流转设计.md) | 工作台、订单 / 商品 / 用户分析和报表数据来源。 |
| [订单数据流转设计](../../docs/订单数据流转设计.md) | 后台订单查询、退款、发货与订单状态边界。 |
| [评价与审核数据流转设计](../../docs/评价与审核数据流转设计.md) | 评价、讨论、AI 摘要审核和前台可见性规则。 |

## 校验

默认检查命令：

```bash
cd frontend/admin
pnpm lint:eslint
pnpm type:check
```

若全量检查因历史问题失败，需要在提交或交付说明中记录失败文件、失败原因，以及是否由本次改动引起。
