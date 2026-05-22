package _const

// Queue 定义项目内统一的队列名称类型。
type Queue string

const (
	// LOG 表示通用日志异步写入队列。
	LOG Queue = "log_queue"
	// JOB_LOG 表示定时任务执行日志队列。
	JOB_LOG Queue = "job_log_queue"
	// RECOMMEND_EVENT_REPORT 表示推荐事件上报处理队列。
	RECOMMEND_EVENT_REPORT Queue = "recommend_event_report_queue"
	// RECOMMEND_SYNC_BASE_USER 表示推荐系统用户主数据同步队列。
	RECOMMEND_SYNC_BASE_USER Queue = "recommend_sync_base_user_queue"
	// RECOMMEND_DELETE_BASE_USER 表示推荐系统用户删除同步队列。
	RECOMMEND_DELETE_BASE_USER Queue = "recommend_delete_base_user_queue"
	// RECOMMEND_SYNC_GOODS_INFO 表示推荐系统商品主数据同步队列。
	RECOMMEND_SYNC_GOODS_INFO Queue = "recommend_sync_goods_info_queue"
	// RECOMMEND_DELETE_GOODS_INFO 表示推荐系统商品删除同步队列。
	RECOMMEND_DELETE_GOODS_INFO Queue = "recommend_delete_goods_info_queue"
	// RECOMMEND_EVENT 表示推荐系统历史推荐事件回放队列。
	RECOMMEND_EVENT Queue = "recommend_replay_event_queue"
	// COMMENT_AUDIT 表示评价与讨论大模型审核队列。
	COMMENT_AUDIT Queue = "comment_audit_queue"
	// COMMENT_AI_REFRESH 表示商品评价 AI 摘要刷新队列。
	COMMENT_AI_REFRESH Queue = "comment_ai_refresh_queue"
)
