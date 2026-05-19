package queue

import _const "shop/pkg/const"

// DispatchAiImageGenerate 投递 AI 图片生成消息。
func DispatchAiImageGenerate(taskID int64) {
	AddQueue(_const.AI_IMAGE_GENERATE, taskID)
}
