package content

import "testing"

// TestBuildSimilarityMap 验证内容相似结果基于语义向量而不是固定类目分桶。
func TestBuildSimilarityMap(t *testing.T) {
	documentList := []Document{
		{
			Id:         101,
			CategoryId: 11,
			Name:       "苹果手机壳",
			Desc:       "透明防摔手机保护壳",
			Detail:     "适配苹果手机，软边透明，轻薄防摔",
			Price:      2990,
			SaleNum:    120,
		},
		{
			Id:         102,
			CategoryId: 22,
			Name:       "苹果手机保护套",
			Desc:       "轻薄透明防摔壳",
			Detail:     "用于苹果手机的透明保护套，软壳防摔",
			Price:      3190,
			SaleNum:    98,
		},
		{
			Id:         103,
			CategoryId: 11,
			Name:       "婴儿奶瓶",
			Desc:       "宽口径耐高温奶瓶",
			Detail:     "适合新生儿喝奶，带防胀气奶嘴",
			Price:      4590,
			SaleNum:    230,
		},
	}

	similarityMap := BuildSimilarityMap(documentList, 2)
	scoredList := similarityMap[101]
	if len(scoredList) == 0 {
		t.Fatalf("expected similarity results for 101")
	}
	// 这里要求跨类目的语义相近商品排在前面，证明当前不再依赖同类目启发式。
	if scoredList[0].Id != 102 {
		t.Fatalf("unexpected top result: %+v", scoredList)
	}
	if len(scoredList) < 2 || scoredList[0].Score <= scoredList[1].Score {
		t.Fatalf("unexpected score ordering: %+v", scoredList)
	}
}
