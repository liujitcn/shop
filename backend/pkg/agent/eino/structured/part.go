package structured

import "shop/pkg/agent/eino/message"

// Part 表示结构化任务可传给模型的多模态输入片段。
type Part = message.ContentBlock

// TextPart 构造文本输入片段。
func TextPart(text string) *Part {
	return message.TextPart(text)
}

// ImageURLPart 构造远程图片输入片段。
func ImageURLPart(rawURL string) *Part {
	return message.ImageURLPart(rawURL)
}

// ImageDataPart 构造图片字节输入片段。
func ImageDataPart(data []byte, mimeType string) *Part {
	return message.ImageDataPart(data, mimeType)
}
