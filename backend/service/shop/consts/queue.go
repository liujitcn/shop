package consts

import baseconst "shop/pkg/const"

const (
	// RECOMMEND_EVENT_REPORT 表示推荐事件上报队列。
	RECOMMEND_EVENT_REPORT baseconst.Queue = "recommend_event_report_queue"
	// RECOMMEND_SYNC_BASE_USER 表示推荐用户同步队列。
	RECOMMEND_SYNC_BASE_USER baseconst.Queue = "recommend_sync_base_user_queue"
	// RECOMMEND_DELETE_BASE_USER 表示推荐用户删除队列。
	RECOMMEND_DELETE_BASE_USER baseconst.Queue = "recommend_delete_base_user_queue"
	// RECOMMEND_SYNC_GOODS_INFO 表示推荐商品同步队列。
	RECOMMEND_SYNC_GOODS_INFO baseconst.Queue = "recommend_sync_goods_info_queue"
	// RECOMMEND_DELETE_GOODS_INFO 表示推荐商品删除队列。
	RECOMMEND_DELETE_GOODS_INFO baseconst.Queue = "recommend_delete_goods_info_queue"
	// RECOMMEND_EVENT 表示推荐事件回放队列。
	RECOMMEND_EVENT baseconst.Queue = "recommend_replay_event_queue"
	// COMMENT_AUDIT 表示评价审核队列。
	COMMENT_AUDIT baseconst.Queue = "comment_audit_queue"
	// COMMENT_SUMMARY_REFRESH 表示评价摘要刷新队列。
	COMMENT_SUMMARY_REFRESH baseconst.Queue = "comment_summary_refresh_queue"
)
