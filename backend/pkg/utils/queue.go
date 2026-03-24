package utils

import (
	_const "shop/pkg/const"

	"github.com/go-kratos/kratos/v2/log"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"github.com/liujitcn/kratos-kit/sdk"
)

func AddQueue(queue _const.Queue, data any) {
	var err error
	id := string(queue)
	// 加入日志队列
	q := sdk.Runtime.GetQueue()
	if q != nil {
		m := make(map[string]interface{})
		m["data"] = data
		var message queueData.Message
		message, err = sdk.Runtime.GetStreamMessage(id, m)
		if err != nil {
			log.Errorf("GetStreamMessage error, %s", err.Error())
			//日志报错错误，不中断请求
		} else {
			err = q.Append(id, message)
			if err != nil {
				log.Errorf("Append message error, %s", err.Error())
			}
		}
	}
}
