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

- `views/base`：登录、AI 助手等 `base.v1` 公共能力页面。
- `views/system`：用户、角色、部门、菜单、字典、配置、日志、定时任务、API 管理、个人中心和代码生成等 `system.admin.v1` 管理能力页面。
- `views/shop`：按业务域组织 `dashboard/workspace`、`analytics`、`report`、`gorse`、`hot`、`recommend/request`、`comment`、`goods`、`order`、`pay`、`shop`、`user` 等 `shop.admin.v1` 商城运营页面；评论业务统一位于 `comment/info`，菜单路由页面不得置于 `components`。
- `views/migration/pending`：动态菜单组件无法匹配时的统一降级提示页。

动态菜单叶子页的 `defineOptions.name` 必须与 SQL 菜单 `name` 一致且全局唯一：表结构页面使用表名及用途后缀，非表结构页面使用业务组件路径的 PascalCase 名称；`Layout` 目录和私有 `components` 不强制声明路由页名称。

订单列表以门店子订单为操作单位：默认租户可查询全部数据并使用租户/门店树筛选，普通租户只查询自身数据并使用门店下拉筛选。工作台、订单分析、商品分析、订单月报和商品月报支持相同的租户/门店筛选；订单分析和订单报表在默认租户无筛选时按交易统计，选定租户/门店后切换为门店子订单口径。

租户管理员使用固定 `tenant` 角色。该角色菜单包含工作台、商品分析、订单分析、商品 / 订单报表、个人资料、组织权限、商品、评价和订单等租户经营能力；统计数据由后端按登录租户隔离。角色列表展示 `super` 和各租户的 `tenant` 内置角色；`super`、普通租户自己的 `tenant` 及其他租户的 `tenant` 不显示操作入口，默认租户自己的 `tenant` 权限模板允许编辑、删除、分配权限和启用/禁用。模板删除后，默认租户可重新创建 `code=tenant` 的角色并恢复原记录，菜单会再次同步到普通租户副本。默认租户维护普通租户的自定义角色时，菜单树以目标租户的 `tenant` 角色作为权限上限。用户账号创建后禁止通过用户管理修改；绑定 `super` 或 `tenant` 内置角色的管理员账号在用户列表中禁止勾选和状态切换，并隐藏重置密码、编辑、删除操作，只能登录后通过个人中心维护自身资料、手机号和密码。租户管理展示全部租户，默认租户在列表中禁止勾选、状态切换、编辑和删除。用户、角色与租户列表统一使用后端返回的末位字段 `is_protected = 300` 判断操作保护，不依赖前端自行推断或当前账号是否拥有角色选项接口权限；直接请求受保护租户、用户或角色详情也会被后端拒绝。请求层仅将 `401` 作为登录失效处理，`403` 会保留当前登录态并直接展示后端权限提示。租户管理、定时任务、API 管理、商城服务、推荐和支付账单仍属于平台公共页面，工作台租户视角会隐藏平台账单风险入口。

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

这些文件由后端目录的 `make ts` 生成。新增或调整接口时，应优先修改后端 `backend/api/proto/{base,common,shop,system}`，再执行生成命令，不要在 `src/rpc` 中手写等价类型；管理端业务类型分别位于 `src/rpc/system/admin/v1` 与 `src/rpc/shop/admin/v1`。

业务页面可以按现有模式在 `src/api` 中封装接口调用，也可以直接复用生成的 RPC 客户端，具体以当前页面风格为准。

管理端 `src/api` 按业务域组织：`src/api/base` 对应公共能力，`src/api/system` 与 `src/api/shop` 分别承载系统和商城管理端调用；终端标识保留在 `src/rpc/system/admin/v1` 与 `src/rpc/shop/admin/v1` 的生成协议中。

代码生成表配置页面使用独立接口封装：

- `src/api/system/code_gen.ts`：封装代码预览、启动生成任务和查询进度接口。
- `src/api/system/code_gen_table.ts`：数据库表选项及表配置 CRUD。
- `src/api/system/code_gen_column.ts`：数据库字段元数据及字段配置查询、保存。
- `src/api/system/code_gen_proto.ts`：Proto 接口检查与配置保存。
- `src/views/system/tool/code-gen/table/index.vue`：基于 `ProTable + FormDialog + ProForm` 的表配置入口；选择业务表后默认使用存在的 `parent_id`、`name` 作为树父字段和显示字段，选择左树数据表后默认使用存在的 `parent_id`、`name`、`id` 作为左树父字段、显示字段和值字段。选择业务表或左树数据表后会自动带出对应数据库表描述，两个描述均可继续修改；重新选择数据表时使用新表描述覆盖。
- `src/views/system/tool/code-gen/columns/index.vue`：结构化字段配置页面，字段配置接口不返回数据库主键和 `deleted_at`；页面完整展示其余数据库字段注释并允许单独修改字段描述，可在数据库字段列中通过拖拽手柄调整共用排序。查询、列表和表单分别维护自己的选项数据源，并按该排序生成和预览；相同组件会将完整选项一次性复刻到尚未配置的范围，复刻后允许独立修改，组件切回相同类型时重新复刻；选项形态由组件自动确定，表单组件范围与 `ProForm` 保持一致。列表组件固定为文本、开关、下拉、树形、图片、金额、日期；列表和表单开关默认使用 `status` 字典及 `1`、`2` 作为开启、关闭值，均可在选项中独立修改；下拉支持静态数据、字典和数据表来源，选择数据表且存在对应列时默认使用 `name`、`id` 作为 Label 字段、Value 字段；表单字典选择固定使用字典来源，并按字段类型生成对应的字典值类型；树形固定使用数据表来源，并在所选表存在对应列时默认使用 `parent_id`、`name`、`id` 作为树父字段、树显示字段、树值字段。表单树形选择默认单选，JSON 字段可在选项弹窗切换为多选；多选页面使用复选框并提交 ID 数组。
- `src/views/system/tool/code-gen/preview/index.vue`：从代码生成列表进入的独立完整页面预览，读取已经保存的表、字段、Proto 和字典配置，按普通表格、树形表格、左树右表三种类型渲染最终页面形态；三种表格统一使用自适应最小列宽，列少时自动铺满且不显示横向滚动条，列多时在表格内显示横向滚动条并固定操作列。左树右表预览复用用户管理的 `TreeFilter`，标题优先使用保存的左树描述，旧配置缺少描述时使用左树数据表描述补齐，并保持搜索、重置、展开、折叠和选中交互一致。开关使用配置的字典开启值、关闭值及标签，其他数据表选项在前端模拟。新增、编辑、删除按钮仅在对应 Proto 接口已存在或已勾选生成时展示，更新和删除接口都不可用时不生成操作列。模拟数据辅助逻辑位于同目录 `data.ts`，隐藏动态路由由 `sql/default-data.sql` 注册。
- `src/views/system/tool/code-gen/code-preview/index.vue`：从代码生成列表进入的独立代码预览页面，按当前项目固定路径加载生成文件，支持直接启动生成任务和查看最近进度；已有 Vue 页面会按模板节点、配置项、顶层声明和函数的稳定功能键增量合并，同名生成项按最新配置更新，已有扩展功能保持原顺序追加，无法安全解析时整批生成会在写文件前取消；隐藏动态路由由 `sql/default-data.sql` 注册。
- `src/views/system/tool/code-gen/components/CodePreviewPane.vue`：在独立代码预览页面中展示本次生成文件、增量动作和源码内容。
- `src/views/system/tool/code-gen/components/CodeGenProgressDialog.vue`：按任务和业务表展示文件、菜单、生成命令进度，通过 SSE 实时更新，并以三秒轮询兜底；最近任务 ID 保存在当前会话中用于页面恢复，后端重启或任务过期时会自动清理记录且不展示无效任务。
- `src/views/system/tool/code-gen/proto/index.vue`：Proto 接口检查与生成选择页面；表格合并展示接口信息、目标位置和检查状态，并将多行内容统一与列标题居中对齐。只有缺失接口勾选生成且接口类型为选项、树形或状态时，才在复选框后展示配置入口，并按接口类型固定显示所需字段；接口字段按目标实体对应的数据库表元数据加载和校验，不复用已过滤的字段配置接口结果。

当前管理后台 AI助手会复用：

- `src/api/base/ai_session.ts`：助手会话查询、创建、重命名和删除接口封装。
- `src/api/base/ai_tool.ts`：助手快捷入口工具接口封装。
- `src/api/base/ai_message.ts`：助手消息查询和流式发送接口封装。
- `src/views/base/ai`：基于 `vue-element-plus-x` 组件能力封装的当前系统专用 AI助手页面。
- `src/views/base/ai/index.vue`：隐藏的 AI助手页面入口，默认由顶部导航工具栏最左侧图标触发。

当前助手页实现说明：

- 当前 `vue-element-plus-x` 通过 `package.json` 与 pnpm lockfile 中锁定的 npm 依赖提供；业务代码统一从 `vue-element-plus-x` 导入。
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
- `ProForm` 字段使用 `colSpan` 控制栅格宽度；需要避免后续字段填充当前行剩余空间时，使用 `rowBreakBefore` 从新行开始布局。
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
| [AI 助手设计](../../docs/AI助手设计.md)                     | 会话、消息、工具调用、SSE 和管理端/商城端差异。             |
| [Gorse 推荐服务管理平台功能设计](docs/gorse-recommend-design.md) | 已有推荐运营页面与后续演进边界。                       |

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
