# 业务域目录拆分设计（Proto / Service / Frontend）

> 目标：将 `backend/api/protos/`、`backend/service/`、`frontend/admin/` 从「端优先」
> （admin/app 混装所有业务）重构为「业务域优先」，
> 使 shop 业务可整体删除，新业务可平行追加。
>
> 全栈统一分层：`base`（脚手架，仅上游同步）→ `common/system`（平台通用）→ `shop`（业务域，可插拔）。

## 一、目标结构

```
backend/api/protos/
  base/v1/                       # 保持不动：跨业务脚手架能力（仅从 kratos-kit 上游同步，本项目不改）
  common/v1/                     # 全局通用：错误、基础类型、通用选项（零业务语义）
  config/v1/                     # 框架配置模型

  system/
    common/v1/                   # 系统域共享：RBAC、租户、系统状态等枚举
    admin/v1/                    # 系统后台接口
    app/v1/                      # 系统用户端接口

  shop/
    common/v1/                   # 商城域共享：商品、订单、支付、推荐、评价、门店枚举与配置
    admin/v1/                    # 商城后台接口
    app/v1/                      # 商城端接口
```

## 二、依赖方向约束（必须遵守）

```
shop ──→ system/common ──→ common / base
```

- 允许：`shop` import `system/common`、`common`、`base`
- 禁止：`system` 反向 import `shop`（否则"整体删除"失效）
- buf 原生 lint 无法约束依赖方向，靠 code review 把关；后续可考虑将各业务域拆为独立 buf module

## 三、文件归属清单

### 3.1 保持不动

| 目录 | 文件 | 说明 |
|---|---|---|
| `base/v1/` | ai_assistant_message, ai_assistant_session, config, file, login, mcp, oauth, sse | 脚手架能力，仅上游同步 |
| `common/v1/` | error.proto | 纯全局错误码（400/401/403/404/409/500），无业务语义 |
| `common/v1/` | types.proto | 基础类型包装（XxxValues、PasswordCrypto） |
| `common/v1/` | analytics.proto | 通用图表结构（趋势/排行/饼图），无业务语义 |
| `config/v1/` | 框架配置部分 | 仅保留框架配置 |

### 3.2 system/（系统域）

| 目标 | 来源 | 文件 |
|---|---|---|
| `system/admin/v1/` | `admin/v1/` | auth, base_api, base_config, base_dept, base_dict, base_job, base_log, base_menu, base_role, base_user, base_tenant |
| `system/admin/v1/` | `admin/v1/` | code_gen, code_gen_column, code_gen_proto, code_gen_table |
| `system/app/v1/` | `app/v1/` | auth, base_dict, base_area |
| `system/common/v1/` | `common/v1/enum.proto` 拆出 | BaseConfigSite, BaseConfigType, BaseJobLogStatus, BaseMenuType, BaseRoleDataScope, BaseUserGender, PasswordCryptoScene |

### 3.3 shop/（商城域）

| 目标 | 来源 | 文件 |
|---|---|---|
| `shop/admin/v1/` | `admin/v1/` | goods_category, goods_info, goods_prop, goods_sku, goods_spec, goods_report, goods_analytics |
| `shop/admin/v1/` | `admin/v1/` | order_info, order_report, order_analytics, user_analytics, pay_bill |
| `shop/admin/v1/` | `admin/v1/` | comment_info, recommend_gorse, recommend_request |
| `shop/admin/v1/` | `admin/v1/` | shop_banner, shop_hot, shop_service, tenant_store, user_store, workspace |
| `shop/app/v1/` | `app/v1/` | goods_category, goods_info, order_info, pay, comment, recommend |
| `shop/app/v1/` | `app/v1/` | shop_banner, shop_hot, shop_service, user_address, user_cart, user_collect, user_store, tenant_store |
| `shop/common/v1/` | `config/v1/` | shop_config.proto（留在全局 config 会导致删 shop 删不干净） |
| `shop/common/v1/` | `common/v1/enum.proto` 拆出 | Goods*, Order*, Pay*, Shop*, Recommend*, Comment*, UserStoreStatus, TenantStoreStatus |
| `shop/common/v1/` | `common/v1/common.proto` 拆出 | CommentSummaryContentItem |

### 3.4 enum.proto 全局残留

`Status`、`Terminal`、`ResourceType`、`AdvanceDataType` 留在 `common/v1/`；
`Sse*` 枚举可跟随 `base/v1/sse.proto` 移入 base，或留全局，取其一保持一致。

## 四、边界判定说明

| 文件 | 归属 | 理由 |
|---|---|---|
| workspace | shop | 工作台聚合的是评价审核、口碑洞察等商城数据，删 shop 后无存在意义；未来若需"各业务自带工作台卡片"再抽 system 级聚合框架 |
| user_address / user_cart / user_collect | shop | 服务于收货/购物场景，脱离商城无用 |
| user_store / tenant_store | shop | 门店是商城业务概念（门店认证、租户门店树），非平台账户体系 |
| user_analytics | shop | 分析对象是商城客户 |
| code_gen 系列 | system | 是本项目的管理功能而非上游脚手架（base 只放上游同步内容），模板需改造支持按业务域输出 |

## 五、迁移连带工程量

1. **package 改名**：`admin.v1` → `system.admin.v1` / `shop.admin.v1`（app 同理）。
   所有 proto import 路径、生成的 Go pb 引用、前端 TS client 全部重新生成并修改引用。
2. **HTTP 路由不受影响**：路由是显式字面量（如 `/api/v1/admin/goods/info`），权限码同理。
3. **enum.proto 拆分**：几十个文件的 `import "common/v1/enum.proto"` 需改为对应域的 common 路径。
4. **buf 配置**：核对 `buf.yaml`、`buf.gen.yaml`、`buf.openapi.gen.yaml`、
   `buf.admin.typescript.gen.yaml`、`buf.app.typescript.gen.yaml` 中的路径与输出规则。
5. **代码生成器**：code_gen 模板中硬编码的 `admin/v1` 输出路径与 `package admin.v1` 需参数化，
   否则新生成代码会回流旧结构。

## 六、backend/service 拆分

### 6.1 目标结构

```
backend/service/
  base/                          # 保持不动：脚手架能力（AI 助手、文件、登录、MCP、OAuth、SSE）

  system/
    admin/                       # 系统后台服务（含 biz/、dto/）
    app/                         # 系统用户端服务（含 biz/）

  shop/
    admin/                       # 商城后台服务（含 biz/、dto/、utils/）
    app/                         # 商城端服务（含 biz/、dto/、utils/）
```

Go 包名约定：目录 `service/system/admin` 与 `service/shop/admin` 包名均为 `admin`，
在 `server/` 装配处用 import 别名区分：`systemadmin`、`shopadmin`（app 同理）。

### 6.2 文件归属清单

**service/admin/ → system/admin/**

| 类别 | 文件 |
|---|---|
| 服务+biz | auth, base_api, base_config, base_dept, base_dict(+item), base_job(+log), base_log, base_menu, base_role, base_tenant, base_user |
| 服务+biz | code_gen, code_gen_column, code_gen_proto, code_gen_table |
| biz | casbin_rule |
| dto | code_gen |

**service/admin/ → shop/admin/**

| 类别 | 文件 |
|---|---|
| 服务+biz | goods_*(analytics/category/info/prop/report/sku/spec) |
| 服务+biz | order_*(analytics/info/report)，biz 另含 order_address/cancel/goods/logistics/paid_fact/payment/refund |
| 服务+biz | comment_info，biz 另含 comment_discussion/review/status/summary/tag |
| 服务+biz | pay_bill, recommend_gorse, recommend_request(+item biz), recommend_event(biz) |
| 服务+biz | shop_banner, shop_hot(+item biz), shop_service, tenant_store, user_store, user_analytics, workspace |
| dto | analytics, comment_info, goods_analytics, goods_report, order_report, recommend_request |
| utils | analytics |

**service/app/ → system/app/**

| 类别 | 文件 |
|---|---|
| biz | auth, base_area, base_dept, base_dict(+item), base_role, base_user |

**service/app/ → shop/app/**

| 类别 | 文件 |
|---|---|
| biz | goods_*, order_*(含 scheduler/trade), comment_*, pay, recommend_* |
| biz | shop_banner, shop_hot(+item), shop_service, tenant_store, user_address, user_cart, user_collect, user_store |
| dto | comment, order |
| utils | util（IsMember 会员判断，会员是商城概念）, wx（微信手机号授权） |

**边界说明**：`app/biz/auth.go` 归 system（登录属账户体系），但其中微信 access_token
缓存与微信登录渠道耦合了小程序形态，拆分时应将微信渠道逻辑下沉到 `pkg/wx` 或随 shop 走，待迁移时核对。

### 6.3 server 装配层连带

`backend/server/services.go` 中 `ServerServices` 混装全部服务，需按域拆文件：

- `services_system.go` / `services_shop.go`（结构体字段、构造、注册辅助各归其域）
- `grpc.go`、`http.go`、`mcp.go` 中的注册代码块按域分组
- 删除 shop = 删 `service/shop/` + `services_shop.go` + 各注册文件中的 shop 分组块

### 6.4 pkg/ 归属标注（不强制搬迁，但需明确随谁删除）

| 包 | 归属 | 说明 |
|---|---|---|
| `pkg/biz/order_inventory.go`、`order_refund_result.go` | shop | 跨端共享的订单逻辑，建议移入 `service/shop/` 共享层或 `pkg/shop/` |
| `pkg/biz/base_api.go`、`casbin_rule.go` | system | RBAC 支撑 |
| `pkg/recommend`、`pkg/wx`、`pkg/workspaceevent` | shop | 商城域支撑包，删 shop 时一并删除 |
| `pkg/gen/models`、`pkg/gen/query` | 生成物 | 平铺不拆；在 gorm gen 配置中按域分组表清单，删 shop 时去掉商城表重新生成 |
| `pkg/agent`、`pkg/codegen`、`pkg/config`、`pkg/job`、`pkg/queue`、`pkg/middleware` 等 | 通用 | 保持不动 |

## 七、frontend/admin 拆分

### 7.1 目标结构

```
frontend/admin/src/
  rpc/                           # buf 生成物：proto 拆分后自动变为
    base/  common/  config/      #   rpc/system/admin/v1、rpc/shop/admin/v1
    system/admin/v1/
    shop/admin/v1/

  api/
    base/                        # 保持不动（脚手架：AI、文件、登录、SSE 等）
    system/                      # auth, base_*, code_gen*
    shop/                        # goods*, order*, comment_info, pay_bill, recommend*,
                                 # shop_*, tenant_store, user_store, *_analytics, *_report, workspace

  views/
    login/  profile/  migration/ # 全局壳层，保持不动
    ai/                          # base 脚手架（AI 助手），保持不动
    system/
      base/                      # api/config/dept/dict/job/log/menu/role/tenant/user
      tool/code-gen/
    shop/
      dashboard/                 # workspace + analytics（聚合的全是商城数据）
      goods/  order/  comment/  pay/  recommend/  report/  shop/  user/
```

### 7.2 归属判定

| 项 | 归属 | 说明 |
|---|---|---|
| `views/dashboard/` | shop | workspace 与 analytics 展示的均为商城指标；删 shop 后需给系统一个兜底首页（可指向 profile 或简单欢迎页） |
| `stores/modules/recommendGorse.ts` | shop | 其余 store（auth/config/dict/global/tabs/keepAlive/user）均为全局 |
| `components/` | 全局 | 均为通用组件，shop 专用组件已内聚在各 views 下（如 comment/components） |
| `routers/` | 全局 | 静态路由不动；动态路由来自后端菜单数据（见 7.3） |

### 7.3 关键连带：菜单数据

`dynamicRouter.ts` 通过 `import.meta.glob("@/views/**/*.vue")` 按 `base_menu` 表中的
`component` 路径解析组件（`/src/views/${component}.vue`）。**views 目录搬迁必须同步：**

1. 更新 `base_menu` 表存量数据的 `component` 路径（`goods/info/index` → `shop/goods/info/index` 等）
2. 更新权限/菜单初始化种子数据
3. `pkg/codegen/menu.go` 生成的 Component 路径按业务域参数化

### 7.4 代码生成器连带

`pkg/codegen/frontend.go`、`api.go` 模板中硬编码的输出路径
（`src/api/admin/`、`src/views/xxx/`、`src/rpc/admin/v1`）需增加业务域参数，
否则新生成页面会回流旧结构。

> frontend/app（商城小程序端）本身整体属于 shop 域，暂不需要内部再拆；若未来新增业务的用户端，再平行新建。

## 八、删除 shop 的整体清单（验收标准）

```
backend/api/protos/shop/          整个目录
backend/service/shop/             整个目录
backend/server/services_shop.go   及各注册文件中的 shop 分组块
backend/pkg/recommend、wx、workspaceevent
backend/pkg/biz 中 shop 归属文件；gorm gen 配置去掉商城表后重新生成
frontend/admin/src/api/shop/      整个目录
frontend/admin/src/views/shop/    整个目录
frontend/admin/src/rpc/shop/      重新 buf generate 即消失
frontend/admin/src/stores/modules/recommendGorse.ts
frontend/app/                     整个目录
数据库：商城表 + base_menu 中商城菜单数据
```

删完后 `system + base + common` 应能独立编译、启动、登录后台。

## 九、执行顺序建议

一次性做完一个阶段，中间状态编译不过没有意义：

**阶段一：Proto（本文档 一~五）**

1. 建目录、移文件，同步改 package 与 import
2. 拆分 enum.proto / common.proto / shop_config.proto
3. buf generate 重新生成全部代码

**阶段二：backend/service（本文档 六）**

4. 按 6.2 清单搬迁 service 文件，修复 Go 引用（配合新 pb 路径一次改完）
5. 拆分 server 装配层（6.3），标注/搬迁 pkg 归属（6.4）

**阶段三：frontend/admin（本文档 七）**

6. 重新生成 TS client，按 7.1 搬迁 api/ 与 views/，修复引用
7. 更新 base_menu 存量数据与初始化种子（7.3）

**阶段四：代码生成器**

8. code_gen 模板（proto/backend/frontend/menu）全链路增加业务域参数

**验收**：按第八节清单做一次「删 shop 演练」（分支上删除后编译启动），验证边界是否干净。
