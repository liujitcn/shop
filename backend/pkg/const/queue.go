package _const

type Queue string

const (
	Log                       Queue = "log_queue"
	JobLog                    Queue = "job_log_queue"
	RecommendExposureEvent    Queue = "recommend_exposure_event_queue"
	RecommendGoodsActionEvent Queue = "recommend_goods_action_event_queue"
)
