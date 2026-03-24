package job

import (
	"shop/pkg/utils"
	"shop/service/admin/task"
	"strings"
	"time"

	"shop/api/gen/go/common"
	_const "shop/pkg/const"
	"shop/pkg/gen/models"

	_string "github.com/liujitcn/go-utils/string"
)

type ExecJob struct {
	JobId        int64             // 任务ID
	Args         map[string]string // 任务参数
	InvokeTarget task.TaskExec
	Status       common.BaseJobLogStatus
	ErrMsg       string
}

// Run 函数任务执行
func (e *ExecJob) Run() {
	// 记录日志
	baseJobLog := models.BaseJobLog{
		JobID:       e.JobId,                                // 定时任务id
		Input:       _string.ConvertAnyToJsonString(e.Args), // 任务参数
		ExecuteTime: time.Now(),                             // 执行时间
	}
	ret, err := e.InvokeTarget.Exec(e.Args)
	if err != nil {
		e.Status = common.BaseJobLogStatus_FAIL
		e.ErrMsg = err.Error()
	} else {
		e.Status = common.BaseJobLogStatus_SUCCESS
	}
	// 执行结果
	baseJobLog.Output = strings.Join(ret, "<br/>")
	// 执行结果-成功
	baseJobLog.Status = int32(e.Status)
	baseJobLog.Error = e.ErrMsg
	// 执行时间
	baseJobLog.ProcessTime = int32(time.Now().Sub(baseJobLog.ExecuteTime).Milliseconds())
	// 加入日志队列
	utils.AddQueue(_const.JobLog, baseJobLog)
	return
}
