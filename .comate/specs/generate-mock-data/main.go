package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// 连接数据库获取商品信息
	db, err := sql.Open("mysql", "root:112233@tcp(127.0.0.1:3306)/shop_test?charset=utf8mb4")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// 查询已上架商品
	rows, err := db.Query("SELECT id, name, price, discount_price, picture FROM goods WHERE status = 1")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	type Goods struct {
		ID           int64
		Name         string
		Price        int64
		DiscountPrice int64
		Picture      string
	}

	var goodsList []Goods
	for rows.Next() {
		var g Goods
		var discountPrice sql.NullInt64
		err := rows.Scan(&g.ID, &g.Name, &g.Price, &discountPrice, &g.Picture)
		if err != nil {
			continue
		}
		if discountPrice.Valid && discountPrice.Int64 > 0 {
			g.DiscountPrice = discountPrice.Int64
		} else {
			g.DiscountPrice = g.Price
		}
		goodsList = append(goodsList, g)
	}

	if len(goodsList) == 0 {
		panic("没有找到商品数据")
	}

	fmt.Printf("读取到 %d 个商品\n", len(goodsList))

	// 配置参数
	startDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	endDate := time.Date(2026, 4, 2, 0, 0, 0, 0, time.Local)
	totalUsers := 5000
	dailyOrders := 100

	// 订单状态分布
	statusWeights := []int{10, 15, 20, 35, 10, 8, 2} // CREATED, PAID, SHIPPED, RECEIVED, REFUNDING, CANCELED, DELETED

	// 打开输出文件
	file, err := os.Create("generate_mock_data.sql")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// 生成用户数据
	fmt.Fprintln(file, "-- 用户数据 (base_user)")
	userCount := 2 // 从2开始，保留id=1和id=2的系统用户
	days := int(endDate.Sub(startDate).Hours() / 24)
	usersPerDay := (totalUsers - 2) / days

	for day := 0; day < days; day++ {
		date := startDate.AddDate(0, 0, day)
		if day == days-1 {
			usersPerDay = totalUsers - 2 - userCount
		}
		for i := 0; i < usersPerDay; i++ {
			userCount++
			userID := userCount
			userName := fmt.Sprintf("user_%04d", userID)
			nickName := fmt.Sprintf("用户_%04d", userID)
			openid := fmt.Sprintf("o%d_%08d", rand.Intn(10), rand.Intn(100000000))
			roleID := int64(rand.Intn(5) + 1)
			deptID := int64(rand.Intn(10) + 1)
			phone := fmt.Sprintf("1%010d", rand.Intn(10000000000))
			password := "$2a$10$1234567890123456789012"
			gender := []int32{1, 2, 3}[rand.Intn(3)]
			avatar := fmt.Sprintf("https://example.com/avatar/%d.jpg", userID)
			status := int32(1)

			fmt.Fprintf(file, "INSERT INTO base_user (id, user_name, nick_name, openid, role_id, dept_id, phone, password, gender, avatar, status, created_at, updated_at) VALUES (%d, '%s', '%s', '%s', %d, %d, '%s', '%s', %d, '%s', %d, '%s', '%s');\n",
				userID, userName, nickName, openid, roleID, deptID, phone, password, gender, avatar, status, date.Format("2006-01-02 15:04:05"), date.Format("2006-01-02 15:04:05"))
		}
	}
	fmt.Fprintf(file, "\n-- 共生成 %d 个用户\n\n", userCount)

	// 生成订单数据
	fmt.Fprintln(file, "-- 订单数据 (order)")
	orderCount := 0
	orderStatusMap := make(map[int][]int)
	orderIDStart := 10001

	// 存储订单的商品信息用于生成order_goods
	type OrderGoodsInfo struct {
		OrderID   int64
		GoodsID   int64
		Name      string
		Picture   string
		Num       int64
		Price     int64
		PayPrice  int64
	}

	var allOrderGoods []OrderGoodsInfo

	for day := 0; day < days; day++ {
		date := startDate.AddDate(0, 0, day)
		for i := 0; i < dailyOrders; i++ {
			orderCount++
			orderID := int64(orderIDStart + orderCount - 1)
			orderNo := fmt.Sprintf("O%s%06d", date.Format("20060102"), orderID)
			userID := int64(rand.Intn(4998) + 3)

			// 随机选择1-3个商品，计算订单金额
			goodsNum := rand.Intn(3) + 1
			var totalMoney int64
			var orderGoodsList []OrderGoodsInfo

			for j := 0; j < goodsNum; j++ {
				goods := goodsList[rand.Intn(len(goodsList))]
				num := int64(rand.Intn(5) + 1)
				payPrice := goods.DiscountPrice
				totalPayPrice := payPrice * num
				totalMoney += totalPayPrice

				orderGoodsList = append(orderGoodsList, OrderGoodsInfo{
					OrderID: orderID,
					GoodsID: goods.ID,
					Name:    goods.Name,
					Picture: goods.Picture,
					Num:     num,
					Price:   goods.Price,
					PayPrice: payPrice,
				})
			}

			// 随机优惠（最低1%，最高15%）
			discountPercent := rand.Intn(15) + 1
			discount := totalMoney * int64(discountPercent) / 100
			payMoney := totalMoney - discount

			payType := int32(rand.Intn(2) + 1)
			payChannel := int32(rand.Intn(2) + 1)
			deliveryTime := int32(rand.Intn(3) + 1)

			status := getRandomStatus(statusWeights)
			createdAt := date.Add(time.Duration(rand.Intn(24)) * time.Hour).Add(time.Duration(rand.Intn(60)) * time.Minute)
			updatedAt := createdAt

			var deletedAt string
			if status == 99 {
				deletedAt = fmt.Sprintf("'%s'", createdAt.Add(time.Duration(rand.Intn(48)+1)*time.Hour).Format("2006-01-02 15:04:05"))
			} else {
				deletedAt = "NULL"
			}

			fmt.Fprintf(file, "INSERT INTO `order` (id, order_no, user_id, pay_money, total_money, goods_num, pay_type, pay_channel, delivery_time, status, created_at, updated_at, deleted_at) VALUES (%d, '%s', %d, %d, %d, %d, %d, %d, %d, %d, '%s', '%s', %s);\n",
				orderID, orderNo, userID, payMoney, totalMoney, goodsNum, payType, payChannel, deliveryTime, status, createdAt.Format("2006-01-02 15:04:05"), updatedAt.Format("2006-01-02 15:04:05"), deletedAt)

			orderStatusMap[status] = append(orderStatusMap[status], int(orderID))
			allOrderGoods = append(allOrderGoods, orderGoodsList...)
		}
	}
	fmt.Fprintf(file, "\n-- 共生成 %d 个订单\n\n", orderCount)

	// 生成订单商品数据 (order_goods)
	fmt.Fprintln(file, "-- 订单商品数据 (order_goods)")
	goodsCount := 0
	for _, og := range allOrderGoods {
		goodsCount++
		skuCode := fmt.Sprintf("SKU%08d", rand.Intn(100000))
		specItem := `{"颜色":"随机","尺寸":"随机"}`

		totalPrice := og.Price * og.Num
		totalPayPrice := og.PayPrice * og.Num

		fmt.Fprintf(file, "INSERT INTO order_goods (id, order_id, goods_id, sku_code, spec_item, picture, name, num, price, pay_price, total_price, total_pay_price) VALUES (%d, %d, %d, '%s', '%s', '%s', '%s', %d, %d, %d, %d, %d);\n",
			goodsCount, og.OrderID, og.GoodsID, skuCode, specItem, og.Picture, og.Name, og.Num, og.Price, og.PayPrice, totalPrice, totalPayPrice)
	}
	fmt.Fprintf(file, "\n-- 共生成 %d 个订单商品\n\n", goodsCount)

	// 生成订单支付数据 (order_payment)
	fmt.Fprintln(file, "-- 订单支付数据 (order_payment)")
	paymentCount := 0
	paidStatuses := []int{2, 3, 4, 97}
	for _, status := range paidStatuses {
		for _, orderID := range orderStatusMap[status] {
			paymentCount++
			orderNo := fmt.Sprintf("O20260101%06d", orderID)
			thirdOrderNo := fmt.Sprintf("T%d%012d", rand.Intn(10), rand.Intn(100000000000))
			tradeType := []string{"JSAPI", "NATIVE", "APP"}[rand.Intn(3)]
			tradeState := "SUCCESS"
			tradeStateDesc := "支付成功"
			bankType := []string{"OTHERS", "CFT", "BCM"}[rand.Intn(3)]
			successTime := time.Now().Add(-time.Duration(rand.Intn(30*24)) * time.Hour)
			payer := `{"openid":"o123456"}`
			amount := `{"total_fee":100,"currency":"CNY"}`
			sceneInfo := `{"payer_client_ip":"127.0.0.1"}`
			payStatus := int32(1)

			fmt.Fprintf(file, "INSERT INTO order_payment (id, order_id, order_no, third_order_no, trade_type, trade_state, trade_state_desc, bank_type, success_time, payer, amount, scene_info, status) VALUES (%d, %d, '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', %d);\n",
				paymentCount, orderID, orderNo, thirdOrderNo, tradeType, tradeState, tradeStateDesc, bankType, successTime.Format("2006-01-02 15:04:05"), payer, amount, sceneInfo, payStatus)
		}
	}
	fmt.Fprintf(file, "\n-- 共生成 %d 个订单支付记录\n\n", paymentCount)

	// 生成订单地址数据 (order_address)
	fmt.Fprintln(file, "-- 订单地址数据 (order_address)")
	addressCount := 0
	provinces := []string{"北京市", "上海市", "广东省", "浙江省", "江苏省"}
	cities := map[string][]string{
		"北京市": {"北京市"},
		"上海市": {"上海市"},
		"广东省": {"广州市", "深圳市", "东莞市"},
		"浙江省": {"杭州市", "宁波市", "温州市"},
		"江苏省": {"南京市", "苏州市", "无锡市"},
	}
	districts := []string{"朝阳区", "海淀区", "浦东新区", "徐汇区", "天河区", "南山区"}

	for i := 1; i <= orderCount; i++ {
		orderID := orderIDStart + i - 1
		addressCount++
		province := provinces[rand.Intn(len(provinces))]
		city := cities[province][rand.Intn(len(cities[province]))]
		district := districts[rand.Intn(len(districts))]
		detail := fmt.Sprintf("xx路%d号", rand.Intn(999)+1)
		receiver := fmt.Sprintf("用户%d", rand.Intn(10000))
		contact := fmt.Sprintf("1%010d", rand.Intn(10000000000))
		address := province + city + district

		fmt.Fprintf(file, "INSERT INTO order_address (id, order_id, receiver, contact, address, detail) VALUES (%d, %d, '%s', '%s', '%s', '%s');\n",
			addressCount, orderID, receiver, contact, address, detail)
	}
	fmt.Fprintf(file, "\n-- 共生成 %d 个订单地址\n\n", addressCount)

	// 生成订单退款数据 (order_refund)
	fmt.Fprintln(file, "-- 订单退款数据 (order_refund)")
	refundCount := 0
	for _, orderID := range orderStatusMap[97] {
		refundCount++
		orderNo := fmt.Sprintf("O20260101%06d", orderID)
		thirdOrderNo := fmt.Sprintf("T%d%012d", rand.Intn(10), rand.Intn(100000000000))
		refundNo := fmt.Sprintf("R%d%012d", rand.Intn(10), rand.Intn(100000000000))
		reason := int32(rand.Intn(6) + 1)
		thirdRefundNo := fmt.Sprintf("RF%d%012d", rand.Intn(10), rand.Intn(100000000000))
		channel := "ORIGINAL"
		userReceivedAccount := "支付用户零钱"
		createTime := time.Now().Add(-time.Duration(rand.Intn(30*24)) * time.Hour)
		successTime := createTime.Add(time.Duration(rand.Intn(24)) * time.Hour)
		refundState := "SUCCESS"
		fundsAccount := "AVAILABLE"
		amount := `{"refund_fee":100,"settle_refund_fee":100}`
		refundStatus := int32(1)

		fmt.Fprintf(file, "INSERT INTO order_refund (id, order_id, order_no, third_order_no, refund_no, reason, third_refund_no, channel, user_received_account, create_time, success_time, refund_state, funds_account, amount, status) VALUES (%d, %d, '%s', '%s', '%s', %d, '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', %d);\n",
			refundCount, orderID, orderNo, thirdOrderNo, refundNo, reason, thirdRefundNo, channel, userReceivedAccount, createTime.Format("2006-01-02 15:04:05"), successTime.Format("2006-01-02 15:04:05"), refundState, fundsAccount, amount, refundStatus)
	}
	fmt.Fprintf(file, "\n-- 共生成 %d 个订单退款记录\n\n", refundCount)

	// 生成订单物流数据 (order_logistics)
	fmt.Fprintln(file, "-- 订单物流数据 (order_logistics)")
	logisticsCount := 0
	shippingStatuses := []int{3, 4}
	companies := []string{"顺丰速运", "圆通速递", "中通快递", "韵达快递", "申通快递"}
	for _, status := range shippingStatuses {
		for _, orderID := range orderStatusMap[status] {
			logisticsCount++
			company := companies[rand.Intn(len(companies))]
			no := fmt.Sprintf("SF%d%012d", rand.Intn(10), rand.Intn(100000000000))
			contact := fmt.Sprintf("400-%d", rand.Intn(9000)+1000)
			detail := `{"status":"运输中","location":"深圳市","time":"2026-01-02 10:00:00"}`
			createdAt := time.Now().Add(-time.Duration(rand.Intn(30*24)) * time.Hour)

			fmt.Fprintf(file, "INSERT INTO order_logistics (id, order_id, name, no, contact, detail, created_at) VALUES (%d, %d, '%s', '%s', '%s', '%s', '%s');\n",
				logisticsCount, orderID, company, no, contact, detail, createdAt.Format("2006-01-02 15:04:05"))
		}
	}
	fmt.Fprintf(file, "\n-- 共生成 %d 个订单物流记录\n\n", logisticsCount)

	// 生成订单取消数据 (order_cancel)
	fmt.Fprintln(file, "-- 订单取消数据 (order_cancel)")
	cancelCount := 0
	for _, orderID := range orderStatusMap[98] {
		cancelCount++
		reason := int32(rand.Intn(6) + 1)
		createdAt := time.Now().Add(-time.Duration(rand.Intn(30*24)) * time.Hour)

		fmt.Fprintf(file, "INSERT INTO order_cancel (id, order_id, reason, created_at) VALUES (%d, %d, %d, '%s');\n",
			cancelCount, orderID, reason, createdAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Fprintf(file, "\n-- 共生成 %d 个订单取消记录\n\n", cancelCount)

	// 统计
	fmt.Fprintln(file, "-- 订单状态统计")
	fmt.Fprintf(file, "-- CREATED(待付款): %d\n", len(orderStatusMap[1]))
	fmt.Fprintf(file, "-- PAID(待发货): %d\n", len(orderStatusMap[2]))
	fmt.Fprintf(file, "-- SHIPPED(待收货): %d\n", len(orderStatusMap[3]))
	fmt.Fprintf(file, "-- RECEIVED(已完成): %d\n", len(orderStatusMap[4]))
	fmt.Fprintf(file, "-- REFUNDING(已退款): %d\n", len(orderStatusMap[97]))
	fmt.Fprintf(file, "-- CANCELED(已取消): %d\n", len(orderStatusMap[98]))
	fmt.Fprintf(file, "-- DELETED(已删除): %d\n", len(orderStatusMap[99]))

	fmt.Fprintln(file, "\n-- 数据生成完成!")
	fmt.Println("数据生成完成！共生成:")
	fmt.Printf("  - %d 个用户\n", userCount)
	fmt.Printf("  - %d 个订单\n", orderCount)
	fmt.Printf("  - %d 个订单商品\n", goodsCount)
	fmt.Printf("  - %d 个订单支付记录\n", paymentCount)
	fmt.Printf("  - %d 个订单地址\n", addressCount)
	fmt.Printf("  - %d 个订单退款记录\n", refundCount)
	fmt.Printf("  - %d 个订单物流记录\n", logisticsCount)
	fmt.Printf("  - %d 个订单取消记录\n", cancelCount)
}

func getRandomStatus(weights []int) int {
	statuses := []int{1, 2, 3, 4, 97, 98, 99}
	total := 0
	for _, w := range weights {
		total += w
	}
	r := rand.Intn(total)
	sum := 0
	for i, w := range weights {
		sum += w
		if r < sum {
			return statuses[i]
		}
	}
	return statuses[0]
}