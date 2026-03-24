package _const

type Queue string

const (
	Log      Queue = "log_queue"
	ApiCheck Queue = "api_check_queue"
	JobLog   Queue = "job_log_queue"
)
