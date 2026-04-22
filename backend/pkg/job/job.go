package job

import (
	"fmt"
	"shop/pkg/job/task"
	pkgQueue "shop/pkg/queue"
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

// Execute 执行任务并写入任务日志。
func (e *ExecJob) Execute() (err error) {
	baseJobLog := models.BaseJobLog{
		JobID:       e.JobId,                                // 定时任务id
		Input:       _string.ConvertAnyToJsonString(e.Args), // 任务参数
		ExecuteTime: time.Now(),                             // 执行时间
	}
	ret := make([]string, 0)

	defer func() {
		// 任务执行发生 panic 时，统一转成失败日志并返回错误。
		if panicValue := recover(); panicValue != nil {
			err = fmt.Errorf("任务执行异常: %v", panicValue)
		}

		if err != nil {
			e.Status = common.BaseJobLogStatus_FAIL
			e.ErrMsg = err.Error()
		} else {
			e.Status = common.BaseJobLogStatus_SUCCESS
			e.ErrMsg = ""
		}
		baseJobLog.Output = strings.Join(ret, "<br/>")
		baseJobLog.Status = int32(e.Status)
		baseJobLog.Error = e.ErrMsg
		baseJobLog.ProcessTime = int32(time.Since(baseJobLog.ExecuteTime).Milliseconds())
		pkgQueue.AddQueue(_const.JobLog, baseJobLog)
	}()

	ret, err = e.InvokeTarget.Exec(e.Args)
	return err
}

// LogJobFailureWithInput 使用原始入参记录任务失败日志。
func LogJobFailureWithInput(jobId int64, input string, err error) {
	// 没有实际错误时，无需生成失败日志。
	if err == nil {
		return
	}

	baseJobLog := models.BaseJobLog{
		JobID:       jobId,      // 定时任务id
		Input:       input,      // 任务参数
		ExecuteTime: time.Now(), // 执行时间
	}
	baseJobLog.Status = int32(common.BaseJobLogStatus_FAIL)
	baseJobLog.Error = err.Error()
	baseJobLog.ProcessTime = int32(time.Since(baseJobLog.ExecuteTime).Milliseconds())
	pkgQueue.AddQueue(_const.JobLog, baseJobLog)
}
