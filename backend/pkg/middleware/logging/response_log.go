package logging

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	responseLogByteLimit     = 32 * 1024
	responseListItemLimit    = 20
	responseNodeCountLimit   = 100
	responseTreeDepthLimit   = 5
	responseReasonCollection = "large_collection_response"
	responseReasonPayload    = "large_payload_response"
)

// responseCollectionHit 描述响应体里命中的大集合字段。
type responseCollectionHit struct {
	Path           string
	Kind           string
	TopLevelCount  int
	TotalNodeCount int
	MaxDepth       int
}

// responseCollectionStats 描述 repeated message 字段的节点统计结果。
type responseCollectionStats struct {
	TopLevelCount  int
	TotalNodeCount int
	MaxDepth       int
}

// responseLogSummary 描述写入日志的响应摘要。
type responseLogSummary struct {
	Truncated      bool   `json:"truncated"`
	Reason         string `json:"reason"`
	Field          string `json:"field,omitempty"`
	Kind           string `json:"kind,omitempty"`
	TopLevelCount  int    `json:"topLevelCount,omitempty"`
	TotalNodeCount int    `json:"totalNodeCount,omitempty"`
	MaxDepth       int    `json:"maxDepth,omitempty"`
	ByteSize       int    `json:"byteSize,omitempty"`
}

// extractResponse 提取响应体日志内容。
func extractResponse(reply interface{}) string {
	responseBody, err := marshalResponseBody(reply)
	// 响应对象可正常序列化时，优先基于真实响应结构判断是否需要摘要化。
	if err == nil {
		// 命中大集合响应时，仅记录摘要，避免把大列表或大树整段写进日志。
		if hit := findLargeResponseCollection(reply); hit != nil {
			return marshalResponseSummary(responseLogSummary{
				Truncated:      true,
				Reason:         responseReasonCollection,
				Field:          hit.Path,
				Kind:           hit.Kind,
				TopLevelCount:  hit.TopLevelCount,
				TotalNodeCount: hit.TotalNodeCount,
				MaxDepth:       hit.MaxDepth,
				ByteSize:       len(responseBody),
			})
		}
		// 未命中大集合但整体响应字节数仍然过大时，回退写入通用摘要。
		if len(responseBody) > responseLogByteLimit {
			return marshalResponseSummary(responseLogSummary{
				Truncated: true,
				Reason:    responseReasonPayload,
				ByteSize:  len(responseBody),
			})
		}
		return string(responseBody)
	}

	// 响应对象实现脱敏接口但无法直接序列化时，回退记录脱敏后的 JSON 字符串。
	if redacter, ok := reply.(Redacter); ok {
		return marshalFallbackText(redacter.Redact())
	}
	// 响应对象实现 Stringer 时，回退复用其字符串表示。
	if stringer, ok := reply.(fmt.Stringer); ok {
		return marshalFallbackText(stringer.String())
	}
	return marshalFallbackText(fmt.Sprintf("%+v", reply))
}

// marshalResponseBody 将响应对象统一序列化成 JSON。
func marshalResponseBody(reply interface{}) ([]byte, error) {
	// 空响应统一写成 null，便于和空对象区分。
	if reply == nil {
		return json.Marshal(nil)
	}

	// Proto 响应优先使用 protojson，保持字段命名和请求日志一致。
	if message, ok := reply.(proto.Message); ok {
		return protojson.MarshalOptions{
			UseProtoNames:   false,
			EmitUnpopulated: false,
		}.Marshal(message)
	}

	return json.Marshal(reply)
}

// marshalResponseSummary 将响应摘要编码成 JSON 字符串。
func marshalResponseSummary(summary responseLogSummary) string {
	body, err := json.Marshal(summary)
	// 摘要对象理论上总能被序列化；若异常失败，则退回到固定文本避免日志为空。
	if err != nil {
		return `{"truncated":true,"reason":"response_summary_marshal_failed"}`
	}
	return string(body)
}

// findLargeResponseCollection 查找响应里的大列表或大树字段。
func findLargeResponseCollection(reply interface{}) *responseCollectionHit {
	message, ok := reply.(proto.Message)
	// 非 Proto 响应当前不做字段级集合识别，仅依赖整体字节数兜底。
	if !ok {
		return nil
	}
	return findLargeCollectionInMessage(message.ProtoReflect(), "")
}

// findLargeCollectionInMessage 递归检查消息中的 repeated 字段。
func findLargeCollectionInMessage(message protoreflect.Message, parentPath string) *responseCollectionHit {
	fields := message.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		fieldPath := joinCollectionFieldPath(parentPath, field.JSONName())

		// 当前字段是 repeated 集合时，优先按集合规则判断是否需要摘要化。
		if field.IsList() {
			list := message.Get(field).List()
			// 空列表不参与日志裁剪判断，避免无意义遍历。
			if list.Len() == 0 {
				continue
			}

			// repeated message 既可能是普通列表，也可能是树形结构，需要继续统计节点规模。
			if field.Kind() == protoreflect.MessageKind {
				// 顶层条数已经明显超限时，直接按大列表处理，避免继续深度遍历。
				if list.Len() > responseListItemLimit {
					return &responseCollectionHit{
						Path:           fieldPath,
						Kind:           "list",
						TopLevelCount:  list.Len(),
						TotalNodeCount: list.Len(),
						MaxDepth:       1,
					}
				}

				stats := collectMessageListStats(list, 1)
				// 节点总数或树深命中阈值时，按树形摘要记录当前字段。
				if stats.TotalNodeCount > responseNodeCountLimit || stats.MaxDepth > responseTreeDepthLimit {
					return &responseCollectionHit{
						Path:           fieldPath,
						Kind:           "tree",
						TopLevelCount:  stats.TopLevelCount,
						TotalNodeCount: stats.TotalNodeCount,
						MaxDepth:       stats.MaxDepth,
					}
				}

				for j := 0; j < list.Len(); j++ {
					// 当前 repeated message 本身未超限时，继续检查子消息里是否存在更深层的大集合。
					if hit := findLargeCollectionInMessage(list.Get(j).Message(), fieldPath); hit != nil {
						return hit
					}
				}
				continue
			}

			// repeated 标量字段长度超限时，按普通列表处理。
			if list.Len() > responseListItemLimit {
				return &responseCollectionHit{
					Path:           fieldPath,
					Kind:           "list",
					TopLevelCount:  list.Len(),
					TotalNodeCount: list.Len(),
					MaxDepth:       1,
				}
			}
			continue
		}

		// 普通 message 字段存在时，递归检查其内部是否包含大集合。
		if field.Kind() == protoreflect.MessageKind && message.Has(field) {
			if hit := findLargeCollectionInMessage(message.Get(field).Message(), fieldPath); hit != nil {
				return hit
			}
		}
	}
	return nil
}

// collectMessageListStats 统计 repeated message 字段的节点规模。
func collectMessageListStats(list protoreflect.List, depth int) responseCollectionStats {
	stats := responseCollectionStats{
		TopLevelCount: list.Len(),
		MaxDepth:      depth,
	}
	for i := 0; i < list.Len(); i++ {
		nodeStats := collectMessageNodeStats(list.Get(i).Message(), depth)
		stats.TotalNodeCount += nodeStats.TotalNodeCount
		// 子树深度更深时，更新当前 repeated message 的最大深度。
		if nodeStats.MaxDepth > stats.MaxDepth {
			stats.MaxDepth = nodeStats.MaxDepth
		}
	}
	return stats
}

// collectMessageNodeStats 统计单个消息节点及其所有子树节点数量。
func collectMessageNodeStats(message protoreflect.Message, depth int) responseCollectionStats {
	stats := responseCollectionStats{
		TotalNodeCount: 1,
		MaxDepth:       depth,
	}

	descendantStats := collectMessageDescendantStats(message, depth)
	stats.TotalNodeCount += descendantStats.TotalNodeCount
	// 当前节点下方存在更深层子树时，更新节点统计深度。
	if descendantStats.MaxDepth > stats.MaxDepth {
		stats.MaxDepth = descendantStats.MaxDepth
	}
	return stats
}

// collectMessageDescendantStats 统计消息内部所有子集合的节点规模。
func collectMessageDescendantStats(message protoreflect.Message, depth int) responseCollectionStats {
	stats := responseCollectionStats{
		MaxDepth: depth,
	}
	fields := message.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)

		// 当前字段是 repeated message 时，把它视为当前节点下的一组子节点继续统计。
		if field.IsList() && field.Kind() == protoreflect.MessageKind {
			list := message.Get(field).List()
			// 空子节点列表不参与树规模统计。
			if list.Len() == 0 {
				continue
			}

			childStats := collectMessageListStats(list, depth+1)
			stats.TotalNodeCount += childStats.TotalNodeCount
			// 子节点树更深时，更新当前消息的最大深度。
			if childStats.MaxDepth > stats.MaxDepth {
				stats.MaxDepth = childStats.MaxDepth
			}
			continue
		}

		// 普通 message 包装字段存在时，继续向下寻找其内部的 repeated message 子树。
		if field.Kind() == protoreflect.MessageKind && message.Has(field) {
			childStats := collectMessageDescendantStats(message.Get(field).Message(), depth)
			stats.TotalNodeCount += childStats.TotalNodeCount
			// 包装消息里存在更深子树时，同步更新当前统计深度。
			if childStats.MaxDepth > stats.MaxDepth {
				stats.MaxDepth = childStats.MaxDepth
			}
		}
	}
	return stats
}

// joinCollectionFieldPath 拼接集合字段路径。
func joinCollectionFieldPath(parentPath string, fieldName string) string {
	// 根字段直接返回自身名称，避免路径前缀出现多余分隔符。
	if parentPath == "" {
		return fieldName
	}
	return parentPath + "." + fieldName
}
