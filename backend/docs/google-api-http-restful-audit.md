# google.api.http RESTful 审计清单

- 审计范围：`backend/api/protos/**/*.proto` 中全部 `google.api.http` 映射
- 接口总数：`206`
- Path：`✅ 158` / `⚠️ 32` / `❓ 16`
- 方法名：`✅ 179` / `⚠️ 16` / `❓ 11`

## 待确认口径

1. 认证接口是否统一按“会话 / 令牌 / 资料 / 验证码”资源模型重命名。
2. `/list` 且存在分页兄弟接口时，是否统一改成更具体的资源名，例如 `/option`、`/detail`、`/brief`。
3. 支付回调、订单确认单这类已对接外部平台或前端流程的接口，是否接受路径重构而不只做大小写修正。
4. RPC 方法名是否只修正明显缩写/歧义，还是要求与最终资源命名完全同步。

## `api/protos/admin/auth.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `GetUserInfo` | `GET /api/admin/auth/userInfo` | ⚠️ | `GET /api/admin/auth/account` | ⚠️ | `GetAccountInfo` | `userInfo` 为驼峰片段，且更适合抽象为当前账号资源。；如果 path 改为 `account`，方法名同步成账号语义更清晰。 |
| `GetUserMenu` | `GET /api/admin/auth/menu` | ✅ | - | ✅ | - | - |
| `GetUserButton` | `GET /api/admin/auth/button` | ✅ | - | ✅ | - | - |
| `GetUserProfile` | `GET /api/admin/auth/userProfile` | ⚠️ | `GET /api/admin/auth/profile` | ✅ | - | `userProfile` 为驼峰片段，建议统一为 `profile` 资源。 |
| `UpdateUserProfile` | `PUT /api/admin/auth/update/userProfile` | ⚠️ | `PUT /api/admin/auth/profile` | ✅ | - | `update` 不应出现在 path，更新语义由 `PUT` 表达。 |
| `SendUpdatePhoneCode` | `POST /api/admin/auth/send/update/phone` | ❓ | `POST /api/admin/auth/phone/code` | ⚠️ | `SendPhoneCode` | 当前是多层动作路径；建议抽象为手机号验证码资源，但是否拆成 `/phone/code` 需你确认。；去掉冗余的 `Update`，保留发送验证码的业务语义。 |
| `UpdateUserPhone` | `PUT /api/admin/auth/update/phone` | ⚠️ | `PUT /api/admin/auth/phone` | ✅ | - | `update` 不应出现在 path。 |
| `UpdateUserPwd` | `PUT /api/admin/auth/update/pwd` | ⚠️ | `PUT /api/admin/auth/password` | ⚠️ | `UpdateUserPassword` | `pwd` 命名不直观，建议统一为 `password` 资源。；建议展开 `Pwd` 缩写。 |

## `api/protos/admin/base_api.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `ListBaseApi` | `GET /api/admin/base/api/list` | ⚠️ | `GET /api/admin/base/api` | ✅ | - | 列表查询不需要 `/list`，`GET` 集合资源即可。 |

## `api/protos/admin/base_config.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `RefreshBaseConfig` | `POST /api/admin/base/config/refresh` | ❓ | `PUT /api/admin/base/config/cache` | ⚠️ | `RefreshBaseConfigCache` | `refresh` 是动作语义，建议改成缓存资源，但 `cache` 还是 `reload` 口径需要你确认。；若 path 改为缓存资源，方法名也应补齐 `Cache` 语义。 |
| `PageBaseConfig` | `GET /api/admin/base/config` | ✅ | - | ✅ | - | - |
| `GetBaseConfig` | `GET /api/admin/base/config/{value}` | ✅ | - | ✅ | - | - |
| `CreateBaseConfig` | `POST /api/admin/base/config` | ✅ | - | ✅ | - | - |
| `UpdateBaseConfig` | `PUT /api/admin/base/config/{id}` | ✅ | - | ✅ | - | - |
| `DeleteBaseConfig` | `DELETE /api/admin/base/config/{value}` | ✅ | - | ✅ | - | - |
| `SetBaseConfigStatus` | `PUT /api/admin/base/config/{id}/status` | ✅ | - | ✅ | - | - |

## `api/protos/admin/base_dept.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `TreeBaseDept` | `GET /api/admin/base/dept/tree` | ✅ | - | ✅ | - | - |
| `OptionBaseDept` | `GET /api/admin/base/dept/option` | ✅ | - | ✅ | - | - |
| `GetBaseDept` | `GET /api/admin/base/dept/{value}` | ✅ | - | ✅ | - | - |
| `CreateBaseDept` | `POST /api/admin/base/dept` | ✅ | - | ✅ | - | - |
| `UpdateBaseDept` | `PUT /api/admin/base/dept/{id}` | ✅ | - | ✅ | - | - |
| `DeleteBaseDept` | `DELETE /api/admin/base/dept/{value}` | ✅ | - | ✅ | - | - |
| `SetBaseDeptStatus` | `PUT /api/admin/base/dept/{id}/status` | ✅ | - | ✅ | - | - |

## `api/protos/admin/base_dict.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `ListBaseDict` | `GET /api/admin/base/dict/list` | ❓ | `GET /api/admin/base/dict/option` | ❓ | `OptionBaseDict` | 与分页接口并存时，`/list` 不稳定；如果该接口用于下拉数据，建议改成 `/option`，否则需改成更具体的资源名。；如果最终 path 采用 `/option`，方法名建议同步为选项语义。 |
| `PageBaseDict` | `GET /api/admin/base/dict` | ✅ | - | ✅ | - | - |
| `GetBaseDict` | `GET /api/admin/base/dict/{value}` | ✅ | - | ✅ | - | - |
| `CreateBaseDict` | `POST /api/admin/base/dict` | ✅ | - | ✅ | - | - |
| `UpdateBaseDict` | `PUT /api/admin/base/dict/{id}` | ✅ | - | ✅ | - | - |
| `DeleteBaseDict` | `DELETE /api/admin/base/dict/{value}` | ✅ | - | ✅ | - | - |
| `SetBaseDictStatus` | `PUT /api/admin/base/dict/{id}/status` | ✅ | - | ✅ | - | - |
| `PageBaseDictItem` | `GET /api/admin/base/dict-item` | ✅ | - | ✅ | - | - |
| `GetBaseDictItem` | `GET /api/admin/base/dict-item/{value}` | ✅ | - | ✅ | - | - |
| `CreateBaseDictItem` | `POST /api/admin/base/dict-item` | ✅ | - | ✅ | - | - |
| `UpdateBaseDictItem` | `PUT /api/admin/base/dict-item/{id}` | ✅ | - | ✅ | - | - |
| `DeleteBaseDictItem` | `DELETE /api/admin/base/dict-item/{value}` | ✅ | - | ✅ | - | - |
| `SetBaseDictItemStatus` | `PUT /api/admin/base/dict-item/{id}/status` | ✅ | - | ✅ | - | - |

## `api/protos/admin/base_job.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `PageBaseJob` | `GET /api/admin/base/job` | ✅ | - | ✅ | - | - |
| `GetBaseJob` | `GET /api/admin/base/job/{value}` | ✅ | - | ✅ | - | - |
| `CreateBaseJob` | `POST /api/admin/base/job` | ✅ | - | ✅ | - | - |
| `UpdateBaseJob` | `PUT /api/admin/base/job/{id}` | ✅ | - | ✅ | - | - |
| `DeleteBaseJob` | `DELETE /api/admin/base/job/{value}` | ✅ | - | ✅ | - | - |
| `SetBaseJobStatus` | `PUT /api/admin/base/job/{id}/status` | ✅ | - | ✅ | - | - |
| `StartBaseJob` | `PUT /api/admin/base/job/{id}/start` | ⚠️ | `PUT /api/admin/base/job/{id}/running` | ✅ | - | `start` 是动作路径，建议抽象为任务运行状态资源。 |
| `StopBaseJob` | `PUT /api/admin/base/job/{id}/stop` | ⚠️ | `DELETE /api/admin/base/job/{id}/running` | ✅ | - | 停止更适合删除运行态子资源；如果不改 method，则次选 `PUT /.../running` + body 标志位。 |
| `ExecBaseJob` | `PUT /api/admin/base/job/{id}/exec` | ❓ | `POST /api/admin/base/job/{id}/execution` | ⚠️ | `CreateBaseJobExecution` | 立即执行更像创建一次执行记录，建议抽象为 `execution` 子资源。；若 path 改为 `execution` 子资源，方法名建议表达“创建一次执行记录”。 |
| `PageBaseJobLog` | `GET /api/admin/base/job-log` | ✅ | - | ✅ | - | - |
| `GetBaseJobLog` | `GET /api/admin/base/job-log/{value}` | ✅ | - | ✅ | - | - |

## `api/protos/admin/base_log.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `PageBaseLog` | `GET /api/admin/base/log` | ✅ | - | ✅ | - | - |
| `GetBaseLog` | `GET /api/admin/base/log/{value}` | ✅ | - | ✅ | - | - |

## `api/protos/admin/base_menu.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `TreeBaseMenu` | `GET /api/admin/base/menu/tree` | ✅ | - | ✅ | - | - |
| `OptionBaseMenu` | `GET /api/admin/base/menu/option` | ✅ | - | ✅ | - | - |
| `GetBaseMenu` | `GET /api/admin/base/menu/{value}` | ✅ | - | ✅ | - | - |
| `CreateBaseMenu` | `POST /api/admin/base/menu` | ✅ | - | ✅ | - | - |
| `UpdateBaseMenu` | `PUT /api/admin/base/menu/{id}` | ✅ | - | ✅ | - | - |
| `DeleteBaseMenu` | `DELETE /api/admin/base/menu/{value}` | ✅ | - | ✅ | - | - |
| `SetBaseMenuStatus` | `PUT /api/admin/base/menu/{id}/status` | ✅ | - | ✅ | - | - |

## `api/protos/admin/base_role.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `OptionBaseRole` | `GET /api/admin/base/role/option` | ✅ | - | ✅ | - | - |
| `PageBaseRole` | `GET /api/admin/base/role` | ✅ | - | ✅ | - | - |
| `GetBaseRole` | `GET /api/admin/base/role/{value}` | ✅ | - | ✅ | - | - |
| `CreateBaseRole` | `POST /api/admin/base/role` | ✅ | - | ✅ | - | - |
| `UpdateBaseRole` | `PUT /api/admin/base/role/{id}` | ✅ | - | ✅ | - | - |
| `DeleteBaseRole` | `DELETE /api/admin/base/role/{value}` | ✅ | - | ✅ | - | - |
| `SetBaseRoleStatus` | `PUT /api/admin/base/role/{id}/status` | ✅ | - | ✅ | - | - |
| `SetBaseRoleMenus` | `PUT /api/admin/base/role/{id}/menus` | ⚠️ | `PUT /api/admin/base/role/{id}/menu` | ⚠️ | `SetBaseRoleMenu` | 子资源禁止复数片段，`menus` 应改为 `menu`。；与子资源 `menu` 保持单数一致。 |

## `api/protos/admin/base_user.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `OptionBaseUser` | `GET /api/admin/base/user/option` | ✅ | - | ✅ | - | - |
| `PageBaseUser` | `GET /api/admin/base/user` | ✅ | - | ✅ | - | - |
| `GetBaseUser` | `GET /api/admin/base/user/{value}` | ✅ | - | ✅ | - | - |
| `CreateBaseUser` | `POST /api/admin/base/user` | ✅ | - | ✅ | - | - |
| `UpdateBaseUser` | `PUT /api/admin/base/user/{id}` | ✅ | - | ✅ | - | - |
| `DeleteBaseUser` | `DELETE /api/admin/base/user/{value}` | ✅ | - | ✅ | - | - |
| `SetBaseUserStatus` | `PUT /api/admin/base/user/{id}/status` | ✅ | - | ✅ | - | - |
| `ResetBaseUserPwd` | `PUT /api/admin/base/user/{id}/pwd` | ⚠️ | `PUT /api/admin/base/user/{id}/password` | ⚠️ | `ResetBaseUserPassword` | `pwd` 建议统一为完整资源名 `password`。；建议展开 `Pwd` 缩写。 |

## `api/protos/admin/goods_analytics.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `GetGoodsAnalyticsSummary` | `GET /api/admin/analytics/goods/summary` | ✅ | - | ✅ | - | - |
| `GetGoodsAnalyticsTrend` | `GET /api/admin/analytics/goods/trend` | ✅ | - | ✅ | - | - |
| `GetGoodsAnalyticsPie` | `GET /api/admin/analytics/goods/pie` | ✅ | - | ✅ | - | - |

## `api/protos/admin/goods_category.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `TreeGoodsCategory` | `GET /api/admin/goods/category/tree` | ✅ | - | ✅ | - | - |
| `OptionGoodsCategory` | `GET /api/admin/goods/category/option` | ✅ | - | ✅ | - | - |
| `GetGoodsCategory` | `GET /api/admin/goods/category/{value}` | ✅ | - | ✅ | - | - |
| `CreateGoodsCategory` | `POST /api/admin/goods/category` | ✅ | - | ✅ | - | - |
| `UpdateGoodsCategory` | `PUT /api/admin/goods/category/{id}` | ✅ | - | ✅ | - | - |
| `DeleteGoodsCategory` | `DELETE /api/admin/goods/category/{value}` | ✅ | - | ✅ | - | - |
| `SetGoodsCategoryStatus` | `PUT /api/admin/goods/category/{id}/status` | ✅ | - | ✅ | - | - |

## `api/protos/admin/goods_info.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `ListGoodsInfo` | `GET /api/admin/goods/info/list` | ❓ | `GET /api/admin/goods/info/brief` | ✅ | - | 与分页接口并存时不建议再用 `/list`；该接口返回轻量列表，建议改成 `/brief` 或你指定的更业务化资源名。 |
| `PageGoodsInfo` | `GET /api/admin/goods/info` | ✅ | - | ✅ | - | - |
| `GetGoodsInfo` | `GET /api/admin/goods/info/{value}` | ✅ | - | ✅ | - | - |
| `CreateGoodsInfo` | `POST /api/admin/goods/info` | ✅ | - | ✅ | - | - |
| `UpdateGoodsInfo` | `PUT /api/admin/goods/info/{id}` | ✅ | - | ✅ | - | - |
| `DeleteGoodsInfo` | `DELETE /api/admin/goods/info/{value}` | ✅ | - | ✅ | - | - |
| `SetGoodsInfoStatus` | `PUT /api/admin/goods/info/{id}/status` | ✅ | - | ✅ | - | - |

## `api/protos/admin/goods_prop.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `PageGoodsProp` | `GET /api/admin/goods/prop` | ✅ | - | ✅ | - | - |
| `GetGoodsProp` | `GET /api/admin/goods/prop/{value}` | ✅ | - | ✅ | - | - |
| `CreateGoodsProp` | `POST /api/admin/goods/prop` | ✅ | - | ✅ | - | - |
| `UpdateGoodsProp` | `PUT /api/admin/goods/prop/{id}` | ✅ | - | ✅ | - | - |
| `DeleteGoodsProp` | `DELETE /api/admin/goods/prop/{value}` | ✅ | - | ✅ | - | - |

## `api/protos/admin/goods_sku.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `PageGoodsSku` | `GET /api/admin/goods/sku` | ✅ | - | ✅ | - | - |
| `GetGoodsSku` | `GET /api/admin/goods/sku/{value}` | ✅ | - | ✅ | - | - |
| `UpdateGoodsSku` | `PUT /api/admin/goods/sku/{id}` | ✅ | - | ✅ | - | - |

## `api/protos/admin/goods_spec.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `ListGoodsSpec` | `GET /api/admin/goods/spec/list` | ⚠️ | `GET /api/admin/goods/spec` | ✅ | - | 无分页兄弟接口时，集合查询直接使用资源本身即可。 |

## `api/protos/admin/order_analytics.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `GetOrderAnalyticsSummary` | `GET /api/admin/analytics/order/summary` | ✅ | - | ✅ | - | - |
| `GetOrderAnalyticsTrend` | `GET /api/admin/analytics/order/trend` | ✅ | - | ✅ | - | - |
| `GetOrderAnalyticsPie` | `GET /api/admin/analytics/order/pie` | ✅ | - | ✅ | - | - |

## `api/protos/admin/order_info.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `PageOrderInfo` | `GET /api/admin/order/info` | ✅ | - | ✅ | - | - |
| `GetOrderInfo` | `GET /api/admin/order/info/{value}` | ✅ | - | ✅ | - | - |
| `GetOrderInfoRefund` | `GET /api/admin/order/info/{value}/refund` | ✅ | - | ✅ | - | - |
| `RefundOrderInfo` | `PUT /api/admin/order/info/{orderId}/refund` | ⚠️ | `PUT /api/admin/order/info/{order_id}/refund` | ✅ | - | 路径参数应使用小写稳定命名，避免 `{orderId}` 驼峰。 |
| `GetOrderInfoShipped` | `GET /api/admin/order/info/{value}/shipped` | ⚠️ | `GET /api/admin/order/info/{value}/shipment` | ⚠️ | `GetOrderInfoShipment` | `shipped` 是状态描述，不如 `shipment` 作为子资源稳定。；与 `shipment` 子资源保持一致。 |
| `ShippedOrderInfo` | `PUT /api/admin/order/info/{orderId}/shipped` | ⚠️ | `PUT /api/admin/order/info/{order_id}/shipment` | ⚠️ | `ShipOrderInfo` | 与查询接口统一成 `shipment` 子资源，并修正路径参数命名。；当前命名不自然，改成发货动作更清晰。 |

## `api/protos/admin/order_report.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `OrderMonthReportSummary` | `GET /api/admin/report/order/month/summary` | ✅ | - | ✅ | - | - |
| `OrderMonthReportList` | `GET /api/admin/report/order/month/list` | ⚠️ | `GET /api/admin/report/order/month/detail` | ✅ | - | 与 `/summary` 并列时，`/detail` 比 `/list` 更符合报表语义。 |
| `OrderDayReportSummary` | `GET /api/admin/report/order/day/summary` | ✅ | - | ✅ | - | - |
| `OrderDayReportList` | `GET /api/admin/report/order/day/list` | ⚠️ | `GET /api/admin/report/order/day/detail` | ✅ | - | 与 `/summary` 并列时，`/detail` 比 `/list` 更符合报表语义。 |

## `api/protos/admin/pay_bill.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `PagePayBill` | `GET /api/admin/pay/bill` | ✅ | - | ✅ | - | - |

## `api/protos/admin/shop_banner.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `PageShopBanner` | `GET /api/admin/shop/banner` | ✅ | - | ✅ | - | - |
| `GetShopBanner` | `GET /api/admin/shop/banner/{value}` | ✅ | - | ✅ | - | - |
| `CreateShopBanner` | `POST /api/admin/shop/banner` | ✅ | - | ✅ | - | - |
| `UpdateShopBanner` | `PUT /api/admin/shop/banner/{id}` | ✅ | - | ✅ | - | - |
| `DeleteShopBanner` | `DELETE /api/admin/shop/banner/{value}` | ✅ | - | ✅ | - | - |
| `SetShopBannerStatus` | `PUT /api/admin/shop/banner/{id}/status` | ✅ | - | ✅ | - | - |

## `api/protos/admin/shop_hot.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `PageShopHot` | `GET /api/admin/shop/hot` | ✅ | - | ✅ | - | - |
| `GetShopHot` | `GET /api/admin/shop/hot/{value}` | ✅ | - | ✅ | - | - |
| `CreateShopHot` | `POST /api/admin/shop/hot` | ✅ | - | ✅ | - | - |
| `UpdateShopHot` | `PUT /api/admin/shop/hot/{id}` | ✅ | - | ✅ | - | - |
| `DeleteShopHot` | `DELETE /api/admin/shop/hot/{value}` | ✅ | - | ✅ | - | - |
| `SetShopHotStatus` | `PUT /api/admin/shop/hot/{id}/status` | ✅ | - | ✅ | - | - |
| `PageShopHotItem` | `GET /api/admin/shop/hot-item` | ✅ | - | ✅ | - | - |
| `GetShopHotItem` | `GET /api/admin/shop/hot-item/{value}` | ✅ | - | ✅ | - | - |
| `CreateShopHotItem` | `POST /api/admin/shop/hot-item` | ✅ | - | ✅ | - | - |
| `UpdateShopHotItem` | `PUT /api/admin/shop/hot-item/{id}` | ✅ | - | ✅ | - | - |
| `DeleteShopHotItem` | `DELETE /api/admin/shop/hot-item/{value}` | ✅ | - | ✅ | - | - |
| `SetShopHotItemStatus` | `PUT /api/admin/shop/hot-item/{id}/status` | ✅ | - | ✅ | - | - |

## `api/protos/admin/shop_service.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `PageShopService` | `GET /api/admin/shop/service` | ✅ | - | ✅ | - | - |
| `GetShopService` | `GET /api/admin/shop/service/{value}` | ✅ | - | ✅ | - | - |
| `CreateShopService` | `POST /api/admin/shop/service` | ✅ | - | ✅ | - | - |
| `UpdateShopService` | `PUT /api/admin/shop/service/{id}` | ✅ | - | ✅ | - | - |
| `DeleteShopService` | `DELETE /api/admin/shop/service/{value}` | ✅ | - | ✅ | - | - |
| `SetShopServiceStatus` | `PUT /api/admin/shop/service/{id}/status` | ✅ | - | ✅ | - | - |

## `api/protos/admin/user_analytics.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `GetUserAnalyticsSummary` | `GET /api/admin/analytics/user/summary` | ✅ | - | ✅ | - | - |
| `GetUserAnalyticsTrend` | `GET /api/admin/analytics/user/trend` | ✅ | - | ✅ | - | - |
| `GetUserAnalyticsRank` | `GET /api/admin/analytics/user/rank` | ✅ | - | ✅ | - | - |

## `api/protos/admin/user_store.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `PageUserStore` | `GET /api/admin/user/store` | ✅ | - | ✅ | - | - |
| `GetUserStore` | `GET /api/admin/user/store/{value}` | ✅ | - | ✅ | - | - |
| `AuditUserStore` | `PUT /api/admin/user/store/{id}/audit` | ✅ | - | ✅ | - | - |

## `api/protos/admin/workspace.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `GetWorkspaceMetrics` | `GET /api/admin/workspace/metric` | ✅ | - | ✅ | - | - |
| `GetWorkspaceTodoList` | `GET /api/admin/workspace/todo` | ✅ | - | ✅ | - | - |
| `GetWorkspaceRiskList` | `GET /api/admin/workspace/risk` | ✅ | - | ✅ | - | - |

## `api/protos/app/auth.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `WxLogin` | `POST /api/app/auth/login/wx` | ❓ | `POST /api/app/auth/session/wechat` | ❓ | `CreateWechatSession` | 建议把登录抽象为会话资源；是否使用 `session/wechat` 还是沿用 `auth/wechat` 需要你确认。；如果 path 采用会话资源模型，方法名建议同步。 |
| `GetUserInfo` | `GET /api/app/auth/userInfo` | ⚠️ | `GET /api/app/auth/profile` | ⚠️ | `GetUserProfile` | `userInfo` 为驼峰片段，建议统一为当前用户资料资源。；与 `profile` 资源保持一致。 |
| `UpdateUserInfo` | `PUT /api/app/auth/userInfo` | ⚠️ | `PUT /api/app/auth/profile` | ⚠️ | `UpdateUserProfile` | 与查询接口统一到 `profile` 资源。；与 `profile` 资源保持一致。 |
| `PhoneAuth` | `PUT /api/app/auth/userInfo/phone` | ⚠️ | `PUT /api/app/auth/phone` | ⚠️ | `BindUserPhone` | `userInfo/phone` 层级冗余，直接更新手机号资源即可。；当前语义更接近手机号绑定。 |

## `api/protos/app/base_area.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `TreeBaseArea` | `GET /api/app/base/area/tree` | ✅ | - | ✅ | - | - |

## `api/protos/app/base_dict.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `ListBaseDict` | `GET /api/app/base/dict/list` | ⚠️ | `GET /api/app/base/dict/{value}` | ⚠️ | `GetBaseDict` | 当前请求入参是单个字典编码，改成按资源标识访问更自然。；如果 path 改为 `/dict/{value}`，方法名建议改为按编码获取。 |

## `api/protos/app/goods_category.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `ListGoodsCategory` | `GET /api/app/goods/category` | ✅ | - | ✅ | - | - |

## `api/protos/app/goods_info.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `PageGoodsInfo` | `GET /api/app/goods/info` | ✅ | - | ✅ | - | - |
| `GetGoodsInfo` | `GET /api/app/goods/info/{value}` | ✅ | - | ✅ | - | - |

## `api/protos/app/order_info.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `OrderInfoPre` | `POST /api/app/order/info/pre` | ❓ | `POST /api/app/order/confirm` | ❓ | `CreateOrderConfirm` | 当前 `pre` 不是资源名；更合理的是抽象成确认单/结算单资源，但这会牵涉 3 个相关接口一起调整。；`Pre` 语义不明确，若 path 改为确认单资源，方法名建议同步。 |
| `OrderInfoBuy` | `POST /api/app/order/info/buy` | ❓ | `POST /api/app/order/confirm/buy-now` | ❓ | `CreateBuyNowOrderConfirm` | 如果不合并接口，至少要把 `buy` 改成更明确的子资源；是否接受 `buy-now` 需你确认。；建议直接表达“立即购买确认单”。 |
| `OrderInfoRepurchase` | `POST /api/app/order/info/repurchase` | ❓ | `POST /api/app/order/confirm/repurchase` | ❓ | `CreateRepurchaseOrderConfirm` | 建议与确认单语义统一，但是否保留单独端点需要你确认。；建议直接表达“再次购买确认单”。 |
| `CountOrderInfo` | `GET /api/app/order/info/count` | ✅ | - | ✅ | - | - |
| `PageOrderInfo` | `GET /api/app/order/info` | ✅ | - | ✅ | - | - |
| `GetOrderInfoIdByOrderNo` | `GET /api/app/order/info/{value}/orderNo` | ⚠️ | `GET /api/app/order/info/no/{value}` | ✅ | - | `orderNo` 为驼峰片段，改成按订单号子资源访问更稳定。 |
| `GetOrderInfoById` | `GET /api/app/order/info/{value}` | ✅ | - | ✅ | - | - |
| `CreateOrderInfo` | `POST /api/app/order/info` | ✅ | - | ✅ | - | - |
| `DeleteOrderInfo` | `DELETE /api/app/order/info/{value}` | ✅ | - | ✅ | - | - |
| `CancelOrderInfo` | `PUT /api/app/order/info/{orderId}/cancel` | ⚠️ | `PUT /api/app/order/info/{order_id}/cancellation` | ✅ | - | `cancel` 是动作，建议抽象为取消资源。 |
| `RefundOrderInfo` | `PUT /api/app/order/info/{orderId}/refund` | ⚠️ | `PUT /api/app/order/info/{order_id}/refund` | ✅ | - | 路径参数应使用小写稳定命名。 |
| `ReceiveOrderInfo` | `PUT /api/app/order/info/{orderId}/receive` | ⚠️ | `PUT /api/app/order/info/{order_id}/receipt` | ✅ | - | `receive` 是动作，建议改成收货确认资源。 |

## `api/protos/app/pay.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `JsapiPay` | `POST /api/app/pay/{orderId}/jsapi` | ❓ | `POST /api/app/order/info/{order_id}/payment/jsapi` | ❓ | `CreateOrderJsapiPayment` | 更合理的是把支付挂到订单支付子资源下，但会影响前端与支付联动。；若 path 改挂到订单支付资源，方法名建议同步。 |
| `H5Pay` | `POST /api/app/pay/{orderId}/h5` | ❓ | `POST /api/app/order/info/{order_id}/payment/h5` | ❓ | `CreateOrderH5Payment` | 建议与 `JsapiPay` 同步迁移到订单支付子资源。；若 path 改挂到订单支付资源，方法名建议同步。 |
| `PayNotify` | `POST /api/app/pay/notify` | ❓ | `POST /api/app/pay/notification` | ❓ | `HandlePaymentNotification` | 建议改成通知资源，但该路径通常配置在第三方平台，需要你确认是否允许改回调地址。；如果 path 改成通知资源，方法名建议突出“处理通知”。 |

## `api/protos/app/recommend.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `RecommendAnonymousActor` | `GET /api/app/recommend/actor/anonymous` | ✅ | - | ✅ | - | - |
| `BindRecommendAnonymousActor` | `POST /api/app/recommend/actor/bind` | ⚠️ | `POST /api/app/recommend/actor/binding` | ⚠️ | `CreateRecommendActorBinding` | `bind` 是动作，建议抽象为绑定关系资源。；若 path 改为 `binding`，方法名建议同步为绑定关系创建。 |
| `RecommendGoods` | `GET /api/app/recommend/goods` | ✅ | - | ✅ | - | - |
| `RecommendExposureReport` | `POST /api/app/recommend/event/exposure` | ✅ | - | ✅ | - | - |
| `RecommendGoodsActionReport` | `POST /api/app/recommend/event/goods` | ✅ | - | ✅ | - | - |

## `api/protos/app/shop_banner.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `ListShopBanner` | `GET /api/app/shop/banner` | ✅ | - | ✅ | - | - |

## `api/protos/app/shop_hot.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `ListShopHot` | `GET /api/app/shop/hot` | ✅ | - | ✅ | - | - |
| `ListShopHotItem` | `GET /api/app/shop/hot/item` | ✅ | - | ✅ | - | - |
| `PageShopHotGoods` | `GET /api/app/shop/hot/goods` | ✅ | - | ✅ | - | - |

## `api/protos/app/shop_service.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `ListShopService` | `GET /api/app/shop/service` | ✅ | - | ✅ | - | - |

## `api/protos/app/user_address.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `ListUserAddress` | `GET /api/app/user/address` | ✅ | - | ✅ | - | - |
| `GetUserAddress` | `GET /api/app/user/address/{value}` | ✅ | - | ✅ | - | - |
| `CreateUserAddress` | `POST /api/app/user/address` | ✅ | - | ✅ | - | - |
| `UpdateUserAddress` | `PUT /api/app/user/address/{id}` | ✅ | - | ✅ | - | - |
| `DeleteUserAddress` | `DELETE /api/app/user/address/{value}` | ✅ | - | ✅ | - | - |

## `api/protos/app/user_cart.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `CountUserCart` | `GET /api/app/user/cart/count` | ✅ | - | ✅ | - | - |
| `ListUserCart` | `GET /api/app/user/cart/list` | ⚠️ | `GET /api/app/user/cart` | ✅ | - | 集合查询不需要 `/list`。 |
| `CreateUserCart` | `POST /api/app/user/cart` | ✅ | - | ✅ | - | - |
| `UpdateUserCart` | `PUT /api/app/user/cart` | ⚠️ | `PUT /api/app/user/cart/{id}` | ✅ | - | 标准更新更适合把 `id` 放入 path；这条属于附加优化，不会改变资源语义。 |
| `DeleteUserCart` | `DELETE /api/app/user/cart/{value}` | ✅ | - | ✅ | - | - |
| `SetUserCartStatus` | `PUT /api/app/user/cart/{id}/status` | ✅ | - | ✅ | - | - |
| `SelectedUserCart` | `PUT /api/app/user/cart/selected` | ⚠️ | `PUT /api/app/user/cart/selection` | ⚠️ | `SetUserCartSelection` | `selected` 不是稳定资源名，建议改成 `selection`。；与 `selection` 子资源保持一致。 |

## `api/protos/app/user_collect.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `PageUserCollect` | `GET /api/app/user/collect` | ✅ | - | ✅ | - | - |
| `GetIsCollect` | `GET /api/app/user/collect/status` | ✅ | - | ✅ | - | - |
| `CreateUserCollect` | `POST /api/app/user/collect` | ✅ | - | ✅ | - | - |
| `DeleteUserCollect` | `DELETE /api/app/user/collect/{value}` | ✅ | - | ✅ | - | - |

## `api/protos/app/user_store.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `GetUserStore` | `GET /api/app/user/store` | ✅ | - | ✅ | - | - |
| `CreateUserStore` | `POST /api/app/user/store` | ✅ | - | ✅ | - | - |
| `UpdateUserStore` | `PUT /api/app/user/store/{id}` | ✅ | - | ✅ | - | - |

## `api/protos/base/config.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `GetConfig` | `GET /api/config` | ⚠️ | `GET /api/base/config` | ✅ | - | 缺少 `base` terminal，建议补齐统一前缀结构。 |

## `api/protos/base/file.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `MultiUploadFile` | `POST /api/file/multi` | ⚠️ | `POST /api/base/file/batch` | ⚠️ | `BatchUploadFile` | 缺少 `base` terminal，且批量上传建议使用集合资源子路径。；若 path 改为 `/batch`，方法名建议同步。 |
| `UploadFile` | `POST /api/file` | ⚠️ | `POST /api/base/file` | ✅ | - | 缺少 `base` terminal。 |
| `DownloadFile` | `GET /api/file` | ⚠️ | `GET /api/base/file` | ✅ | - | 缺少 `base` terminal；保留同资源的 GET/POST 是合理的。 |

## `api/protos/base/login.proto`

| RPC | 当前 HTTP | Path 结论 | Path 建议 | 方法名结论 | 方法名建议 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `Captcha` | `GET /api/login/captcha` | ❓ | `GET /api/base/auth/captcha` | ✅ | - | 建议补齐 `base` terminal，并把登录相关能力收口到 `auth` 模块。 |
| `Logout` | `DELETE /api/login/logout` | ❓ | `DELETE /api/base/auth/session` | ❓ | `DeleteSession` | 更 RESTful 的表达是删除当前会话资源。；如果采用会话资源模型，方法名建议同步。 |
| `RefreshToken` | `POST /api/login/refreshToken` | ❓ | `POST /api/base/auth/token` | ❓ | `RefreshAuthToken` | 建议把刷新令牌改成令牌资源重发，但是否接受这套 auth 资源模型需要你确认。；如果采用令牌资源模型，方法名建议同步。 |
| `Login` | `POST /api/login` | ❓ | `POST /api/base/auth/session` | ❓ | `CreateSession` | 更 RESTful 的表达是创建当前会话资源。；如果采用会话资源模型，方法名建议同步。 |
