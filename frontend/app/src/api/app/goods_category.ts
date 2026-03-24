import { http } from '@/utils/http'
import type {
  GoodsCategoryService,
  ListGoodsCategoryRequest,
  ListGoodsCategoryResponse,
} from '@/rpc/app/goods_category'

const GOODS_CATEGORY_URL = '/app/goods/category'

export class GoodsCategoryServiceImpl implements GoodsCategoryService {
  ListGoodsCategory(request: ListGoodsCategoryRequest): Promise<ListGoodsCategoryResponse> {
    return http<ListGoodsCategoryResponse>({
      url: `${GOODS_CATEGORY_URL}`,
      method: 'GET',
      data: request,
    })
  }
}

export const defGoodsCategoryService = new GoodsCategoryServiceImpl()
