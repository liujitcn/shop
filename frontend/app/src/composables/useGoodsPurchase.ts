import type { SkuPopupEvent, SkuPopupLocalData } from '@/components/goods-sku-popup/goods-sku-popup'
import { defGoodsInfoService } from '@/api/app/goods_info'
import { defUserCartService } from '@/api/app/user_cart'
import { defUserCollectService } from '@/api/app/user_collect'
import type { GoodsInfoResponse } from '@/rpc/app/v1/goods_info'
import type { RecommendContext } from '@/rpc/app/v1/recommend'
import type { RecommendScene } from '@/rpc/common/v1/enum'
import { useUserStore } from '@/stores'
import { navigateToLogin, navigateToOrderCreate } from '@/utils/navigation'
import { ref } from 'vue'

export enum SkuMode {
  Both = 1,
  Cart = 2,
  Buy = 3,
}

type GoodsPurchaseTarget = {
  id: number
}

type GoodsOrderRouteQuery = {
  scene?: RecommendScene
  request_id?: string | number
  index?: string | number
}

type OpenSkuOptions = {
  recommendContext?: RecommendContext
  orderRouteQuery?: GoodsOrderRouteQuery
}

type UseGoodsPurchaseOptions<T extends GoodsPurchaseTarget> = {
  ensureGoodsDetail?: (item: T) => Promise<GoodsInfoResponse | undefined>
  getRecommendContext?: (item: T) => RecommendContext | undefined
  getOrderRouteQuery?: (item: T) => GoodsOrderRouteQuery | undefined
}

const buildSkuLocalData = (goods: GoodsInfoResponse): SkuPopupLocalData => {
  return {
    _id: goods.id,
    name: goods.name,
    goods_thumb: goods.picture,
    spec_list: goods.spec_list.map((item) => ({
      name: item.name,
      list: item.item,
    })),
    sku_list: goods.sku_list.map((item) => ({
      _id: item.sku_code,
      goods_id: goods.id,
      goods_name: goods.name,
      image: item.picture,
      price: item.price,
      stock: item.inventory,
      sku_name_arr: item.spec_item,
    })),
  }
}

const defaultEnsureGoodsDetail = async (item: GoodsPurchaseTarget) => {
  return defGoodsInfoService.GetGoodsInfo({ id: item.id })
}

export const useGoodsPurchase = <T extends GoodsPurchaseTarget>(
  options: UseGoodsPurchaseOptions<T> = {},
) => {
  const userStore = useUserStore()
  const cartNum = ref(0)
  const buyingGoodsId = ref(0)
  const isShowSku = ref(false)
  const skuMode = ref<SkuMode>(SkuMode.Cart)
  const localData = ref({} as SkuPopupLocalData)
  const collectMap = ref<Record<number, boolean>>({})
  const currentRecommendContext = ref<RecommendContext>()
  const currentOrderRouteQuery = ref<GoodsOrderRouteQuery>({})

  const ensureAuthenticated = () => {
    if (userStore.ensureAuthenticated()) {
      return true
    }
    navigateToLogin()
    return false
  }

  const resolveRecommendContext = (item: T, recommendContext?: RecommendContext) => {
    return recommendContext ?? options.getRecommendContext?.(item)
  }

  const resolveOrderRouteQuery = (item: T, orderRouteQuery?: GoodsOrderRouteQuery) => {
    return orderRouteQuery ?? options.getOrderRouteQuery?.(item) ?? {}
  }

  const isCollected = (item: T | number | undefined) => {
    const id = typeof item === 'number' ? item : item?.id
    return id ? collectMap.value[id] === true : false
  }

  const refreshCartNum = async () => {
    if (!userStore.isAuthenticated()) return
    try {
      const res = await defUserCartService.CountUserCart({})
      cartNum.value = res.count
    } catch (error) {
      console.error(error)
    }
  }

  const refreshCollectState = async (item: T | undefined) => {
    if (!item || !userStore.isAuthenticated()) return
    try {
      const res = await defUserCollectService.GetIsCollect({ goods_id: item.id })
      collectMap.value = {
        ...collectMap.value,
        [item.id]: res.is_collected,
      }
    } catch (error) {
      console.error(error)
    }
  }

  const refreshGoodsState = async (item: T | undefined) => {
    if (!item || !userStore.isAuthenticated()) return
    try {
      const [cartRes, collectRes] = await Promise.all([
        defUserCartService.CountUserCart({}),
        defUserCollectService.GetIsCollect({ goods_id: item.id }),
      ])
      cartNum.value = cartRes.count
      collectMap.value = {
        ...collectMap.value,
        [item.id]: collectRes.is_collected,
      }
    } catch (error) {
      console.error(error)
    }
  }

  const openSkuPopup = async (
    item: T | undefined,
    mode: SkuMode,
    openOptions: OpenSkuOptions = {},
  ) => {
    if (!item) return
    buyingGoodsId.value = item.id
    try {
      const ensureGoodsDetail = options.ensureGoodsDetail ?? defaultEnsureGoodsDetail
      const goodsDetail = await ensureGoodsDetail(item)
      if (!goodsDetail) {
        uni.showToast({ icon: 'none', title: '商品规格加载失败' })
        return
      }
      if (!goodsDetail.sku_list.length) {
        uni.showToast({ icon: 'none', title: '当前商品暂无可售规格' })
        return
      }
      localData.value = buildSkuLocalData(goodsDetail)
      skuMode.value = mode
      currentRecommendContext.value = resolveRecommendContext(item, openOptions.recommendContext)
      currentOrderRouteQuery.value = resolveOrderRouteQuery(item, openOptions.orderRouteQuery)
      isShowSku.value = true
    } catch (error) {
      console.error(error)
      uni.showToast({ icon: 'none', title: '商品规格加载失败' })
    } finally {
      buyingGoodsId.value = 0
    }
  }

  const toggleCollect = async (item: T | undefined, recommendContext?: RecommendContext) => {
    if (!item) return
    if (!ensureAuthenticated()) return
    await defUserCollectService.CreateUserCollect({
      goods_id: item.id,
      recommend_context: resolveRecommendContext(item, recommendContext),
    })
    collectMap.value = {
      ...collectMap.value,
      [item.id]: !collectMap.value[item.id],
    }
    await uni.showToast({ title: collectMap.value[item.id] ? '收藏成功' : '取消成功' })
  }

  const addCart = async (event: SkuPopupEvent) => {
    if (!ensureAuthenticated()) return
    await defUserCartService.CreateUserCart({
      goods_id: event.goods_id,
      sku_code: event._id,
      num: event.buy_num,
      recommend_context: currentRecommendContext.value,
    })
    await refreshCartNum()
    await uni.showToast({ title: '添加成功' })
    isShowSku.value = false
  }

  const buyNow = (event: SkuPopupEvent) => {
    if (!ensureAuthenticated()) return
    isShowSku.value = false
    void navigateToOrderCreate({
      goods_id: event.goods_id,
      sku_code: event._id,
      num: event.buy_num,
      ...currentOrderRouteQuery.value,
    })
  }

  return {
    addCart,
    buyNow,
    buyingGoodsId,
    cartNum,
    collectMap,
    isCollected,
    isShowSku,
    localData,
    openSkuPopup,
    refreshCartNum,
    refreshCollectState,
    refreshGoodsState,
    skuMode,
    toggleCollect,
  }
}
