import { RecommendScene } from '@/rpc/common/enum'

/** 路由 query 支持的值类型。 */
type QueryValue = string | number | boolean | null | undefined
/** 用于统一拼接页面 query 的键值对象。 */
type QueryRecord = Record<string, QueryValue>
/** 推荐路由上下文字段。 */
export type RecommendRouteQuery = {
  scene?: RecommendScene | string | number
  requestId?: string | number
  index?: string | number
}
/** 商品详情页支持的入参。 */
type GoodsDetailQuery = {
  id: string | number
  scene?: RecommendScene
  requestId?: string | number
  index?: string | number
}

/** 搜索结果页支持的入参。 */
type SearchQuery = {
  name?: string
  categoryId?: string | number
  categoryName?: string
}

/** 下单页支持的入参。 */
type OrderCreateQuery = {
  goodsId?: string | number
  skuCode?: string
  num?: string | number
  orderId?: string | number
  scene?: RecommendScene
  requestId?: string | number
  index?: string | number
}

/** 订单详情页支持的入参。 */
type OrderDetailQuery = {
  id: string | number
  internal?: boolean
}

/** 登录前记录当前路由的本地存储 key。 */
const LAST_ROUTE_KEY = 'lastRoute'
/** 登录页路径。 */
const LOGIN_PAGE = '/pages/login/login'
/** 首页 tab 页面路径。 */
const HOME_TAB_PAGE = '/pages/index/index'
/** 商品详情页路径。 */
const GOODS_PAGE = '/pages/goods/goods'
/** 搜索结果页路径。 */
const SEARCH_PAGE = '/pages/search/index'
/** 下单页路径。 */
const ORDER_CREATE_PAGE = '/pagesOrder/create/create'
/** 订单详情页路径。 */
const ORDER_DETAIL_PAGE = '/pagesOrder/detail/detail'
/** 订单列表页路径。 */
const ORDER_LIST_PAGE = '/pagesOrder/list/list'

/** 首页 tab 页面路径。 */
export const homeTabPage = HOME_TAB_PAGE

/** 判断 query 值是否应该参与 URL 拼接。 */
const isValidQueryValue = (value: QueryValue) => {
  return value !== undefined && value !== null && value !== ''
}

/** 统一把推荐场景序列化成路由参数。 */
const normalizeRecommendScene = (scene?: RecommendScene) => {
  if (scene === undefined || scene === null) {
    return undefined
  }
  return scene
}

/** 统一清洗推荐请求编号，避免把空值或非法值写入路由。 */
const normalizeRecommendRequestId = (requestId?: string | number) => {
  if (requestId === undefined || requestId === null || requestId === '') {
    return undefined
  }
  const value = Number(requestId)
  if (!Number.isFinite(value) || value <= 0) {
    return undefined
  }
  return value
}

/** 统一解析路由里的推荐上下文字段。 */
export const parseRecommendRouteQuery = (query: RecommendRouteQuery) => {
  const sceneValue =
    query.scene === undefined || query.scene === null || query.scene === ''
      ? undefined
      : Number(query.scene)
  const scene =
    sceneValue !== undefined && Number.isFinite(sceneValue)
      ? (sceneValue as RecommendScene)
      : undefined
  const indexValue =
    query.index === undefined || query.index === null || query.index === ''
      ? undefined
      : Number(query.index)
  const index = indexValue !== undefined && Number.isFinite(indexValue) ? indexValue : undefined

  return {
    scene: scene === RecommendScene.UNKNOWN_RS ? undefined : scene,
    requestId: normalizeRecommendRequestId(query.requestId),
    index,
  }
}

/** 统一清洗推荐相关的路由参数，避免页面侧重复处理默认值。 */
const normalizeRecommendRouteQuery = (query: RecommendRouteQuery) => {
  const { scene, requestId, index } = parseRecommendRouteQuery(query)
  const hasRecommendContext =
    scene !== undefined || requestId !== undefined || isValidQueryValue(index)

  return {
    scene: normalizeRecommendScene(scene),
    requestId,
    index: hasRecommendContext ? (isValidQueryValue(index) ? index : 0) : undefined,
  }
}

/** 把已有 query 字符串安全拼接回页面路径。 */
const withQueryString = (path: string, queryString?: string) => {
  if (!queryString) {
    return path
  }
  const normalizedQuery = queryString.replace(/^[?#&]+/, '')
  return normalizedQuery ? `${path}?${normalizedQuery}` : path
}

// URL 统一在这里编码，组件里不再手写 query 拼接。
/** 构建带 query 的页面 URL。 */
const buildPageUrl = (path: string, query: QueryRecord = {}) => {
  const queryString = Object.entries(query)
    .filter(([, value]) => isValidQueryValue(value))
    .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`)
    .join('&')

  return withQueryString(path, queryString)
}

/** 构建商品详情页 URL。 */
export const goodsDetailUrl = (query: GoodsDetailQuery | string | number) => {
  if (typeof query === 'string' || typeof query === 'number') {
    return buildPageUrl(GOODS_PAGE, { id: query })
  }
  return buildPageUrl(GOODS_PAGE, {
    ...query,
    ...normalizeRecommendRouteQuery(query),
  })
}

/** 构建下单页 URL。 */
export const orderCreateUrl = (query: OrderCreateQuery = {}) => {
  return buildPageUrl(ORDER_CREATE_PAGE, {
    ...query,
    ...normalizeRecommendRouteQuery(query),
  })
}

/** 构建订单详情页 URL。 */
export const orderDetailUrl = (query: OrderDetailQuery) => {
  return buildPageUrl(ORDER_DETAIL_PAGE, query)
}

/** 构建订单列表页 URL。 */
export const orderListUrl = (status: string | number) => {
  return buildPageUrl(ORDER_LIST_PAGE, { status })
}

/** 获取当前页面的完整路由，包含 query，供登录前回跳使用。 */
const getCurrentRouteUrl = () => {
  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1]
  if (!currentPage?.route) {
    return ''
  }

  let params: Record<string, string> = {}
  const miniPage = currentPage as { options?: Record<string, string> }
  const routePage = currentPage as { $vm?: { $route?: { query?: Record<string, string> } } }

  // 微信小程序通过页面实例 options 读取参数。
  // #ifdef MP-WEIXIN
  params = miniPage.options || {}
  // #endif

  // H5 和 APP 通过 Vue 路由读取参数。
  // #ifdef H5 || APP-PLUS
  if (routePage.$vm?.$route) {
    params = routePage.$vm.$route.query || {}
  }
  // #endif

  return buildPageUrl(`/${currentPage.route}`, params)
}

/** 保存当前页面路由，登录成功后恢复到用户原来的页面。 */
export const saveCurrentRoute = () => {
  const currentRoute = getCurrentRouteUrl()
  if (!currentRoute || currentRoute.startsWith(LOGIN_PAGE)) {
    uni.removeStorageSync(LAST_ROUTE_KEY)
    return
  }
  uni.setStorageSync(LAST_ROUTE_KEY, currentRoute)
}

/** 跳转到登录页，并在跳转前记录当前页面。 */
export const navigateToLogin = () => {
  saveCurrentRoute()
  uni.navigateTo({
    url: LOGIN_PAGE,
    fail: () => {
      uni.reLaunch({ url: LOGIN_PAGE })
    },
  })
}

/** 切换到首页 tab。 */
export const switchTabToHome = () => {
  return uni.switchTab({ url: homeTabPage })
}

/** 跳转到搜索页。 */
export const navigateToSearch = (query: SearchQuery = {}) => {
  return uni.navigateTo({ url: buildPageUrl(SEARCH_PAGE, query) })
}

/** 跳转到下单页。 */
export const navigateToOrderCreate = (query: OrderCreateQuery = {}) => {
  return uni.navigateTo({ url: orderCreateUrl(query) })
}

/** 重定向到支付结果页。 */
export const redirectToOrderPayment = (id: string | number) => {
  return uni.redirectTo({ url: `/pagesOrder/payment/payment?id=${encodeURIComponent(String(id))}` })
}
