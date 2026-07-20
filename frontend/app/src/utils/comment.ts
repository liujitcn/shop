import { defCommentInfoService } from '@/api/shop/app/comment'
import { formatSrc } from './index'
import { orderCommentWriteUrl } from './navigation'

/** 查询门店订单首个未评价商品并打开评价发布页。 */
export async function openPendingOrderComment(orderID: number) {
  const response = await defCommentInfoService.PagePendingCommentGoods({
    order_id: orderID,
    page_num: 1,
    page_size: 1,
  })
  const goods = response.pending_comment_goods[0]
  if (!goods) {
    uni.showToast({ icon: 'none', title: '该订单暂无待评价商品' })
    return
  }
  await uni.navigateTo({
    url: orderCommentWriteUrl({
      order_id: goods.order_id,
      goods_id: goods.goods_id,
      goods_name: goods.goods_name,
      goods_picture: goods.goods_picture ? formatSrc(goods.goods_picture) : undefined,
      sku_code: goods.sku_code,
      sku_desc: goods.sku_desc,
    }),
  })
}
