package _const

// Queue 定义项目内统一的队列名称类型。
type Queue string

const (
	// Log 表示通用日志异步写入队列。
	Log Queue = "log_queue"
	// JobLog 表示定时任务执行日志队列。
	JobLog Queue = "job_log_queue"
	// RecommendEventReport 表示推荐事件上报处理队列。
	RecommendEventReport Queue = "recommend_event_report_queue"
	// RecommendSyncBaseUser 表示推荐系统用户主数据同步队列。
	RecommendSyncBaseUser Queue = "recommend_sync_base_user_queue"
	// RecommendDeleteBaseUser 表示推荐系统用户删除同步队列。
	RecommendDeleteBaseUser Queue = "recommend_delete_base_user_queue"
	// RecommendSyncGoodsInfo 表示推荐系统商品主数据同步队列。
	RecommendSyncGoodsInfo Queue = "recommend_sync_goods_info_queue"
	// RecommendDeleteGoodsInfo 表示推荐系统商品删除同步队列。
	RecommendDeleteGoodsInfo Queue = "recommend_delete_goods_info_queue"
	// RecommendFeedbackEvent 表示推荐系统推荐反馈事件投递队列。
	RecommendFeedbackEvent Queue = "recommend_feedback_event_queue"
	// RecommendEvent 表示推荐系统历史推荐事件回放队列。
	RecommendEvent Queue = "recommend_replay_event_queue"
)
