# shop

`shop` 是一个前后端一体的商城项目，仓库内包含：

- `backend`：基于 Go + Kratos 的后端服务，同时提供 HTTP、gRPC、OpenAPI 文档、文件上传和静态资源能力。
- `frontend/admin`：基于 Vue 3 + Vite + Element Plus 的管理后台。
- `frontend/app`：基于 uni-app + Vue 3 + TypeScript 的商城端，支持微信小程序、H5 和 App 条件编译。
- `sql`：初始化数据、权限数据、地区数据和演示商品数据。

当前代码实现的业务模块主要覆盖：

- 管理端：用户、角色、部门、菜单、字典、配置、定时任务、日志。
- 商城端：商品分类、商品/SPU、规格、SKU、轮播图、热门推荐、商城服务。
- 交易链路：购物车、地址、订单、退款、支付、账单下载。
- 账号体系：后台账号密码登录、验证码、Token 刷新、微信小程序登录。

## 技术栈

### 后端

- Go `1.26`
- Kratos
- gRPC + HTTP
- GORM / GORM Gen
- MySQL
- Casbin
- Wire
- Buf / Protobuf / OpenAPI

### 前端

- 管理台：Vue 3、TypeScript、Vite、Element Plus、Pinia、Axios
- 商城端：uni-app、Vue 3、TypeScript、Pinia

## 目录结构

```text
.
├── backend                 # Go 后端
│   ├── internal/cmd/server # 服务真实入口
│   ├── service             # admin / app / base 业务服务
│   ├── server              # HTTP / gRPC 服务装配
│   ├── configs             # 运行配置
│   ├── api                 # proto、buf 配置、生成代码
│   └── pkg                 # 中间件、配置解析、模型与查询代码
├── frontend
│   ├── admin               # 管理后台
│   └── app                 # uni-app 商城端
├── sql                     # 初始化 SQL
└── AGENTS.md               # 仓库内协作约束
```

## 环境要求

- Go `1.26+`
- Node.js `18+`
- `pnpm` `8+` 或更高版本
- MySQL `8.x`

## 配置说明

后端启动命令使用 `-conf ./configs`，会读取 `backend/configs` 目录下的配置文件。最少需要关注以下文件：

| 文件 | 作用 | 说明 |
| --- | --- | --- |
| `backend/configs/data.yaml` | 数据库配置 | 默认库名为 `shop_test`，请改成自己的 MySQL 连接串 |
| `backend/configs/server.yaml` | HTTP/gRPC 端口 | HTTP 默认 `7001`，gRPC 默认 `6001` |
| `backend/configs/auth.yaml` | JWT 配置 | 包含白名单接口 |
| `backend/configs/oss.yaml` | 文件存储 | 默认 `type: local`，上传目录为 `./data/shop` |
| `backend/configs/configs.yaml` | 商城自定义配置 | 包含微信小程序与微信支付配置 |

说明：

- `backend/configs/configs.yaml` 中的微信小程序与微信支付字段在当前实现里要求非空，开发联调阶段可以先填占位值；真正使用微信登录和支付时再替换成真实配置。
- 仓库中存在 `backend/configs/configs_local.yaml`，建议按本地环境自行覆盖，不要直接依赖仓库里的现有值。
- 本地文件上传会暴露在 `/shop/*` 路径下，例如 `./data/shop/...` 会映射为 `http://localhost:7001/shop/...`。

## 数据库初始化

### 1. 创建数据库

先在 MySQL 中创建一个空库，例如：

```sql
CREATE DATABASE shop_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

然后修改 `backend/configs/data.yaml` 中的 `source`。

### 2. 启动一次后端自动建表

项目当前配置 `enable_migrate: true`，建议先启动一次后端，让 GORM 自动创建表结构：

```bash
cd backend
go run ./internal/cmd/server -conf ./configs
```

### 3. 导入初始化数据

回到仓库根目录后，按下面顺序导入 SQL：

```bash
mysql -uroot -p shop_test < sql/default-data.sql
mysql -uroot -p shop_test < sql/casbin_rule.sql
mysql -uroot -p shop_test < sql/base_area.sql
```

如需导入演示商品、分类和商品详情数据，再额外执行：

```bash
mysql -uroot -p shop_test < sql/shop.sql
```

### 4. 默认账号

导入 `sql/default-data.sql` 后，默认可用账号如下：

| 用户名 | 角色 | 密码       |
| --- | --- |----------|
| `super` | 超级管理员 | `112233` |
| `admin` | 管理员 | `112233` |

登录时还需要输入验证码，管理台和商城 H5 登录页都会自动请求验证码接口。

## 启动项目

### 启动后端

```bash
cd backend
go run ./internal/cmd/server -conf ./configs
```

默认端口：

- HTTP：`http://localhost:7001`
- gRPC：`localhost:6001`
- Swagger UI：`http://localhost:7001/docs/`
- OpenAPI：`http://localhost:7001/docs/openapi.yaml`

说明：

- 后端会扫描 `backend/data` 下包含 `index.html` 的一级子目录，并自动按目录名挂载单页应用，例如 `backend/data/admin/index.html` 会对应 `http://localhost:7001/admin`。
- 本地文件上传目录仍然是 `backend/data/shop`，通过 `http://localhost:7001/shop/*` 访问。

### 启动管理后台

```bash
cd frontend/admin
pnpm install
pnpm dev
```

默认开发地址：`http://localhost:5001`

管理台开发环境默认读取 `frontend/admin/.env.development`。

默认会把 `/api` 和 `/shop` 代理到 `http://localhost:7001`。

### 启动商城 H5

```bash
cd frontend/app
pnpm install
pnpm dev:h5
```

默认开发地址：`http://localhost:5002`

商城端开发环境默认读取：

- `frontend/app/.env.development`
- `frontend/app/.env.development-h5`

补充说明：

- `frontend/app/.env.development-h5` 中的 `VITE_APP_BASE_PATH` 默认为 `/`，便于本地直接访问开发服务。
- H5 页面中返回 `/shop/...` 这类静态资源路径时，会自动按当前站点根域名补全，避免在 `/app` 子路径部署后被解析成 `/app/shop/...`。

### 启动微信小程序

```bash
cd frontend/app
pnpm install
pnpm dev:mp-weixin
```

然后使用微信开发者工具导入构建目录。

### 使用 `frontend/Makefile`

仓库额外提供了 `frontend/Makefile`，用于统一管理前端依赖安装、开发启动和构建命令。

常用方式：

```bash
make -C frontend init
make -C frontend build
make -C frontend run-admin
make -C frontend run-app
```

目标说明：

- `make -C frontend init`：安装 `admin` 和 `app` 两端依赖。
- `make -C frontend build`：顺序执行管理台构建和商城端微信小程序构建。
- `make -C frontend build-admin`：仅构建管理台。
- `make -C frontend build-app`：仅构建商城端微信小程序。
- `make -C frontend run-admin`：启动管理台开发服务。
- `make -C frontend run-app`：启动商城端微信小程序开发服务。

说明：

- `frontend/Makefile` 中的命令统一附带 `CI=1`，用于关闭交互式单行刷新，控制台日志按多行输出。
- `make -C frontend run` 会顺序执行 `run-admin` 和 `run-app`，但 `run-admin` 启动后会持续占用当前终端，因此日常开发更建议分两个终端分别执行 `run-admin`、`run-app`。

## 常用命令

以下命令都在 `backend` 目录执行：

```bash
make init      # 安装 protoc / buf / wire / goimports 等工具
make fmt       # 格式化 Go 代码
make api       # 生成 proto 对应 Go 代码
make openapi   # 生成 OpenAPI 文档
make ts        # 生成前端 TypeScript RPC 代码
make gen       # 一键生成 Go/OpenAPI/TS 产物
make wire      # 生成依赖注入代码
make docker-build # 构建后端 Docker 镜像，默认镜像名 backend:latest
```

说明：

- `backend/Makefile` 中的 `run` 目标仍指向 `./cmd/server`，和当前代码入口不一致。
- 当前仓库实际可用的后端启动命令是 `go run ./internal/cmd/server -conf ./configs`。

## Docker 打包

后端目录已提供多阶段 `Dockerfile`，默认可直接在 `backend` 目录构建镜像：

```bash
cd backend
make docker-build
```

如需自定义镜像名和标签：

```bash
cd backend
make docker-build IMAGE=your-registry/backend TAG=v1.0.0
```

容器运行时建议将以下目录从宿主机挂载出来，便于修改配置、更新证书以及持久化日志和本地上传文件：

- `data` -> `/app/data`
- `configs` -> `/app/configs`
- `certs` -> `/app/certs`

示例：

```bash
cd backend
docker run -d \
  -p 7001:7001 \
  -p 6001:6001 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/configs:/app/configs \
  -v $(pwd)/certs:/app/certs \
  backend:latest
```

## 前端构建与嵌入发布

### 管理台构建

```bash
cd frontend/admin
pnpm install
pnpm build
```

管理台构建产物默认输出到：

- `backend/data/admin`

### 商城 H5 构建

```bash
cd frontend/app
pnpm install
pnpm build:h5
```

商城 H5 构建产物同样默认输出到：

- 自定义 `UNI_OUTPUT_DIR` 后可输出到 `backend/data/app`

补充说明：

- `frontend/app/.env.production-h5` 中的 `VITE_APP_BASE_PATH` 默认为 `/app/`，对应后端自动挂载出的 `http://localhost:7001/app` 路径。
- 如需把静态资源明确指向独立域名或端口，可在 `frontend/app/.env.production-h5` 中设置 `VITE_APP_STATIC_URL`，例如 `http://localhost:7001`。

### 注意事项

- 后端会自动扫描 `backend/data` 下的一级子目录，只有目录内存在 `index.html` 时才会注册对应路由。
- 如果需要同时发布多个 Web 端，请分别输出到不同目录，例如 `backend/data/admin`、`backend/data/app`。
- 前端静态资源走本地文件系统，重新构建前端后重启后端即可加载新的目录内容。

## 接口与静态资源

后端当前暴露的关键入口：

- `/api/*`：HTTP 业务接口
- `/docs/`：Swagger UI
- `/docs/openapi.yaml`：OpenAPI 文档
- `/shop/*`：本地文件上传后的静态资源
- `/{目录名}`：`backend/data/{目录名}/index.html` 对应的单页应用入口，例如 `/admin`、`/app`

已注册的服务分组：

- `base`：公共配置、登录、文件上传
- `admin`：管理端业务接口
- `app`：商城端业务接口

## 开发建议

- 后端命令请在 `backend` 目录执行，前端命令分别在 `frontend/admin`、`frontend/app` 目录执行。
- 如果改了 `proto`，按 `make gen` 重新生成 Go、OpenAPI 和 TypeScript 代码。
- 如果改了 Wire 装配，执行 `make wire`。
- 如果只是本地开发文件上传，保持 `backend/configs/oss.yaml` 为 `local` 即可。
