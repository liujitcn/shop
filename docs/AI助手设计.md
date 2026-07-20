# AI 助手设计

## 文档目标

本文档说明 AI 助手在管理后台和商城端的产品定位、页面结构、快捷操作、会话消息、结构化流程动作和端侧维护约定。

## 能力定位

AI 助手是跨终端的对话式业务入口，目标是把常见运营、客服和用户自助流程收敛到同一套会话内：

- 管理后台面向运营、客服和管理员，强调订单、商品、评价、推荐、统计等后台工作流的快速触发和可追踪执行。
- 商城端面向用户，强调购物咨询、订单协助、地址填写、评价引导和售后自助。
- 后端保持统一会话、消息、快捷入口和流程动作协议，前端按终端做不同的信息密度和交互呈现。

## 总体链路

~~~mermaid
flowchart LR
  Shortcut["快捷入口"] --> Composer["输入框 / 发送载荷"]
  UserMessage["用户消息"] --> MessageAPI["AI 助手消息接口"]
  Composer --> MessageAPI
  MessageAPI --> Stream["流式回复"]
  Stream --> AiMessage["助手消息"]
  AiMessage --> Blocks["结构化流程块"]
  Blocks --> Action["流程动作"]
  Action --> MessageAPI
~~~

- 快捷入口来自 ListAiShortcut，按终端过滤后返回。
- 前端点击快捷入口时，发送 text + action，不在端侧伪造业务结果。
- 助手消息可以包含 Markdown、工具调用摘要、附件和结构化流程块。
- 结构化流程动作继续通过消息接口提交，保证流程上下文在会话内闭环。

## 后端契约与运行时

AI 公共接口位于 `backend/api/proto/base/v1/ai_session.proto` 与 `ai_message.proto`，路径前缀为 `/api/v1/base/ai`：会话、快捷入口和消息列表由 `AiService` 提供，发送、删除、重试和重新生成由 `AiMessageService` 提供。发送消息使用会话级 SSE 响应；它不占用工作台通用 `/events` 流。

会话与消息分别持久化为 `ai_session`、`ai_message`。每轮消息保存输入、输出、附件、工具调用、Token、首 Token 耗时、总耗时和生成状态。失败消息可重试，成功输出可重新生成，任意锚点消息可创建持久化分支会话。

后端运行时位于 `service/base/agent/ai`，Eino 适配层位于 `pkg/agent/eino`。运行时按终端和 `agent_enabled` 挑选生成工具，再通过同一进程内服务实例执行；MCP 是否暴露由独立的 `mcp_enabled` 控制。公开实时信息可使用 Responses 的联网搜索工具，评价审核和摘要仍使用独立的结构化模型调用链。

## 管理后台设计

管理后台入口位于 frontend/admin/src/views/base/ai/chat，主要组件包括：

| 组件 | 职责 |
| --- | --- |
| index.vue | 会话列表、消息加载、发送、流式回复、流程动作分发 |
| ChatPanel.vue | 空态、消息列表、快捷操作、输入框、消息级操作 |
| SessionPanel.vue | 会话搜索、创建、切换、重命名和删除 |
| FlowBlocks.vue | 渲染结构化流程卡片和动作按钮 |
| XSender.vue | 文本、附件和发送状态 |

### 页面布局

~~~mermaid
flowchart TB
  Page["AI 助手页面"] --> Session["左侧会话栏"]
  Page --> Chat["右侧聊天区"]
  Chat --> Empty["空态欢迎区"]
  Chat --> Messages["消息列表"]
  Chat --> Quick["快捷操作区"]
  Chat --> Sender["输入区"]
~~~

- 空态展示欢迎语、快捷操作面板和输入框，适合新会话快速开始。
- 已有消息时，快捷操作以紧凑横向入口展示在输入框上方，避免只在第一次会话可见。
- 后台页面强调扫描效率，快捷项展示标题、业务分组和工具依赖数量，不使用大面积营销式视觉。
- 消息区保留工具调用、运行明细、复制、删除、编辑、重试和分支会话等管理能力。

### 快捷操作规则

- 快捷操作是否展示由“快捷入口加载中或已有可用入口”决定，不依赖当前会话是否为空。
- 空态面板使用两列网格，突出“从常用工作流开始”。
- 会话中使用紧凑横向滚动条，降低占屏高度，同时保持随时可触达。
- 点击快捷操作复用普通发送链路，统一调用 handleShortcutClick 并补齐 AiAction。

## 商城端设计

商城端入口位于 frontend/app/src/pagesMember/ai，面向 H5 和小程序端：

| 组件 | 职责 |
| --- | --- |
| index.vue | 导航栏、会话抽屉、消息流、快捷操作、流程动作和端侧交互 |
| WelcomePanel.vue | 首次进入或空会话欢迎区和快捷入口 |
| Composer.vue | 文本、附件、语音和发送入口 |
| SessionDrawer.vue | 移动端会话列表抽屉 |
| FlowBlocks.vue | 商品、地址、评价、订单等结构化流程 |

### 页面布局

~~~mermaid
flowchart TB
  Page["商城端 AI 助手"] --> Nav["顶部导航"]
  Page --> Body["滚动消息区"]
  Body --> Welcome["空态欢迎面板"]
  Body --> ThreadQuick["会话内快捷操作条"]
  Body --> Messages["用户 / 助手消息"]
  Page --> Composer["底部输入区"]
  Page --> Drawer["会话抽屉"]
~~~

- 空会话展示欢迎头像、问候语和“您可以这样问”快捷卡片。
- 已有消息时，在消息列表顶部展示横向快捷操作条，解决只有首次会话能看到快捷入口的问题。
- 快捷操作支持本地分页换一换，减少移动端纵向空间占用。
- 发送中、录音中、附件上传中或会话加载中时，快捷操作进入禁用态，避免重复提交。

### 移动端交互规则

- 操作区优先使用横向滚动和短标题，避免遮挡聊天内容。
- 输入区固定在底部，消息区通过底部留白避免最后一条消息被覆盖。
- 流程块在消息内闭环展示，涉及地址、商品规格、评价等表单时由端侧收集输入，再通过流程动作提交。
- H5 和小程序均应保持按钮、滚动容器和弹层能力可用，平台敏感能力使用 uni-app 能力封装。

## 数据与状态约定

| 数据 | 来源 | 端侧处理 |
| --- | --- | --- |
| 快捷入口 | ListAiShortcut | 按 sort 排序，过滤无 key 且无标题 / prompt 的异常项 |
| 会话列表 | ListAiSession | 支持创建、切换、搜索、重命名、删除 |
| 消息列表 | ListAiMessage | 按当前会话缓存，切换会话时加载 |
| 流式回复 | 消息流接口 | 合并增量，维护生成态和失败态 |
| 流程动作 | 助手消息 blocks | 点击后携带 action 回传消息接口 |

快捷入口的展示状态只和快捷入口数据有关，不应绑定 messages.length === 0。空态和会话中可以使用不同视觉密度，但必须复用同一批终端快捷入口和同一套发送链路。

流程动作必须回传 `source_message_id`、`action_id`、`flow_version`。服务端只接受当前会话最新成功消息的 `blocks_json` 中仍然存在的动作，阻止历史消息中的表单、支付或订单操作被重复执行。

## 维护与验证

- 修改管理后台 AI 助手页面后，在 frontend/admin 执行 pnpm lint:oxlint，必要时执行 pnpm type:check。
- 修改商城端 AI 助手页面后，在 frontend/app 执行 pnpm lint 和 pnpm tsc。
- 涉及协议字段变化时，由后端重新生成 frontend/admin/src/rpc 与 frontend/app/src/rpc，不要手工修改生成类型。
- 涉及快捷入口种类、分组或默认提示调整时，应同步检查两端展示是否仍然能容纳最长标题。
- 页面视觉改动应至少检查空会话、有消息会话、发送中和快捷入口加载失败后的状态。
