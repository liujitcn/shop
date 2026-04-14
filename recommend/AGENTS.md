# Codex 规则

## 适用范围
- 本规则适用于当前目录 `shop/recommend` 及其全部子目录。

## 模块定位
- `recommend` 是商城推荐工具模块，不是独立服务。
- 当前模块不提供 HTTP、gRPC、CLI、进程启动入口，不新增 `cmd`、`server`、`wire`、`bootstrap` 这类服务化目录。
- `backend` 负责对外接口、认证、事务、业务事实落库；`recommend` 负责推荐算法、缓存、离线构建与评估。

## 目录约定
- `contract` 只定义外部数据契约，由 `backend` 实现，不在当前模块中直接依赖 `backend/service` 或 `backend/pkg/gen`。
- `api/protos/recommend/v1/recommend.proto` 仅用于定义 LevelDB value 的 `message`，不要在当前模块中定义 `service`。
- `api/gen` 为 `proto` 生成产物目录；涉及协议变更时，优先通过模块内命令生成，不手写伪生成结果。
- `internal/scene`、`internal/recall`、`internal/rank`、`internal/replace` 必须优先保持商城业务语义，避免为了贴近通用引擎而弱化 `home`、`goods_detail`、`cart`、`order_paid` 等业务场景表达。

## 设计约定
- 对外入口优先采用包级方法，不使用 `NewXXX` 构造器作为唯一入口。
- 新增能力时，优先复用现有 `recommend_*` 表与商城业务表，不额外抽象通用 `user/item/feedback` 平台。
- 缓存统一以 `LevelDB` 为核心实现；当前设计固定为 `pool.db`、`runtime.db`、`trace.db` 三类库，不要无必要继续拆散。
- 新增或调整推荐能力时，必须同时说明对应的场景、召回来源、排序信号、replacement/fallback 策略和评估口径。

## 代码规范

当前模块新增或修改代码时，除遵循仓库根目录规则外，还必须同步遵循 `shop/backend/AGENTS.md` 中与代码质量直接相关的约束。下面四类规则在当前模块内视为强制规则，不依赖当前会话窗口：

### 注释规范
- 代码注释统一使用中文。
- 新增或修改的方法，必须补充中文方法注释。
- 关键逻辑、边界条件、异常分支需要补充必要的中文行内注释。
- `if`、`switch` 分支涉及业务语义、边界处理、降级逻辑时，必须补充中文注释；常规 `err` 判断和普通判空分支可不额外注释。
- `proto` 中每个 `message` 必须补充中文注释，每个字段必须保留中文尾注释。

### 代码修改约束
- 生成代码禁止手写修改，包括 `api/gen` 下的协议产物。
- 变量必须就近声明，禁止在方法开头堆积无关局部变量。
- 同一个方法内，`err` 只允许初始化一次；首次初始化后只能复用 `err =`，禁止重复使用 `err :=`。
- 只要 `:=` 左侧任意变量已经声明过，就禁止继续使用 `:=`，必须改用 `=`。
- 方法顺序必须保持“外部入口在前，内部辅助在后”，按真实调用链从上到下展开。
- 不要拆无意义的子方法；只被单一分支调用、且不能提升可读性的辅助逻辑，不要额外下沉成新方法。

### 变量命名规则
- Go 标识符统一使用驼峰命名。
- 标识符中禁止使用 `ID`、`IDs`、`URL`、`HTTP`、`SKU` 这类全大写缩写片段，统一使用 `Id`、`Ids`、`Url`、`Http`、`Sku` 写法。
- 布尔变量优先使用 `is`、`has`、`can`、`should` 这类语义前缀。
- 变量名必须见名知意，避免使用无业务语义的缩写。

### Proto 接口契约命名规则
- 当前模块 `api/protos/recommend/v1/recommend.proto` 只定义 `message`，不定义 `service`。
- `proto` 文件名固定为 `recommend.proto`，生成目录结构保持与 `backend` 一致。
- 后续若在当前模块新增 RPC 契约，方法名、请求名、返回名必须同步遵循 `shop/backend/AGENTS.md` 的 `Proto接口契约命名规则`。
- 修改 `proto` 后必须执行 `make proto`，禁止手写生成产物。

## 文档约定
- 当前模块处于设计优先阶段。新增目录、协议、工具方法或缓存结构时，必须同步更新：
  - `README.md`
  - `docs/architecture.md`
  - `docs/pipeline.md`
  - `docs/integration.md`
  - `docs/capability-map.md`

## 校验约定
- 修改 `proto` 后，优先执行 `make proto`。
- 修改 Go 文件后，至少执行 `make test`。
- 若当前改动仅涉及文档与模块元信息，至少执行 `make help` 或其他可覆盖本模块的检查命令，并在结论中说明原因。
