# geeker-admin

`geeker-admin` 是 `shop` 项目的后台管理前端，基于 `Vue 3 + Vite + TypeScript + Element Plus + Pinia` 构建，负责商城后台的登录认证、权限菜单、基础数据维护、商品管理、订单管理、店铺运营和支付账单等功能。

项目当前采用后端动态菜单驱动路由，前端按接口返回的菜单树动态注册页面组件；认证链路使用 `access token + refresh token`，在令牌即将过期时自动刷新，适配当前商城后台的权限体系。

## 技术栈

- `Vue 3`
- `Vite 5`
- `TypeScript 5`
- `Element Plus`
- `Pinia`
- `Vue Router`
- `Axios`
- `ECharts`
- `WangEditor`
- `pnpm`

## 运行要求

- `Node.js >= 16.18.0`
- `pnpm`

建议直接使用仓库内已经约定的 `pnpm` 作为包管理器。

## 快速开始

### 1. 安装依赖

```bash
pnpm install
```

### 2. 启动开发环境

```bash
pnpm dev
```

默认行为：

- 本地端口：`8848`
- 自动打开浏览器：`true`
- 开发路由模式：`hash`
- 开发接口前缀：`/api`

### 3. 代码检查

```bash
pnpm lint:eslint
pnpm type:check
```

如需补充格式化与样式检查，可执行：

```bash
pnpm lint:prettier
pnpm lint:stylelint
```

## 构建与预览

### 开发环境构建

```bash
pnpm build:dev
```

### 测试环境构建

```bash
pnpm build:test
```

### 生产环境构建

```bash
pnpm build:pro
```

### 本地预览

```bash
pnpm preview
```

## 构建产物说明

当前 `vite.config.ts` 已将构建输出目录固定为：

```text
../../backend/data/geeker
```

这意味着前端打包结果会直接写入 `shop` 仓库下的后端静态资源目录，而不是当前前端目录内的 `dist`。在联调或发布时，需要同时关注后端目录中的静态文件是否已被正确覆盖。

生产环境公共访问前缀当前配置为：

```text
/geeker/
```

如果后端静态资源挂载路径调整，需要同步修改 `.env.production` 中的 `VITE_PUBLIC_PATH`。

## 环境变量

项目使用 `.env`、`.env.development`、`.env.production` 管理环境配置。

### 通用变量

| 变量名 | 说明 | 当前值 |
| --- | --- | --- |
| `VITE_GLOB_APP_TITLE` | 应用标题 | `Geeker Admin` |
| `VITE_PORT` | 本地开发端口 | `8848` |
| `VITE_OPEN` | 启动时自动打开浏览器 | `true` |
| `VITE_DEVTOOLS` | 是否开启 Vue DevTools | `false` |
| `VITE_REPORT` | 是否生成打包分析报告 | `false` |
| `VITE_CODEINSPECTOR` | 是否开启 Code Inspector | `false` |

### 开发环境变量

| 变量名 | 说明 | 当前值 |
| --- | --- | --- |
| `VITE_USER_NODE_ENV` | 环境标识 | `development` |
| `VITE_PUBLIC_PATH` | 公共访问路径 | `/` |
| `VITE_ROUTER_MODE` | 路由模式 | `hash` |
| `VITE_DROP_CONSOLE` | 构建时移除 `console` | `true` |
| `VITE_PWA` | 是否启用 PWA | `false` |
| `VITE_API_URL` | 开发接口基础地址 | `/api` |
| `VITE_PROXY` | 开发代理配置 | `[["/api","http://localhost:7001"],["/shop","http://localhost:7001"]]` |

### 生产环境变量

| 变量名 | 说明 | 当前值 |
| --- | --- | --- |
| `VITE_USER_NODE_ENV` | 环境标识 | `production` |
| `VITE_PUBLIC_PATH` | 公共访问路径 | `/geeker/` |
| `VITE_ROUTER_MODE` | 路由模式 | `hash` |
| `VITE_BUILD_COMPRESS` | 构建压缩方式 | `none` |
| `VITE_BUILD_COMPRESS_DELETE_ORIGIN_FILE` | 压缩后是否删除源文件 | `false` |
| `VITE_DROP_CONSOLE` | 构建时移除 `console` | `true` |
| `VITE_PWA` | 是否启用 PWA | `true` |
| `VITE_API_URL` | 生产接口基础地址 | `/api` |

## 开发代理说明

开发环境下，Vite 会将下列请求代理到本地后端：

- `/api -> http://localhost:7001`
- `/shop -> http://localhost:7001`

如果后端联调端口变化，优先修改 `.env.development` 中的 `VITE_PROXY`。

## 目录结构

```text
geeker-admin
├── build                  Vite 构建扩展配置
├── public                 静态资源
├── src
│   ├── api                基于 axios 的业务接口封装
│   ├── assets             图片、字体、svg 图标等静态资源
│   ├── components         通用业务组件
│   ├── config             全局配置
│   ├── directives         自定义指令
│   ├── enums              枚举定义
│   ├── hooks              组合式 hooks
│   ├── layouts            多布局容器与头部/标签栏等框架组件
│   ├── routers            静态路由与动态路由初始化
│   ├── rpc                接口类型与生成代码
│   ├── stores             Pinia 状态管理
│   ├── styles             全局样式与主题变量
│   ├── utils              请求封装、工具函数、权限辅助等
│   └── views              页面级业务模块
├── .env*                  环境变量配置
├── package.json           脚本与依赖声明
└── vite.config.ts         Vite 主配置
```

## 业务模块

当前仓库内已经落地的页面模块主要包括：

- 登录与个人中心
- 工作台、数据看板
- 用户、角色、菜单、部门、岗位、日志、字典、系统配置
- 门店管理
- 商品分类、商品信息、规格、属性、SKU
- 订单管理、发货处理
- 店铺轮播图、热门推荐、服务配置
- 支付账单

首页默认路由为：

```text
/dashboard/workspace
```

## 权限与路由

- 静态路由仅保留登录页、布局页和错误页。
- 业务页面路由主要由后端接口动态返回。
- 前端启动后会拉取用户菜单和按钮权限，再按菜单配置动态注册路由。
- 菜单组件路径通过 `src/views` 下的页面文件自动匹配，未匹配到的页面会回退到待迁移页面占位逻辑。
- 隐藏业务页跳转统一复用 `src/utils/router.ts` 中的 `navigateTo`，避免各页面重复维护 `router.push + resolve` 降级逻辑。
- 列表页跳转到隐藏业务页时，前端应优先按完整 `path` 跳转，并与 `/Users/liujun/workspace/shop/shop/sql/geeker-admin.sql` 中 `base_menu` 的父子菜单路径保持一致；当前已校验的隐藏页包括 `/base/dict-item`、`/base/job-log`、`/goods/edit`、`/order/detail/:orderId`、`/shop/hot-item`。

相关接口：

- `GET /admin/auth/userInfo`
- `GET /admin/auth/menu`
- `GET /admin/auth/button`

## 认证机制

登录相关接口位于 `/login` 命名空间下，主要包括：

- `GET /login/captcha`
- `POST /login`
- `POST /login/refreshToken`
- `DELETE /login/logout`

当前前端认证处理策略：

- 登录成功后缓存 `access token`、`refresh token`、`tokenType` 和过期时间。
- 请求发送前，如果令牌剩余有效期小于 5 分钟，会自动调用刷新接口。
- 遇到 `401` 或 `403` 响应时，会提示重新登录并清理本地认证信息。

## 常用脚本

| 命令 | 说明 |
| --- | --- |
| `pnpm dev` | 启动开发服务器 |
| `pnpm serve` | 启动开发服务器，等同于 `pnpm dev` |
| `pnpm build` | 执行生产构建，等同于 `pnpm build:pro` |
| `pnpm build:dev` | 开发环境构建 |
| `pnpm build:test` | 测试环境构建 |
| `pnpm build:pro` | 生产环境构建 |
| `pnpm preview` | 构建后本地预览 |
| `pnpm type:check` | TypeScript 类型检查 |
| `pnpm lint:eslint` | ESLint 检查并自动修复 `src` 下代码 |
| `pnpm lint:prettier` | Prettier 格式化 |
| `pnpm lint:stylelint` | Stylelint 检查并自动修复样式 |

## 开发约定

- 包管理器使用 `pnpm`。
- 路径别名 `@` 默认指向 `src`。
- 路由模式当前统一使用 `hash`。
- 所有页面组件集中放在 `src/views`，动态菜单的组件路径也依赖该目录结构。
- 接口请求统一走 `src/utils/request.ts`，并在其中处理鉴权头、错误提示和令牌刷新。
- 类型定义与接口描述主要位于 `src/rpc`。

### 列表页约定

- 后台列表页优先参考 [`src/views/shop/banner/index.vue`](./src/views/shop/banner/index.vue) 的实现方式，统一采用 `ProTable + FormDialog + ProForm` 结构。
- `src/views/base/menu/index.vue` 这类树形列表页也按上述结构收敛，顶部按钮使用 `headerActions`，状态列和操作列优先使用 `cellType` 与 `actions` 配置，仅保留图标选择、穿梭项展示等必要插槽。
- 详情查看类弹窗若明显不属于表单场景，优先使用 `ProDialog`，不要直接回退到原生 `el-dialog`。
- 弹窗内若同时包含详情展示和少量编辑字段，保留 `ProDialog` 承载展示区，表单部分优先改用 `ProForm`；当前订单发货/退款弹窗已按该模式收敛，纯表单弹窗则继续优先使用 `FormDialog`。
- 顶部按钮优先通过 `headerActions` 配置，行内操作优先通过 `actions` 配置；只有基础配置无法覆盖时，才保留必要的页面 slot。
- 状态列优先使用 `cellType: "status"`，图片列优先使用 `cellType: "image"`，避免在页面模板中重复手写通用展示逻辑。
- 弹窗关闭时必须显式重置表单与校验状态；编辑态回填、选项预加载、提交成功后刷新表格都应在页面方法内明确处理。
- 业务确认与提示文案优先体现明确对象，单项确认推荐使用“是否确定执行动作？\n字段中文名：字段值”，成功/取消提示也应尽量带上业务对象，例如“删除角色成功”“已取消删除角色”。

## 维护建议

- 新增页面时，优先保持与后端菜单配置中的组件路径一致，避免动态路由无法命中组件文件。
- 修改部署路径、代理地址或接口前缀时，同时检查 `.env*`、`vite.config.ts` 和后端静态资源挂载配置。
- 如果后端输出的菜单路径采用相对路径，当前前端已内置路径规范化逻辑，但仍建议后端保持路径定义风格统一。
