# gorse

`gorse` 目录用于本地启动 Gorse 推荐服务，方便后端推荐链路、管理后台 Gorse 推荐页面、推荐调试和推荐编排联调。

## 目录职责

```text
gorse
├── config
│   └── config.toml      # Gorse 本地配置
├── data                 # gorse 运行数据目录
└── docker-compose.yml   # 本地容器启动配置
```

## 环境要求

- `Docker` 或 `Docker Desktop`
- 本机可访问的 `MySQL 8.x`
- 已创建推荐库 `shop_gorse`

创建推荐库：

```sql
CREATE DATABASE shop_gorse CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

## 配置说明

`config/config.toml` 当前默认使用 MySQL 作为数据存储和缓存存储：

```toml
data_store = "mysql://root:112233@tcp(host.docker.internal:3306)/shop_gorse?charset=utf8mb4&parseTime=True&loc=Local&timeout=1000ms"
cache_store = "mysql://root:112233@tcp(host.docker.internal:3306)/shop_gorse?charset=utf8mb4&parseTime=True&loc=Local&timeout=1000ms"
```

`docker-compose.yml` 已配置：

```yaml
extra_hosts:
  - "host.docker.internal:host-gateway"
ports:
  - "8086:8086"
  - "8088:8088"
```

默认端口：

- `8086`：gRPC / 主服务端口
- `8088`：HTTP API 端口

后台和后端调用推荐管理接口时，需要使用 HTTP API 端口。

## 后端联调配置

后端本地配置位于：

```text
backend/configs/configs_local.yaml
```

需要保证其中的推荐配置与 `config/config.toml` 保持一致：

```yaml
shop:
  recommend:
    entryPoint: http://127.0.0.1:8088
    apiKey: ...
```

- `entryPoint` 指向 Gorse HTTP API 端口，即 `8088`。
- `apiKey` 需要与 `config/config.toml` 中的 API Key 配置一致。
- 默认凭据只适合本地开发，生产环境应改为独立密钥并避免提交到仓库。

## 启动与停止

启动：

```bash
cd gorse
docker compose up -d
```

查看状态：

```bash
cd gorse
docker compose ps
```

查看日志：

```bash
cd gorse
docker compose logs -f
```

停止：

```bash
cd gorse
docker compose down
```

## 推荐数据链路

当前项目的推荐链路主要包含：

- 后端通过定时任务同步用户、商品等主数据到 gorse。
- 商城端上报推荐请求、曝光、点击、浏览等前端行为。
- 收藏、加购、下单、支付等业务事件由后端在真实业务落库成功后回写。
- 管理后台通过后端代理访问 Gorse 推荐概览、任务、用户、商品、相似内容、反馈、高级调试、推荐编排和推荐配置能力。

推荐事件和推荐请求也会落在主业务库中，便于后台排查请求、结果和行为归因链路。

## 设计文档

| 文档 | 说明 |
| --- | --- |
| [推荐系统设计](../docs/推荐系统设计.md) | 项目推荐场景、Gorse 责任链、本地兜底和后台管理能力。 |
| [推荐数据流转设计](../docs/推荐数据流转设计.md) | 用户 / 商品同步、推荐请求、反馈事件和后台排查链路。 |
| [数据库与初始化数据设计](../docs/数据库与初始化数据设计.md) | `shop_gorse` 推荐库与主业务库的职责边界。 |

## 重置本地环境

如果需要完全重置本地推荐环境，先停止容器，再清理 `gorse/data` 和 `shop_gorse` 数据库后重新启动。这个操作会删除本地推荐训练数据、缓存和运行状态，只建议在本地开发环境执行。

```bash
cd gorse
docker compose down
```
