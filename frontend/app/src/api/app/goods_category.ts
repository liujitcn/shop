import { http } from '@/utils/http'
import type {
  GoodsCategoryService,
  ListGoodsCategoriesRequest,
  ListGoodsCategoriesResponse,
} from '@/rpc/app/v1/goods_category'

const GOODS_CATEGORY_URL = '/v1/app/goods/category'

export class GoodsCategoryServiceImpl implements GoodsCategoryService {
  async ListGoodsCategories(
    request: ListGoodsCategoriesRequest,
  ): Promise<ListGoodsCategoriesResponse> {
    const response = await http<Partial<ListGoodsCategoriesResponse>>({
      url: `${GOODS_CATEGORY_URL}`,
      method: 'GET',
      authMode: 'none',
      data: request,
    })
    return {
      ...response,
      goods_categories: response.goods_categories ?? [],
    }
  }
}

export const defGoodsCategoryService = new GoodsCategoryServiceImpl()
