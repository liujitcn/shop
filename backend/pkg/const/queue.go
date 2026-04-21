package _const

type Queue string

const (
	Log                  Queue = "log_queue"
	JobLog               Queue = "job_log_queue"
	RecommendEventReport Queue = "recommend_event_report_queue"
	GorseSyncBaseUser    Queue = "gorse_sync_base_user_queue"
	GorseDeleteBaseUser  Queue = "gorse_delete_base_user_queue"
	GorseSyncGoodsInfo   Queue = "gorse_sync_goods_info_queue"
	GorseDeleteGoodsInfo Queue = "gorse_delete_goods_info_queue"
	GorseRecommendEvent  Queue = "gorse_recommend_event_queue"
	GorseReplayEvent     Queue = "gorse_replay_event_queue"
)
