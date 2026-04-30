package _const

import commonv1 "shop/api/gen/go/common/v1"

const (
	// STATUS_ENABLE 表示业务记录处于启用状态，允许在前台展示、参与查询或继续被业务流程使用。
	STATUS_ENABLE = int32(commonv1.Status_ENABLE)
	// STATUS_DISABLE 表示业务记录处于禁用状态，通常需要从可用列表中过滤或禁止继续操作。
	STATUS_DISABLE = int32(commonv1.Status_DISABLE)
)

const (
	// BASE_CONFIG_SITE_SYSTEM 表示系统内部使用的配置项，适用于不直接区分管理端或移动端的全局配置。
	BASE_CONFIG_SITE_SYSTEM = int32(commonv1.BaseConfigSite_SYSTEM)
	// BASE_CONFIG_SITE_ADMIN 表示管理端使用的配置项，主要影响后台管理页面或后台接口行为。
	BASE_CONFIG_SITE_ADMIN = int32(commonv1.BaseConfigSite_ADMIN)
	// BASE_CONFIG_SITE_APP 表示移动端使用的配置项，主要影响商城端页面展示或移动端接口行为。
	BASE_CONFIG_SITE_APP = int32(commonv1.BaseConfigSite_APP)
)

const (
	// BASE_CONFIG_TYPE_TEXT 表示文本类型配置，适用于短文本、开关文案或普通字符串配置。
	BASE_CONFIG_TYPE_TEXT = int32(commonv1.BaseConfigType_TEXT)
	// BASE_CONFIG_TYPE_IMAGE 表示图片类型配置，配置值通常保存图片地址或资源标识。
	BASE_CONFIG_TYPE_IMAGE = int32(commonv1.BaseConfigType_IMAGE)
	// BASE_CONFIG_TYPE_RICH_TEXT 表示富文本类型配置，适用于公告、协议、说明等包含格式内容的配置。
	BASE_CONFIG_TYPE_RICH_TEXT = int32(commonv1.BaseConfigType_RICH_TEXT)
)

const (
	// BASE_JOB_LOG_STATUS_SUCCESS 表示定时任务执行成功，任务流程已按预期完成。
	BASE_JOB_LOG_STATUS_SUCCESS = int32(commonv1.BaseJobLogStatus_SUCCESS)
	// BASE_JOB_LOG_STATUS_FAIL 表示定时任务执行失败，需要结合错误信息排查失败原因。
	BASE_JOB_LOG_STATUS_FAIL = int32(commonv1.BaseJobLogStatus_FAIL)
)

const (
	// BASE_MENU_TYPE_FOLDER 表示目录节点，仅用于组织菜单层级，不直接承载具体页面。
	BASE_MENU_TYPE_FOLDER = int32(commonv1.BaseMenuType_FOLDER)
	// BASE_MENU_TYPE_MENU 表示菜单节点，通常对应一个可访问的后台页面或路由。
	BASE_MENU_TYPE_MENU = int32(commonv1.BaseMenuType_MENU)
	// BASE_MENU_TYPE_BUTTON 表示按钮权限节点，通常用于控制页面内具体操作权限。
	BASE_MENU_TYPE_BUTTON = int32(commonv1.BaseMenuType_BUTTON)
	// BASE_MENU_TYPE_EXT_LINK 表示外部链接节点，通常跳转到系统外部地址或第三方页面。
	BASE_MENU_TYPE_EXT_LINK = int32(commonv1.BaseMenuType_EXT_LINK)
)

const (
	// BASE_ROLE_DATA_SCOPE_ALL 表示角色拥有全部数据范围，可查看或操作授权业务内的所有数据。
	BASE_ROLE_DATA_SCOPE_ALL = int32(commonv1.BaseRoleDataScope_ALL)
	// BASE_ROLE_DATA_SCOPE_DEPT_AND_CHILDREN 表示角色拥有本部门及子部门数据范围。
	BASE_ROLE_DATA_SCOPE_DEPT_AND_CHILDREN = int32(commonv1.BaseRoleDataScope_DEPT_AND_CHILDREN)
	// BASE_ROLE_DATA_SCOPE_SELF_DEPT 表示角色仅拥有本部门数据范围，不包含子部门数据。
	BASE_ROLE_DATA_SCOPE_SELF_DEPT = int32(commonv1.BaseRoleDataScope_SELF_DEPT)
	// BASE_ROLE_DATA_SCOPE_SELF_USER 表示角色仅拥有本人数据范围，适用于只能查看或操作自己数据的场景。
	BASE_ROLE_DATA_SCOPE_SELF_USER = int32(commonv1.BaseRoleDataScope_SELF_USER)
)

const (
	// BASE_USER_GENDER_SECRET 表示用户选择保密性别，不对外展示具体性别。
	BASE_USER_GENDER_SECRET = int32(commonv1.BaseUserGender_SECRET)
	// BASE_USER_GENDER_BOY 表示用户性别为男。
	BASE_USER_GENDER_BOY = int32(commonv1.BaseUserGender_BOY)
	// BASE_USER_GENDER_GIRL 表示用户性别为女。
	BASE_USER_GENDER_GIRL = int32(commonv1.BaseUserGender_GIRL)
)

const (
	// USER_STORE_STATUS_PENDING_REVIEW 表示用户门店资料已提交但尚未完成审核。
	USER_STORE_STATUS_PENDING_REVIEW = int32(commonv1.UserStoreStatus_PENDING_REVIEW)
	// USER_STORE_STATUS_FAILED_REVIEW 表示用户门店审核未通过，需要用户补充或修正资料后重新提交。
	USER_STORE_STATUS_FAILED_REVIEW = int32(commonv1.UserStoreStatus_FAILED_REVIEW)
	// USER_STORE_STATUS_APPROVED 表示用户门店审核已通过，允许进入后续门店相关业务流程。
	USER_STORE_STATUS_APPROVED = int32(commonv1.UserStoreStatus_APPROVED)
)

const (
	// GOODS_STATUS_PUT_ON 表示商品已上架，可在商城端展示并参与购买流程。
	GOODS_STATUS_PUT_ON = int32(commonv1.GoodsStatus_PUT_ON)
	// GOODS_STATUS_PULL_OFF 表示商品已下架，通常不再对用户展示或允许下单。
	GOODS_STATUS_PULL_OFF = int32(commonv1.GoodsStatus_PULL_OFF)
)

const (
	// ORDER_STATUS_CREATED 表示订单已创建但尚未支付，用户仍可继续支付或取消订单。
	ORDER_STATUS_CREATED = int32(commonv1.OrderStatus_CREATED)
	// ORDER_STATUS_PAID 表示订单已完成支付并等待商家发货。
	ORDER_STATUS_PAID = int32(commonv1.OrderStatus_PAID)
	// ORDER_STATUS_SHIPPED 表示订单已发货并等待用户确认收货。
	ORDER_STATUS_SHIPPED = int32(commonv1.OrderStatus_SHIPPED)
	// ORDER_STATUS_WAIT_REVIEW 表示订单已收货并等待用户评价。
	ORDER_STATUS_WAIT_REVIEW = int32(commonv1.OrderStatus_WAIT_REVIEW)
	// ORDER_STATUS_COMPLETED 表示订单已完成，通常不再进入支付、发货或收货流程。
	ORDER_STATUS_COMPLETED = int32(commonv1.OrderStatus_COMPLETED)
	// ORDER_STATUS_REFUNDING 表示订单已进入退款相关状态，后续需要结合退款单判断具体进度。
	ORDER_STATUS_REFUNDING = int32(commonv1.OrderStatus_REFUNDING)
	// ORDER_STATUS_CANCELED 表示订单已取消，通常不允许继续支付、发货或确认收货。
	ORDER_STATUS_CANCELED = int32(commonv1.OrderStatus_CANCELED)
	// ORDER_STATUS_DELETED 表示订单已删除，通常只用于用户侧隐藏或软删除后的展示过滤。
	ORDER_STATUS_DELETED = int32(commonv1.OrderStatus_DELETED)
)

const (
	// ORDER_PAY_TYPE_ONLINE_PAY 表示订单使用在线支付方式，需要走支付渠道完成扣款。
	ORDER_PAY_TYPE_ONLINE_PAY = int32(commonv1.OrderPayType_ONLINE_PAY)
	// ORDER_PAY_TYPE_CASH_ON_DELIVERY 表示订单使用货到付款方式，不在下单时立即完成线上扣款。
	ORDER_PAY_TYPE_CASH_ON_DELIVERY = int32(commonv1.OrderPayType_CASH_ON_DELIVERY)
)

const (
	// ORDER_PAY_CHANNEL_WX_PAY 表示订单通过微信支付渠道完成支付。
	ORDER_PAY_CHANNEL_WX_PAY = int32(commonv1.OrderPayChannel_WX_PAY)
	// ORDER_PAY_CHANNEL_UNION_PAY 表示订单通过银联支付渠道完成支付。
	ORDER_PAY_CHANNEL_UNION_PAY = int32(commonv1.OrderPayChannel_UNION_PAY)
)

const (
	// ORDER_DELIVERY_TIME_ALL_TIME 表示配送时间不限，周一至周日均可配送。
	ORDER_DELIVERY_TIME_ALL_TIME = int32(commonv1.OrderDeliveryTime_ALL_TIME)
	// ORDER_DELIVERY_TIME_WEEKDAY 表示仅工作日配送，通常指周一至周五。
	ORDER_DELIVERY_TIME_WEEKDAY = int32(commonv1.OrderDeliveryTime_WEEKDAY)
	// ORDER_DELIVERY_TIME_WEEKEND 表示仅周末配送，通常指周六至周日。
	ORDER_DELIVERY_TIME_WEEKEND = int32(commonv1.OrderDeliveryTime_WEEKEND)
)

const (
	// ORDER_CANCEL_REASON_GOODS_NO_STOCK 表示因商品无货取消订单。
	ORDER_CANCEL_REASON_GOODS_NO_STOCK = int32(commonv1.OrderCancelReason_CANCEL_GOODS_NO_STOCK)
	// ORDER_CANCEL_REASON_SELF 表示用户主观不想要了而取消订单。
	ORDER_CANCEL_REASON_SELF = int32(commonv1.OrderCancelReason_CANCEL_SELF)
	// ORDER_CANCEL_REASON_GOODS_ERROR 表示因商品信息填写或选择错误取消订单。
	ORDER_CANCEL_REASON_GOODS_ERROR = int32(commonv1.OrderCancelReason_CANCEL_GOODS_ERROR)
	// ORDER_CANCEL_REASON_ADDRESS_ERROR 表示因收货地址信息填写错误取消订单。
	ORDER_CANCEL_REASON_ADDRESS_ERROR = int32(commonv1.OrderCancelReason_CANCEL_ADDRESS_ERROR)
	// ORDER_CANCEL_REASON_GOODS_DISCOUNT 表示因商品降价等价格变化原因取消订单。
	ORDER_CANCEL_REASON_GOODS_DISCOUNT = int32(commonv1.OrderCancelReason_CANCEL_GOODS_DISCOUNT)
	// ORDER_CANCEL_REASON_OTHER 表示除预置原因外的其他取消原因。
	ORDER_CANCEL_REASON_OTHER = int32(commonv1.OrderCancelReason_CANCEL_OTHER)
)

const (
	// ORDER_BILL_STATUS_NO_CHECK 表示订单尚未对账，需要等待后续对账任务处理。
	ORDER_BILL_STATUS_NO_CHECK = int32(commonv1.OrderBillStatus_NO_CHECK)
	// ORDER_BILL_STATUS_CHECK_SUCCESS 表示订单对账成功，业务订单与支付账单匹配。
	ORDER_BILL_STATUS_CHECK_SUCCESS = int32(commonv1.OrderBillStatus_CHECK_SUCCESS)
	// ORDER_BILL_STATUS_CHECK_FAIL 表示订单对账失败，需要人工或补偿流程排查差异。
	ORDER_BILL_STATUS_CHECK_FAIL = int32(commonv1.OrderBillStatus_CHECK_FAIL)
)

const (
	// ORDER_REFUND_REASON_GOODS_NO_STOCK 表示因商品无货发起退款。
	ORDER_REFUND_REASON_GOODS_NO_STOCK = int32(commonv1.OrderRefundReason_REFUND_GOODS_NO_STOCK)
	// ORDER_REFUND_REASON_SELF 表示用户主观不想要了而发起退款。
	ORDER_REFUND_REASON_SELF = int32(commonv1.OrderRefundReason_REFUND_SELF)
	// ORDER_REFUND_REASON_GOODS_ERROR 表示因商品信息填写或选择错误发起退款。
	ORDER_REFUND_REASON_GOODS_ERROR = int32(commonv1.OrderRefundReason_REFUND_GOODS_ERROR)
	// ORDER_REFUND_REASON_ADDRESS_ERROR 表示因收货地址信息填写错误发起退款。
	ORDER_REFUND_REASON_ADDRESS_ERROR = int32(commonv1.OrderRefundReason_REFUND_ADDRESS_ERROR)
	// ORDER_REFUND_REASON_GOODS_DISCOUNT 表示因商品降价等价格变化原因发起退款。
	ORDER_REFUND_REASON_GOODS_DISCOUNT = int32(commonv1.OrderRefundReason_REFUND_GOODS_DISCOUNT)
	// ORDER_REFUND_REASON_OTHER 表示除预置原因外的其他退款原因。
	ORDER_REFUND_REASON_OTHER = int32(commonv1.OrderRefundReason_REFUND_OTHER)
)

const (
	// SHOP_BANNER_SITE_INDEX 表示轮播图展示在商城首页。
	SHOP_BANNER_SITE_INDEX = int32(commonv1.ShopBannerSite_INDEX)
	// SHOP_BANNER_SITE_CATEGORY 表示轮播图展示在商品分类页。
	SHOP_BANNER_SITE_CATEGORY = int32(commonv1.ShopBannerSite_CATEGORY)
)

const (
	// SHOP_BANNER_TYPE_BANNER_GOODS_DETAIL 表示轮播图跳转到商品详情页。
	SHOP_BANNER_TYPE_BANNER_GOODS_DETAIL = int32(commonv1.ShopBannerType_BANNER_GOODS_DETAIL)
	// SHOP_BANNER_TYPE_CATEGORY_DETAIL 表示轮播图跳转到分类详情页。
	SHOP_BANNER_TYPE_CATEGORY_DETAIL = int32(commonv1.ShopBannerType_CATEGORY_DETAIL)
	// SHOP_BANNER_TYPE_WEB_VIEW 表示轮播图跳转到 H5 页面或外部 WebView 地址。
	SHOP_BANNER_TYPE_WEB_VIEW = int32(commonv1.ShopBannerType_WEB_VIEW)
	// SHOP_BANNER_TYPE_MINI 表示轮播图跳转到其他小程序。
	SHOP_BANNER_TYPE_MINI = int32(commonv1.ShopBannerType_MINI)
)

const (
	// PAY_BILL_STATUS_NO_COMPARE 表示支付账单尚未完成比对。
	PAY_BILL_STATUS_NO_COMPARE = int32(commonv1.PayBillStatus_NO_COMPARE)
	// PAY_BILL_STATUS_NO_ERROR 表示支付账单比对无误差。
	PAY_BILL_STATUS_NO_ERROR = int32(commonv1.PayBillStatus_NO_ERROR)
	// PAY_BILL_STATUS_HAS_ERROR 表示支付账单比对存在误差，需要继续排查或处理。
	PAY_BILL_STATUS_HAS_ERROR = int32(commonv1.PayBillStatus_HAS_ERROR)
)

const (
	// RECOMMEND_SCENE_HOME 表示首页推荐场景，用于首页商品或内容推荐。
	RECOMMEND_SCENE_HOME = int32(commonv1.RecommendScene_HOME)
	// RECOMMEND_SCENE_GOODS_DETAIL 表示商品详情推荐场景，用于详情页关联推荐。
	RECOMMEND_SCENE_GOODS_DETAIL = int32(commonv1.RecommendScene_GOODS_DETAIL)
	// RECOMMEND_SCENE_CART 表示购物车推荐场景，用于购物车页补充推荐。
	RECOMMEND_SCENE_CART = int32(commonv1.RecommendScene_CART)
	// RECOMMEND_SCENE_PROFILE 表示个人中心推荐场景，用于用户中心页个性化推荐。
	RECOMMEND_SCENE_PROFILE = int32(commonv1.RecommendScene_PROFILE)
	// RECOMMEND_SCENE_ORDER_DETAIL 表示订单详情推荐场景，用于订单详情页关联推荐。
	RECOMMEND_SCENE_ORDER_DETAIL = int32(commonv1.RecommendScene_ORDER_DETAIL)
	// RECOMMEND_SCENE_ORDER_PAID 表示支付成功推荐场景，用于支付完成后的追加推荐。
	RECOMMEND_SCENE_ORDER_PAID = int32(commonv1.RecommendScene_ORDER_PAID)
)

const (
	// RECOMMEND_ACTOR_TYPE_ANONYMOUS 表示匿名推荐主体，适用于未登录用户或临时访客。
	RECOMMEND_ACTOR_TYPE_ANONYMOUS = int32(commonv1.RecommendActorType_ANONYMOUS_ACTOR)
	// RECOMMEND_ACTOR_TYPE_USER 表示登录用户推荐主体，适用于已识别用户的个性化推荐。
	RECOMMEND_ACTOR_TYPE_USER = int32(commonv1.RecommendActorType_USER_ACTOR)
)

const (
	// RECOMMEND_STRATEGY_REMOTE 表示 Gorse 推荐策略，推荐结果来自外部推荐服务。
	RECOMMEND_STRATEGY_REMOTE = int32(commonv1.RecommendStrategy_REMOTE_STRATEGY)
	// RECOMMEND_STRATEGY_LOCAL 表示本地推荐策略，推荐结果由本地同类目等规则计算得出。
	RECOMMEND_STRATEGY_LOCAL = int32(commonv1.RecommendStrategy_LOCAL_STRATEGY)
)

const (
	// RECOMMEND_EVENT_TYPE_EXPOSURE 表示推荐曝光事件，用于记录推荐内容被展示给用户。
	RECOMMEND_EVENT_TYPE_EXPOSURE = int32(commonv1.RecommendEventType_EXPOSURE)
	// RECOMMEND_EVENT_TYPE_CLICK 表示推荐点击事件，用于记录用户点击推荐内容。
	RECOMMEND_EVENT_TYPE_CLICK = int32(commonv1.RecommendEventType_CLICK)
	// RECOMMEND_EVENT_TYPE_VIEW 表示商品浏览事件，用于记录用户查看商品详情行为。
	RECOMMEND_EVENT_TYPE_VIEW = int32(commonv1.RecommendEventType_VIEW)
	// RECOMMEND_EVENT_TYPE_COLLECT 表示商品收藏事件，用于记录用户收藏商品行为。
	RECOMMEND_EVENT_TYPE_COLLECT = int32(commonv1.RecommendEventType_COLLECT)
	// RECOMMEND_EVENT_TYPE_ADD_CART 表示商品加购事件，用于记录用户将商品加入购物车行为。
	RECOMMEND_EVENT_TYPE_ADD_CART = int32(commonv1.RecommendEventType_ADD_CART)
	// RECOMMEND_EVENT_TYPE_ORDER_CREATE 表示下单事件，用于记录用户创建订单行为。
	RECOMMEND_EVENT_TYPE_ORDER_CREATE = int32(commonv1.RecommendEventType_ORDER_CREATE)
	// RECOMMEND_EVENT_TYPE_ORDER_PAY 表示支付事件，用于记录用户完成订单支付行为。
	RECOMMEND_EVENT_TYPE_ORDER_PAY = int32(commonv1.RecommendEventType_ORDER_PAY)
)

const (
	// ADVANCE_DATA_TYPE_USER 表示 Gorse 推荐高级调试中的用户数据。
	ADVANCE_DATA_TYPE_USER = int32(commonv1.AdvanceDataType_USER_RRADT)
	// ADVANCE_DATA_TYPE_ITEM 表示 Gorse 推荐高级调试中的商品数据。
	ADVANCE_DATA_TYPE_ITEM = int32(commonv1.AdvanceDataType_ITEM_RRADT)
	// ADVANCE_DATA_TYPE_FEEDBACK 表示 Gorse 推荐高级调试中的反馈数据。
	ADVANCE_DATA_TYPE_FEEDBACK = int32(commonv1.AdvanceDataType_FEEDBACK_RRADT)
)

const (
	// COMMENT_STATUS_PENDING_REVIEW 表示评价已提交但尚未审核，通常不应直接对其他用户公开展示。
	COMMENT_STATUS_PENDING_REVIEW = int32(commonv1.CommentStatus_PENDING_REVIEW_CS)
	// COMMENT_STATUS_APPROVED 表示评价已审核通过，允许在商品评价、讨论或统计场景中公开使用。
	COMMENT_STATUS_APPROVED = int32(commonv1.CommentStatus_APPROVED_CS)
	// COMMENT_STATUS_REJECTED 表示评价审核不通过，不允许在前台公开展示。
	COMMENT_STATUS_REJECTED = int32(commonv1.CommentStatus_REJECTED_CS)
)

const (
	// COMMENT_REVIEW_TARGET_TYPE_COMMENT 表示审核目标为评价主记录。
	COMMENT_REVIEW_TARGET_TYPE_COMMENT = int32(commonv1.CommentReviewTargetType_COMMENT_REVIEW_TARGET_TYPE_COMMENT)
	// COMMENT_REVIEW_TARGET_TYPE_DISCUSSION 表示审核目标为评价讨论记录。
	COMMENT_REVIEW_TARGET_TYPE_DISCUSSION = int32(commonv1.CommentReviewTargetType_COMMENT_REVIEW_TARGET_TYPE_DISCUSSION)
)

const (
	// COMMENT_REVIEW_TYPE_AI 表示大模型自动审核。
	COMMENT_REVIEW_TYPE_AI = int32(commonv1.CommentReviewType_COMMENT_REVIEW_TYPE_AI)
	// COMMENT_REVIEW_TYPE_MANUAL 表示后台人工审核。
	COMMENT_REVIEW_TYPE_MANUAL = int32(commonv1.CommentReviewType_COMMENT_REVIEW_TYPE_MANUAL)
)

const (
	// COMMENT_REVIEW_STATUS_APPROVED 表示本次审核通过。
	COMMENT_REVIEW_STATUS_APPROVED = int32(commonv1.CommentReviewStatus_COMMENT_REVIEW_STATUS_APPROVED)
	// COMMENT_REVIEW_STATUS_REJECTED 表示本次审核不通过。
	COMMENT_REVIEW_STATUS_REJECTED = int32(commonv1.CommentReviewStatus_COMMENT_REVIEW_STATUS_REJECTED)
	// COMMENT_REVIEW_STATUS_EXCEPTION 表示本次审核异常。
	COMMENT_REVIEW_STATUS_EXCEPTION = int32(commonv1.CommentReviewStatus_COMMENT_REVIEW_STATUS_EXCEPTION)
)

const (
	// COMMENT_AI_SCENE_OVERVIEW 表示商品详情页评价 AI 摘要场景，用于聚合展示商品评价概览。
	COMMENT_AI_SCENE_OVERVIEW = int32(commonv1.CommentAiScene_OVERVIEW)
	// COMMENT_AI_SCENE_LIST 表示评价列表页 AI 摘要场景，用于在评价列表中展示摘要卡片。
	COMMENT_AI_SCENE_LIST = int32(commonv1.CommentAiScene_LIST)
)

const (
	// COMMENT_REACTION_TARGET_TYPE_COMMENT 表示互动目标是单条评价内容。
	COMMENT_REACTION_TARGET_TYPE_COMMENT = int32(commonv1.CommentReactionTargetType_COMMENT)
	// COMMENT_REACTION_TARGET_TYPE_DISCUSSION 表示互动目标是评价讨论内容。
	COMMENT_REACTION_TARGET_TYPE_DISCUSSION = int32(commonv1.CommentReactionTargetType_DISCUSSION)
	// COMMENT_REACTION_TARGET_TYPE_AI 表示互动目标是评价 AI 摘要内容。
	COMMENT_REACTION_TARGET_TYPE_AI = int32(commonv1.CommentReactionTargetType_AI)
)

const (
	// COMMENT_REACTION_TYPE_LIKE 表示点赞互动，用于表达用户对评价、讨论或 AI 摘要的正向反馈。
	COMMENT_REACTION_TYPE_LIKE = int32(commonv1.CommentReactionType_LIKE)
	// COMMENT_REACTION_TYPE_DISLIKE 表示点踩互动，用于表达用户对评价、讨论或 AI 摘要的负向反馈。
	COMMENT_REACTION_TYPE_DISLIKE = int32(commonv1.CommentReactionType_DISLIKE)
)

const (
	// COMMENT_FILTER_TYPE_ALL 表示查询全部评价，不按图片、评分或标签额外过滤。
	COMMENT_FILTER_TYPE_ALL = int32(commonv1.CommentFilterType_COMMENT_FILTER_ALL)
	// COMMENT_FILTER_TYPE_MEDIA 表示仅查询包含图片等媒体内容的评价。
	COMMENT_FILTER_TYPE_MEDIA = int32(commonv1.CommentFilterType_COMMENT_FILTER_MEDIA)
	// COMMENT_FILTER_TYPE_GOOD 表示仅查询好评评价，通常对应较高商品评分。
	COMMENT_FILTER_TYPE_GOOD = int32(commonv1.CommentFilterType_COMMENT_FILTER_GOOD)
	// COMMENT_FILTER_TYPE_MIDDLE 表示仅查询中评评价，通常对应中间商品评分。
	COMMENT_FILTER_TYPE_MIDDLE = int32(commonv1.CommentFilterType_COMMENT_FILTER_MIDDLE)
	// COMMENT_FILTER_TYPE_BAD 表示仅查询差评评价，通常对应较低商品评分。
	COMMENT_FILTER_TYPE_BAD = int32(commonv1.CommentFilterType_COMMENT_FILTER_BAD)
	// COMMENT_FILTER_TYPE_TAG 表示按内容标签查询评价。
	COMMENT_FILTER_TYPE_TAG = int32(commonv1.CommentFilterType_COMMENT_FILTER_TAG)
)

const (
	// COMMENT_SORT_TYPE_DEFAULT 表示评价列表使用默认推荐排序。
	COMMENT_SORT_TYPE_DEFAULT = int32(commonv1.CommentSortType_COMMENT_SORT_DEFAULT)
	// COMMENT_SORT_TYPE_LATEST 表示评价列表按最新发布时间排序。
	COMMENT_SORT_TYPE_LATEST = int32(commonv1.CommentSortType_COMMENT_SORT_LATEST)
)

const (
	// RESOURCE_TYPE_TRANSACTION 表示支付通知资源类型为交易支付。
	RESOURCE_TYPE_TRANSACTION = int32(commonv1.ResourceType_TRANSACTION)
	// RESOURCE_TYPE_REFUND 表示支付通知资源类型为退款。
	RESOURCE_TYPE_REFUND = int32(commonv1.ResourceType_REFUND)
)

const (
	// ANALYTICS_TIME_TYPE_WEEK 表示按周统计分析数据。
	ANALYTICS_TIME_TYPE_WEEK = int32(commonv1.AnalyticsTimeType_ANALYTICS_TIME_TYPE_WEEK)
	// ANALYTICS_TIME_TYPE_MONTH 表示按月统计分析数据。
	ANALYTICS_TIME_TYPE_MONTH = int32(commonv1.AnalyticsTimeType_ANALYTICS_TIME_TYPE_MONTH)
	// ANALYTICS_TIME_TYPE_YEAR 表示按年统计分析数据。
	ANALYTICS_TIME_TYPE_YEAR = int32(commonv1.AnalyticsTimeType_ANALYTICS_TIME_TYPE_YEAR)
)

const (
	// ANALYTICS_SERIES_TYPE_BAR 表示柱状图系列。
	ANALYTICS_SERIES_TYPE_BAR = int32(commonv1.AnalyticsSeriesType_ANALYTICS_SERIES_TYPE_BAR)
	// ANALYTICS_SERIES_TYPE_LINE 表示折线图系列。
	ANALYTICS_SERIES_TYPE_LINE = int32(commonv1.AnalyticsSeriesType_ANALYTICS_SERIES_TYPE_LINE)
)

const (
	// ERROR_REASON_INVALID_ARGUMENT 表示请求参数错误。
	ERROR_REASON_INVALID_ARGUMENT = int32(commonv1.ErrorReason_INVALID_ARGUMENT)
	// ERROR_REASON_UNAUTHENTICATED 表示用户未通过认证。
	ERROR_REASON_UNAUTHENTICATED = int32(commonv1.ErrorReason_UNAUTHENTICATED)
	// ERROR_REASON_PERMISSION_DENIED 表示用户没有权限。
	ERROR_REASON_PERMISSION_DENIED = int32(commonv1.ErrorReason_PERMISSION_DENIED)
	// ERROR_REASON_RESOURCE_NOT_FOUND 表示资源不存在。
	ERROR_REASON_RESOURCE_NOT_FOUND = int32(commonv1.ErrorReason_RESOURCE_NOT_FOUND)
	// ERROR_REASON_CONFLICT 表示当前状态冲突。
	ERROR_REASON_CONFLICT = int32(commonv1.ErrorReason_CONFLICT)
	// ERROR_REASON_INTERNAL_ERROR 表示服务内部异常。
	ERROR_REASON_INTERNAL_ERROR = int32(commonv1.ErrorReason_INTERNAL_ERROR)
)

const (
	// ORDER_STATUS_UNKNOWN 表示订单状态未指定，默认值为 0。
	ORDER_STATUS_UNKNOWN = int32(commonv1.OrderStatus_UNKNOWN_OS)
	// RECOMMEND_SCENE_UNKNOWN 表示推荐场景未指定，默认值为 0。
	RECOMMEND_SCENE_UNKNOWN = int32(commonv1.RecommendScene_UNKNOWN_RS)
	// RECOMMEND_ACTOR_TYPE_UNKNOWN 表示推荐主体类型未指定，默认值为 0。
	RECOMMEND_ACTOR_TYPE_UNKNOWN = int32(commonv1.RecommendActorType_UNKNOWN_RAT)
	// RECOMMEND_STRATEGY_UNKNOWN 表示推荐策略未指定，默认值为 0。
	RECOMMEND_STRATEGY_UNKNOWN = int32(commonv1.RecommendStrategy_UNKNOWN_RST)
	// RECOMMEND_EVENT_TYPE_UNKNOWN 表示推荐事件类型未指定，默认值为 0。
	RECOMMEND_EVENT_TYPE_UNKNOWN = int32(commonv1.RecommendEventType_UNKNOWN_RET)
	// ADVANCE_DATA_TYPE_UNKNOWN 表示 Gorse 推荐高级调试数据类型未指定，默认值为 0。
	ADVANCE_DATA_TYPE_UNKNOWN = int32(commonv1.AdvanceDataType_UNKNOWN_RRADT)
)
