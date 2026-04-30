import { http } from '@/utils/http'
import type {
  GoodsCategory,
  GoodsCategoryService,
  ListGoodsCategoriesRequest,
  ListGoodsCategoriesResponse,
} from '@/rpc/app/v1/goods_category'

const GOODS_CATEGORY_URL = '/v1/app/goods/category'

type ListGoodsCategoriesResponseCompat = ListGoodsCategoriesResponse & {
  list: GoodsCategory[]
}

type ListGoodsCategoriesHTTPResponse = Partial<ListGoodsCategoriesResponse> & {
  list?: GoodsCategory[]
}

export class GoodsCategoryServiceImpl implements GoodsCategoryService {
  async ListGoodsCategories(
    request: ListGoodsCategoriesRequest,
  ): Promise<ListGoodsCategoriesResponseCompat> {
    const response = await http<ListGoodsCategoriesHTTPResponse>({
      url: `${GOODS_CATEGORY_URL}`,
      method: 'GET',
      data: request,
    })
    // 兼容未生成前的旧响应 list，同时向新协议的 goodsCategories 字段收敛。
    const goodsCategories = response.goods_categories ?? response.list ?? []
    return {
      ...response,
      goods_categories: goodsCategories,
      list: goodsCategories,
    }
  }
}

export const defGoodsCategoryService = new GoodsCategoryServiceImpl()
