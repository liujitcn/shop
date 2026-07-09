# frontend/admin

`frontend/admin` 是商城管理后台，基于 `Vue 3 + Vite + TypeScript + Element Plus + Pinia` 开发，负责平台系统管理、租户经营后台、商品、订单、评价、报表、支付和推荐相关后台页面。

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

`types/generated` 当前包含 `auto-imports.d.ts` 和 `components.d.ts`，分别由 `unplugin-auto-import` 与
`unplugin-vue-components` 生成，用于 Element Plus API、图标和组件按需导入的类型提示。

## 页面模块

- `views/base`：用户、角色、部门、菜单、字典、配置、日志、定时任务、API 管理等系统管理。
- `views/dashboard`：工作台与分析页。
- `views/ai`：AI助手等 AI 能力页面。
- `views/goods`：商品分类、商品信息、属性、SKU。
- `views/shop`：租户门店管理、轮播图、商城服务、热门推荐。
- `views/order`：订单管理。
- `views/comment`：评价审核、评价详情、讨论审核。
- `views/report`：商品月报、订单月报。
- `views/pay`：交易账单。
- `views/recommend`：推荐请求、热门推荐、Gorse 推荐概览、任务、用户、商品、相似内容、反馈、高级调试、推荐编排、推荐配置。
- `views/user`：用户相关历史页面。
- `views/migration/pending`：动态菜单组件无法匹配时的统一降级提示页。

租户管理员使用固定 `tenant` 角色。该角色菜单只包含工作台、订单分析、个人资料、组织权限、商品、评价和订单等租户经营能力，不包含租户管理、定时任务、API 管理、商城服务、推荐、支付账单和报表分析等平台公共页面。工作台租户视角会隐藏平台账单风险入口。

## 环境要求

- `Node.js ^20.19.0 || >=22.12.0`
- `pnpm`
- 后端服务默认运行在 `http://localhost:7001`

安装依赖：

```bash
cd frontend/admin
pnpm install
```

## 环境变量

| 文件               | 说明                                                                   |
| ------------------ | ---------------------------------------------------------------------- |
| `.env`             | 通用配置，默认端口 `8848`、应用标题、是否自动打开浏览器等。            |
| `.env.development` | 开发环境配置，公共路径 `/`，代理 `/api`、`/events` 与 `/shop` 到后端。 |
| `.env.production`  | 生产环境配置，公共路径 `/admin/`，接口前缀 `/api`。                    |

常用变量：

- `VITE_API_URL`：HTTP API 前缀，开发环境默认为 `/api`。

开发代理：

```text
/api  -> http://localhost:7001
/events/* -> http://localhost:7001
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
pnpm lint:oxlint
pnpm lint:prettier
pnpm lint:stylelint
pnpm preview
```

说明：

- `pnpm type:check`：执行 `vue-tsc --noEmit --skipLibCheck`。
- `pnpm lint:oxlint`：对 `src` 下 `.js`、`.ts`、`.vue` 执行 Oxlint 自动修复。
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

当前管理后台 AI助手会复用：

- `src/api/base/ai_assistant_session.ts`：助手会话查询、创建、重命名和删除接口封装。
- `src/api/base/ai_assistant_message.ts`：助手消息查询和流式发送接口封装。
- `src/views/ai/assistant`：基于 `vue-element-plus-x` 组件能力封装的当前系统专用 AI助手页面。
- `src/views/ai/assistant/index.vue`：隐藏的 AI助手页面入口，默认由顶部导航工具栏最左侧图标触发。

当前助手页实现说明：

- 当前 `vue-element-plus-x` 通过 `file:./vendor/vue-element-plus-x` 引用本地最新源码包；业务代码仍统一从 `vue-element-plus-x` 导入，后续切回官方 npm tag 时不需要调整页面 import。
- 会话列表、消息流、输入器、附件区和 Markdown 回复分别复用 `Conversations`、`BubbleList`、`Sender`、`Attachments`、`XMarkdown`。
- 附件发送会透传 `id/name/size/url/mime_type`，用于后端读取真实文件内容参与模型推理。
- 输入器支持点击回形针选择附件、直接粘贴文件和浏览器语音输入；当前限制最多 6 个附件，单个附件不超过 20MB。
- 附件卡片支持预览：图片使用大图预览，PDF、Word、Excel 等其它文件通过浏览器新窗口打开。
- 图片附件会由服务端读取为多模态视觉输入发送给模型；文本、Markdown、日志、JSON、XML、CSV 等文本类附件会读取正文后随本轮问题发送给模型。PDF、Word、Excel 等文档当前先上传、展示并保存元信息，暂不解析正文。
- 助手回复使用 `vue-element-plus-x` 的 `XMarkdown` 渲染 Markdown，支持标题、列表、表格、引用和代码块；用户消息仍按纯文本展示，避免把原始输入误解析为 Markdown。
- 聊天区会先本地回显用户消息，再显示助手“思考中”占位态；后端一轮消息会在前端拆成用户气泡与助手气泡，助手回复通过 SSE 流式逐段渲染，最终再与服务端正式消息收敛，并按 `reply_source` 区分模型回答、工具回答和降级回复。最终助手气泡会展示服务端返回的真实 token 用量、首 Token 耗时和总耗时；function 工具命中后会在消息内容区展示轻量工具过程行，原始请求与响应报文默认收起，仅在展开过程行后查看。
- 助手消息流按会话独立消费；某个会话正在流式回复时，切换到其它会话仍可继续发送新问题，后台会话的增量回复会继续写回对应消息列表。
- 聊天气泡提供消息级操作：失败的一轮消息可从用户气泡重新发送，成功用户气泡支持复制和删除；助手气泡支持重新生成、从当前消息创建持久化分支会话、浏览器朗读、复制和删除。删除消息、重新发送、重新生成和创建分支都走后端接口持久化；复制和朗读只在浏览器侧完成。

## 开发约定

- 后台列表页优先延续 `ProTable + FormDialog + ProForm` 的页面结构。
- 图片列、状态列、顶部按钮、行内按钮优先使用 `ProTable` 现有配置能力。
- 业务页样式优先复用 `src/styles/common.scss`、`src/styles/element-dark.scss` 与主题变量。
- 新增页面后需要同步检查菜单初始化数据和后端接口权限初始化数据。
- 新增租户可见页面时，需要同步更新 `sql/default-data.sql` 中默认租户 `tenant` 角色菜单模板，并刷新对应租户的接口权限策略。
- 修改前端代码后，至少执行 `pnpm lint:oxlint`；涉及类型或构建链路时补充 `pnpm type:check` 或构建命令。

## 设计文档

| 文档                                                           | 说明                                                       |
| -------------------------------------------------------------- | ---------------------------------------------------------- |
| [管理后台设计](../../docs/管理后台设计.md)                     | 后台页面组织、接口调用、权限菜单、通用页面模式和主题样式。 |
| [推荐系统设计](../../docs/推荐系统设计.md)                     | 推荐场景、Gorse 集成、本地兜底和后台管理能力。             |
| [推荐数据流转设计](../../docs/推荐数据流转设计.md)             | 推荐请求、推荐事件、同步任务和后台排查链路。               |
| [统计数据流转设计](../../docs/统计数据流转设计.md)             | 工作台、订单 / 商品 / 用户分析和报表数据来源。             |
| [订单数据流转设计](../../docs/订单数据流转设计.md)             | 后台订单查询、退款、发货与订单状态边界。                   |
| [评价与审核数据流转设计](../../docs/评价与审核数据流转设计.md) | 评价、讨论、评价摘要审核和前台可见性规则。                 |

## 校验

默认检查命令：

```bash
cd frontend/admin
pnpm lint:oxlint
pnpm type:check
```

## GoLand / IDEA 保存格式化

GoLand / IDEA 中建议开启：

- `Settings | Languages & Frameworks | JavaScript | Prettier`：选择 `frontend/admin/node_modules/prettier`，勾选 `On save`，文件范围使用 `{**/*,*}.{js,ts,vue,json,scss,css,html,md}`。
- `Settings | Tools | Actions on Save`：勾选 `Reformat code`、`Optimize imports`，并确认 `Run Prettier` 已启用。
- 如已安装 Oxlint 插件，可在插件设置中开启保存时检查；否则使用 `pnpm lint:oxlint` 做提交前自动修复。

若全量检查因历史问题失败，需要在提交或交付说明中记录失败文件、失败原因，以及是否由本次改动引起。
