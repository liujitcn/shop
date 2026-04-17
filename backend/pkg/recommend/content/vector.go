package content

import (
	"hash/fnv"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/tiktoken-go/tokenizer"
)

const (
	// DefaultDimension 表示默认内容向量维度。
	DefaultDimension = 192
	// DefaultEmbeddingName 表示内容向量在训练样本中的默认名称。
	DefaultEmbeddingName = "goods_content"
	// maxDetailRunes 表示详情字段参与向量化时的最大 rune 数。
	maxDetailRunes = 512
)

// Document 表示内容相似与内容向量计算所需的商品文档。
type Document struct {
	Id         int64
	CategoryId int64
	Name       string
	Desc       string
	Detail     string
	Price      int64
	SaleNum    int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Vector 表示归一化后的内容向量。
type Vector struct {
	Values []float32
}

// ScoredDocument 表示带相似度的文档结果。
type ScoredDocument struct {
	Id    int64
	Score float64
}

var (
	contentCodecOnce sync.Once
	contentCodec     tokenizer.Codec
)

// BuildDocumentVector 构建单个商品的内容向量。
func BuildDocumentVector(document Document, dimension int) Vector {
	if dimension <= 0 {
		dimension = DefaultDimension
	}
	values := make([]float32, dimension)
	addWeightedText(values, document.Name, 3)
	addWeightedText(values, document.Desc, 2)
	addWeightedText(values, truncateRunes(document.Detail, maxDetailRunes), 1)
	addWeightedToken(values, "category:"+strconv.FormatInt(document.CategoryId, 10), 2)
	addWeightedToken(values, "price_bucket:"+strconv.Itoa(bucketPrice(document.Price)), 1.5)
	addWeightedToken(values, "sale_bucket:"+strconv.Itoa(bucketSale(document.SaleNum)), 1)
	addWeightedToken(values, "age_bucket:"+strconv.Itoa(bucketAge(document.CreatedAt, document.UpdatedAt)), 0.8)
	normalizeVector(values)
	return Vector{Values: values}
}

// CosineSimilarity 计算两个归一化向量的余弦相似度。
func CosineSimilarity(left []float32, right []float32) float64 {
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	size := len(left)
	if len(right) < size {
		size = len(right)
	}
	score := 0.0
	for index := 0; index < size; index++ {
		score += float64(left[index] * right[index])
	}
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

// BuildSimilarityMap 基于内容向量构建每个商品的 TopN 相似商品。
func BuildSimilarityMap(documentList []Document, limit int64) map[int64][]ScoredDocument {
	result := make(map[int64][]ScoredDocument)
	if limit <= 0 {
		return result
	}
	filteredDocumentList := make([]Document, 0, len(documentList))
	vectorMap := make(map[int64]Vector, len(documentList))
	for _, item := range documentList {
		if item.Id <= 0 {
			continue
		}
		filteredDocumentList = append(filteredDocumentList, item)
		vectorMap[item.Id] = BuildDocumentVector(item, DefaultDimension)
	}
	for _, baseDocument := range filteredDocumentList {
		baseVector := vectorMap[baseDocument.Id]
		scoredList := make([]ScoredDocument, 0, len(filteredDocumentList))
		for _, targetDocument := range filteredDocumentList {
			if targetDocument.Id <= 0 || targetDocument.Id == baseDocument.Id {
				continue
			}
			score := CosineSimilarity(baseVector.Values, vectorMap[targetDocument.Id].Values)
			if score <= 0 {
				continue
			}
			scoredList = append(scoredList, ScoredDocument{
				Id:    targetDocument.Id,
				Score: score,
			})
		}
		sort.SliceStable(scoredList, func(i, j int) bool {
			if scoredList[i].Score == scoredList[j].Score {
				return scoredList[i].Id < scoredList[j].Id
			}
			return scoredList[i].Score > scoredList[j].Score
		})
		if int64(len(scoredList)) > limit {
			scoredList = scoredList[:limit]
		}
		result[baseDocument.Id] = scoredList
	}
	return result
}

// addWeightedText 把文本按权重写入向量。
func addWeightedText(values []float32, text string, weight float32) {
	for _, token := range collectTokens(text) {
		addWeightedToken(values, token, weight)
	}
}

// addWeightedToken 把单个 token 投影到固定维度向量。
func addWeightedToken(values []float32, token string, weight float32) {
	token = strings.TrimSpace(strings.ToLower(token))
	if len(values) == 0 || token == "" || weight == 0 {
		return
	}
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(token))
	index := int(hasher.Sum64() % uint64(len(values)))
	signHasher := fnv.New64a()
	_, _ = signHasher.Write([]byte("sign:" + token))
	sign := float32(1)
	if signHasher.Sum64()%2 == 0 {
		sign = -1
	}
	values[index] += sign * weight
}

// collectTokens 把文本拆成可用于近似语义向量的 token 列表。
func collectTokens(text string) []string {
	trimmedText := strings.TrimSpace(strings.ToLower(text))
	if trimmedText == "" {
		return []string{}
	}
	tokenSet := make(map[string]struct{})
	result := make([]string, 0, 16)
	appendToken := func(token string) {
		token = sanitizeToken(token)
		if token == "" {
			return
		}
		if _, exists := tokenSet[token]; exists {
			return
		}
		tokenSet[token] = struct{}{}
		result = append(result, token)
	}
	codec := getDefaultCodec()
	if codec != nil {
		_, tokenList, err := codec.Encode(trimmedText)
		if err == nil {
			for _, token := range tokenList {
				appendToken(token)
			}
		}
	}
	for _, token := range fallbackTokens(trimmedText) {
		appendToken(token)
	}
	return result
}

// getDefaultCodec 获取默认的文本编码器。
func getDefaultCodec() tokenizer.Codec {
	contentCodecOnce.Do(func() {
		codec, err := tokenizer.Get(tokenizer.O200kBase)
		if err == nil {
			contentCodec = codec
		}
	})
	return contentCodec
}

// fallbackTokens 在编码器不可用或粒度不够时补充基础分词结果。
func fallbackTokens(text string) []string {
	result := make([]string, 0, 16)
	segment := strings.Builder{}
	flushASCII := func() {
		if segment.Len() == 0 {
			return
		}
		result = append(result, segment.String())
		segment.Reset()
	}
	for _, item := range text {
		switch {
		case isCJK(item):
			flushASCII()
			result = append(result, string(item))
		case unicode.IsLetter(item) || unicode.IsDigit(item):
			segment.WriteRune(item)
		default:
			flushASCII()
		}
	}
	flushASCII()
	if utf8.RuneCountInString(text) <= 1 {
		return result
	}
	runeList := []rune(text)
	for index := 0; index < len(runeList)-1; index++ {
		left := runeList[index]
		right := runeList[index+1]
		if !isCJK(left) || !isCJK(right) {
			continue
		}
		result = append(result, string([]rune{left, right}))
	}
	return result
}

// sanitizeToken 清洗 token，只保留字母、数字和中文字符。
func sanitizeToken(token string) string {
	var builder strings.Builder
	for _, item := range token {
		if unicode.IsLetter(item) || unicode.IsDigit(item) || isCJK(item) {
			builder.WriteRune(item)
		}
	}
	return strings.TrimSpace(builder.String())
}

// normalizeVector 对内容向量做 L2 归一化。
func normalizeVector(values []float32) {
	sum := 0.0
	for _, item := range values {
		sum += float64(item * item)
	}
	if sum == 0 {
		return
	}
	norm := float32(math.Sqrt(sum))
	for index := range values {
		values[index] /= norm
	}
}

// truncateRunes 截断过长文本，避免详情字段放大计算成本。
func truncateRunes(text string, limit int) string {
	if limit <= 0 || utf8.RuneCountInString(text) <= limit {
		return text
	}
	runeList := []rune(text)
	return string(runeList[:limit])
}

// bucketPrice 返回价格分桶。
func bucketPrice(price int64) int {
	switch {
	case price <= 0:
		return 0
	case price < 2000:
		return 1
	case price < 5000:
		return 2
	case price < 10000:
		return 3
	default:
		return 4
	}
}

// bucketSale 返回销量分桶。
func bucketSale(saleNum int64) int {
	switch {
	case saleNum <= 0:
		return 0
	case saleNum < 10:
		return 1
	case saleNum < 50:
		return 2
	case saleNum < 200:
		return 3
	default:
		return 4
	}
}

// bucketAge 返回商品年龄分桶。
func bucketAge(createdAt time.Time, updatedAt time.Time) int {
	baseTime := createdAt
	if updatedAt.After(baseTime) {
		baseTime = updatedAt
	}
	if baseTime.IsZero() {
		return 0
	}
	ageDays := int(time.Since(baseTime).Hours() / 24)
	switch {
	case ageDays <= 0:
		return 0
	case ageDays <= 7:
		return 1
	case ageDays <= 30:
		return 2
	case ageDays <= 90:
		return 3
	default:
		return 4
	}
}

// isCJK 判断 rune 是否属于常见中文字符区间。
func isCJK(item rune) bool {
	return unicode.Is(unicode.Han, item)
}
