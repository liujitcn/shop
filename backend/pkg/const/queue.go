package _const

type Queue string

const (
	Log                  Queue = "log_queue"
	JobLog               Queue = "job_log_queue"
	RecommendEventReport Queue = "recommend_event_report_queue"
)
