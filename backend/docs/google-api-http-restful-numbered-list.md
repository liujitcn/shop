# google.api.http 不合规项编号清单

- 仅包含当前不符合 `backend/AGENTS.md` RESTful 规则的接口
- `Path 结论`：`⚠️` 表示可直接按建议修改，`❓` 表示需要你先拍板
- `方法名结论`：`⚠️` 表示建议同步改名，`❓` 表示需结合你的命名口径确认
- `已确认` 表示用户已拍板，后续按该方案落地
- `保留` 表示用户确认保持当前实现，本轮不改
- 你本轮给出的“13 => /api/admin/goods/info/option + OptionGoodsInfo”与当前第 `13` 项内容不一致，已按接口内容归并到第 `15` 项

## 你可以这样回复

```markdown
修改：
- 1, 2, 3, 7

保留：
- 22, 34, 35, 36

不懂：
- 8, 9, 27

按我的方案修改：
- 4 => path 改成 /api/admin/auth/phone/sms-code，方法名改成 SendUpdatePhoneSmsCode
- 15 => path 改成 /api/admin/goods/info/option
```

## 编号清单

| 编号 | 文件 | RPC | 当前 HTTP | Path 结论 | 建议 Path | 方法名结论 | 建议方法名 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | `api/protos/admin/auth.proto` | `GetUserInfo` | `GET /api/admin/auth/userInfo` | `已确认` | `GET /api/admin/auth/user` | `保留` | - |
| 2 | `api/protos/admin/auth.proto` | `GetUserProfile` | `GET /api/admin/auth/userProfile` | `⚠️` | `GET /api/admin/auth/profile` | `✅` | - |
| 3 | `api/protos/admin/auth.proto` | `UpdateUserProfile` | `PUT /api/admin/auth/update/userProfile` | `⚠️` | `PUT /api/admin/auth/profile` | `✅` | - |
| 4 | `api/protos/admin/auth.proto` | `SendUpdatePhoneCode` | `POST /api/admin/auth/send/update/phone` | `❓` | `POST /api/admin/auth/phone/code` | `⚠️` | `SendPhoneCode` |
| 5 | `api/protos/admin/auth.proto` | `UpdateUserPhone` | `PUT /api/admin/auth/update/phone` | `⚠️` | `PUT /api/admin/auth/phone` | `✅` | - |
| 6 | `api/protos/admin/auth.proto` | `UpdateUserPwd` | `PUT /api/admin/auth/update/pwd` | `⚠️` | `PUT /api/admin/auth/password` | `⚠️` | `UpdateUserPassword` |
| 7 | `api/protos/admin/base_api.proto` | `ListBaseApi` | `GET /api/admin/base/api/list` | `⚠️` | `GET /api/admin/base/api` | `✅` | - |
| 8 | `api/protos/admin/base_config.proto` | `RefreshBaseConfig` | `POST /api/admin/base/config/refresh` | `❓` | `PUT /api/admin/base/config/cache` | `⚠️` | `RefreshBaseConfigCache` |
| 9 | `api/protos/admin/base_dict.proto` | `ListBaseDict` | `GET /api/admin/base/dict/list` | `❓` | `GET /api/admin/base/dict/option` | `❓` | `OptionBaseDict` |
| 10 | `api/protos/admin/base_job.proto` | `StartBaseJob` | `PUT /api/admin/base/job/{id}/start` | `⚠️` | `PUT /api/admin/base/job/{id}/running` | `✅` | - |
| 11 | `api/protos/admin/base_job.proto` | `StopBaseJob` | `PUT /api/admin/base/job/{id}/stop` | `⚠️` | `DELETE /api/admin/base/job/{id}/running` | `✅` | - |
| 12 | `api/protos/admin/base_job.proto` | `ExecBaseJob` | `PUT /api/admin/base/job/{id}/exec` | `❓` | `POST /api/admin/base/job/{id}/execution` | `已确认` | `ExecuteBaseJob` |
| 13 | `api/protos/admin/base_role.proto` | `SetBaseRoleMenus` | `PUT /api/admin/base/role/{id}/menus` | `⚠️` | `PUT /api/admin/base/role/{id}/menu` | `⚠️` | `SetBaseRoleMenu` |
| 14 | `api/protos/admin/base_user.proto` | `ResetBaseUserPwd` | `PUT /api/admin/base/user/{id}/pwd` | `⚠️` | `PUT /api/admin/base/user/{id}/password` | `⚠️` | `ResetBaseUserPassword` |
| 15 | `api/protos/admin/goods_info.proto` | `ListGoodsInfo` | `GET /api/admin/goods/info/list` | `已确认` | `GET /api/admin/goods/info/option` | `已确认` | `OptionGoodsInfo` |
| 16 | `api/protos/admin/goods_spec.proto` | `ListGoodsSpec` | `GET /api/admin/goods/spec/list` | `⚠️` | `GET /api/admin/goods/spec` | `✅` | - |
| 17 | `api/protos/admin/order_info.proto` | `RefundOrderInfo` | `PUT /api/admin/order/info/{orderId}/refund` | `⚠️` | `PUT /api/admin/order/info/{order_id}/refund` | `✅` | - |
| 18 | `api/protos/admin/order_info.proto` | `GetOrderInfoShipped` | `GET /api/admin/order/info/{value}/shipped` | `⚠️` | `GET /api/admin/order/info/{value}/shipment` | `⚠️` | `GetOrderInfoShipment` |
| 19 | `api/protos/admin/order_info.proto` | `ShippedOrderInfo` | `PUT /api/admin/order/info/{orderId}/shipped` | `⚠️` | `PUT /api/admin/order/info/{order_id}/shipment` | `⚠️` | `ShipOrderInfo` |
| 20 | `api/protos/admin/order_report.proto` | `OrderMonthReportList` | `GET /api/admin/report/order/month/list` | `⚠️` | `GET /api/admin/report/order/month/detail` | `✅` | - |
| 21 | `api/protos/admin/order_report.proto` | `OrderDayReportList` | `GET /api/admin/report/order/day/list` | `⚠️` | `GET /api/admin/report/order/day/detail` | `✅` | - |
| 22 | `api/protos/app/auth.proto` | `WxLogin` | `POST /api/app/auth/login/wx` | `已确认` | `POST /api/app/auth/wechat` | `已确认` | `WechatLogin` |
| 23 | `api/protos/app/auth.proto` | `GetUserInfo` | `GET /api/app/auth/userInfo` | `⚠️` | `GET /api/app/auth/profile` | `⚠️` | `GetUserProfile` |
| 24 | `api/protos/app/auth.proto` | `UpdateUserInfo` | `PUT /api/app/auth/userInfo` | `⚠️` | `PUT /api/app/auth/profile` | `⚠️` | `UpdateUserProfile` |
| 25 | `api/protos/app/auth.proto` | `PhoneAuth` | `PUT /api/app/auth/userInfo/phone` | `⚠️` | `PUT /api/app/auth/phone` | `⚠️` | `BindUserPhone` |
| 26 | `api/protos/app/base_dict.proto` | `ListBaseDict` | `GET /api/app/base/dict/list` | `⚠️` | `GET /api/app/base/dict/{value}` | `⚠️` | `GetBaseDict` |
| 27 | `api/protos/app/order_info.proto` | `OrderInfoPre` | `POST /api/app/order/info/pre` | `❓` | `POST /api/app/order/confirm` | `已确认` | `ConfirmOrderInfo` |
| 28 | `api/protos/app/order_info.proto` | `OrderInfoBuy` | `POST /api/app/order/info/buy` | `❓` | `POST /api/app/order/confirm/buy-now` | `已确认` | `BuyNowOrderInfo` |
| 29 | `api/protos/app/order_info.proto` | `OrderInfoRepurchase` | `POST /api/app/order/info/repurchase` | `❓` | `POST /api/app/order/confirm/repurchase` | `已确认` | `RepurchaseOrderInfo` |
| 30 | `api/protos/app/order_info.proto` | `GetOrderInfoIdByOrderNo` | `GET /api/app/order/info/{value}/orderNo` | `⚠️` | `GET /api/app/order/info/no/{value}` | `✅` | - |
| 31 | `api/protos/app/order_info.proto` | `CancelOrderInfo` | `PUT /api/app/order/info/{orderId}/cancel` | `⚠️` | `PUT /api/app/order/info/{order_id}/cancellation` | `✅` | - |
| 32 | `api/protos/app/order_info.proto` | `RefundOrderInfo` | `PUT /api/app/order/info/{orderId}/refund` | `⚠️` | `PUT /api/app/order/info/{order_id}/refund` | `✅` | - |
| 33 | `api/protos/app/order_info.proto` | `ReceiveOrderInfo` | `PUT /api/app/order/info/{orderId}/receive` | `⚠️` | `PUT /api/app/order/info/{order_id}/receipt` | `✅` | - |
| 34 | `api/protos/app/pay.proto` | `JsapiPay` | `POST /api/app/pay/{orderId}/jsapi` | `❓` | `POST /api/app/order/info/{order_id}/payment/jsapi` | `保留` | - |
| 35 | `api/protos/app/pay.proto` | `H5Pay` | `POST /api/app/pay/{orderId}/h5` | `❓` | `POST /api/app/order/info/{order_id}/payment/h5` | `保留` | - |
| 36 | `api/protos/app/pay.proto` | `PayNotify` | `POST /api/app/pay/notify` | `❓` | `POST /api/app/pay/notification` | `保留` | - |
| 37 | `api/protos/app/recommend.proto` | `BindRecommendAnonymousActor` | `POST /api/app/recommend/actor/bind` | `⚠️` | `POST /api/app/recommend/actor/binding` | `保留` | - |
| 38 | `api/protos/app/user_cart.proto` | `ListUserCart` | `GET /api/app/user/cart/list` | `⚠️` | `GET /api/app/user/cart` | `✅` | - |
| 39 | `api/protos/app/user_cart.proto` | `UpdateUserCart` | `PUT /api/app/user/cart` | `⚠️` | `PUT /api/app/user/cart/{id}` | `✅` | - |
| 40 | `api/protos/app/user_cart.proto` | `SelectedUserCart` | `PUT /api/app/user/cart/selected` | `⚠️` | `PUT /api/app/user/cart/selection` | `⚠️` | `SetUserCartSelection` |
| 41 | `api/protos/base/config.proto` | `GetConfig` | `GET /api/config` | `保留` | - | `保留` | - |
| 42 | `api/protos/base/file.proto` | `MultiUploadFile` | `POST /api/file/multi` | `保留` | - | `保留` | - |
| 43 | `api/protos/base/file.proto` | `UploadFile` | `POST /api/file` | `保留` | - | `保留` | - |
| 44 | `api/protos/base/file.proto` | `DownloadFile` | `GET /api/file` | `保留` | - | `保留` | - |
| 45 | `api/protos/base/login.proto` | `Captcha` | `GET /api/login/captcha` | `保留` | - | `保留` | - |
| 46 | `api/protos/base/login.proto` | `Logout` | `DELETE /api/login/logout` | `已确认` | `DELETE /api/auth` | `保留` | - |
| 47 | `api/protos/base/login.proto` | `RefreshToken` | `POST /api/login/refreshToken` | `已确认` | `POST /api/auth/token` | `保留` | - |
| 48 | `api/protos/base/login.proto` | `Login` | `POST /api/login` | `已确认` | `POST /api/auth` | `保留` | - |

## Proto message 命名整理（按本轮最终口径）

- 推导口径：
  - 本轮既处理“`RPC 方法名` 会改”的接口，也处理“方法名不改、但入参或返回值 message 不符合规则”的接口。
  - 当前已经使用 `google.protobuf.*` 的入参或返回值，一律保持当前类型，不因为规则或方法名变化继续改名。
  - 当前不是 `google.protobuf.*` 的入参或返回值，仍按 `backend/AGENTS.md` 的规则收口；如果规则要求改成 `google.protobuf.*`，本轮也纳入调整。
  - 下表只整理“入参/返回值 message 名字”，不展开字段级别调整。

### 方法名会改，且需要同步调整 message

| 编号 | 最终 RPC | 当前入参 | 建议入参 | 当前返回 | 建议返回 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| 4 | `SendPhoneCode` | `SendUpdatePhoneCodeForm` | `SendPhoneCodeRequest` | `google.protobuf.Empty` | `google.protobuf.Empty` | 返回已是 `google.protobuf.Empty`，保持不动 |
| 6 | `UpdateUserPassword` | `UpdatePwdForm` | `UserPasswordForm` | `google.protobuf.Empty` | `google.protobuf.Empty` | 标准 `Update业务名`，仅调整非 `google.protobuf` 的入参 |
| 9 | `OptionBaseDict` | `google.protobuf.Empty` | `google.protobuf.Empty` | `ListBaseDictResponse` | `OptionBaseDictResponse` | `OptionBaseDict` 属于非标准方法，仅调整返回值 |
| 12 | `ExecuteBaseJob` | `ExecBaseJobRequest` | `ExecuteBaseJobRequest` | `google.protobuf.Empty` | `google.protobuf.Empty` | 返回已是 `google.protobuf.Empty`，保持不动 |
| 13 | `SetBaseRoleMenu` | `SetMenusRequest` | `SetBaseRoleMenuRequest` | `google.protobuf.Empty` | `google.protobuf.Empty` | 返回已是 `google.protobuf.Empty`，保持不动 |
| 14 | `ResetBaseUserPassword` | `ResetBaseUserPwdRequest` | `ResetBaseUserPasswordRequest` | `google.protobuf.Empty` | `google.protobuf.Empty` | 返回已是 `google.protobuf.Empty`，保持不动 |
| 15 | `OptionGoodsInfo` | `ListGoodsInfoRequest` | `OptionGoodsInfoRequest` | `ListGoodsInfoResponse` | `OptionGoodsInfoResponse` | 方法名改动后，入参与返回值同步跟随 |
| 18 | `GetOrderInfoShipment` | `google.protobuf.Int64Value` | `google.protobuf.Int64Value` | `OrderInfoShippedResponse` | `OrderInfoShipmentForm` | 标准 `Get业务名`，仅调整非 `google.protobuf` 的返回值 |
| 19 | `ShipOrderInfo` | `ShippedOrderInfoRequest` | `ShipOrderInfoRequest` | `google.protobuf.Empty` | `google.protobuf.Empty` | 返回已是 `google.protobuf.Empty`，保持不动 |
| 22 | `WechatLogin` | `WxLoginRequest` | `WechatLoginRequest` | `WxLoginResponse` | `WechatLoginResponse` | 方法名改动后，入参与返回值同步跟随 |
| 23 | `GetUserProfile` | `google.protobuf.Empty` | `google.protobuf.Empty` | `UserInfo` | `UserProfileForm` | 标准 `Get业务名`，仅调整非 `google.protobuf` 的返回值 |
| 24 | `UpdateUserProfile` | `UpdateUserInfoRequest` | `UserProfileForm` | `google.protobuf.Empty` | `google.protobuf.Empty` | 标准 `Update业务名`，仅调整非 `google.protobuf` 的入参 |
| 25 | `BindUserPhone` | `PhoneAuthRequest` | `BindUserPhoneRequest` | `PhoneAuthResponse` | `BindUserPhoneResponse` | 非标准方法，入参与返回值按方法名同步改名 |
| 26 | `GetBaseDict` | `google.protobuf.StringValue` | `google.protobuf.StringValue` | `ListBaseDictResponse` | `BaseDictForm` | 标准 `Get业务名`，入参因已是 `google.protobuf` 保持不动，仅调整返回值 |
| 28 | `BuyNowOrderInfo` | `CreateOrderInfoGoods` | `BuyNowOrderInfoRequest` | `ConfirmOrderInfoResponse` | `BuyNowOrderInfoResponse` | 方法名改动后，入参与返回值同步跟随 |
| 29 | `RepurchaseOrderInfo` | `OrderRepurchaseInfoRequest` | `RepurchaseOrderInfoRequest` | `ConfirmOrderInfoResponse` | `RepurchaseOrderInfoResponse` | 方法名改动后，入参与返回值同步跟随 |
| 40 | `SetUserCartSelection` | `SelectedUserCartRequest` | `SetUserCartSelectionRequest` | `google.protobuf.Empty` | `google.protobuf.Empty` | 返回已是 `google.protobuf.Empty`，保持不动 |

### 方法名不改，但按规则仍需调整 message

| 编号 | RPC | 当前入参 | 建议入参 | 当前返回 | 建议返回 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `GetUserInfo` | `google.protobuf.Empty` | `google.protobuf.Empty` | `UserInfo` | `UserInfoForm` | 标准 `Get业务名`，仅调整非 `google.protobuf` 的返回值 |
| 5 | `UpdateUserPhone` | `UpdatePhoneForm` | `UserPhoneForm` | `google.protobuf.Empty` | `google.protobuf.Empty` | 标准 `Update业务名`，仅调整非 `google.protobuf` 的入参 |
| 34 | `JsapiPay` | `PayRequest` | `JsapiPayRequest` | `JsapiPayResponse` | `JsapiPayResponse` | 非标准方法，入参按方法名收口 |
| 35 | `H5Pay` | `PayRequest` | `H5PayRequest` | `H5PayResponse` | `H5PayResponse` | 非标准方法，入参按方法名收口 |
| 39 | `UpdateUserCart` | `UpdateUserCartRequest` | `UserCartForm` | `google.protobuf.Empty` | `google.protobuf.Empty` | 标准 `Update业务名`，仅调整非 `google.protobuf` 的入参 |

### message 不用动

- 方法名会改但 message 不用动：
- 8 `RefreshBaseConfigCache`
- 27 `ConfirmOrderInfo`
- 方法名不改且 message 也不用动：
- 2 `GetUserProfile`
- 3 `UpdateUserProfile`
- 7 `ListBaseApi`
- 10 `StartBaseJob`
- 11 `StopBaseJob`
- 16 `ListGoodsSpec`
- 17 `RefundOrderInfo`
- 20 `OrderMonthReportList`
- 21 `OrderDayReportList`
- 30 `GetOrderInfoIdByOrderNo`
- 31 `CancelOrderInfo`
- 32 `RefundOrderInfo`
- 33 `ReceiveOrderInfo`
- 36 `PayNotify`
- 37 `BindRecommendAnonymousActor`
- 38 `ListUserCart`
- 41 `GetConfig`
- 42 `MultiUploadFile`
- 43 `UploadFile`
- 44 `DownloadFile`
- 45 `Captcha`
- 46 `Logout`
- 47 `RefreshToken`
- 48 `Login`

### 额外说明

- `2 GetUserProfile` 目前只有入参侧与标准 `Get业务名` 固定签名不一致，但当前入参已经是 `google.protobuf.Empty`，按你的口径保持不动。
- `16 ListGoodsSpec` 当前 `ListGoodsSpecRequest` 里已有实际参数值，本轮不再按 `List业务名 -> google.protobuf.Empty` 收口。
- `41 GetConfig` 的 `ConfigRequest` 是枚举类型，`ConfigResponse` 也按你的要求保持现状。
- `43 UploadFile` 按你的要求保持当前 `UploadFileInfo` / `FileInfo` 不动。
- `30 GetOrderInfoIdByOrderNo`、`36 PayNotify`、`37 BindRecommendAnonymousActor`、`44 DownloadFile`、`45 Captcha`、`46 Logout` 这些接口也存在“若纯按规则可以继续收口”的空间，但当前不合规侧已经是 `google.protobuf.*`，所以本轮不再改。
