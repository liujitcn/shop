import service from "@/utils/request";
import {
  type PageRecommendModelVersionRequest,
  type PageRecommendModelVersionResponse,
  type RecommendModelVersionService,
  type UpdateRecommendModelVersionPublishRequest,
  type UpdateRecommendModelVersionPublishResponse
} from "@/rpc/admin/recommend_model_version";

const RECOMMEND_MODEL_VERSION_URL = "/admin/recommend/model-version";

/** 推荐版本管理服务 */
export class RecommendModelVersionServiceImpl implements RecommendModelVersionService {
  /** 查询推荐版本分页列表 */
  PageRecommendModelVersion(request: PageRecommendModelVersionRequest): Promise<PageRecommendModelVersionResponse> {
    return service<PageRecommendModelVersionRequest, PageRecommendModelVersionResponse>({
      url: `${RECOMMEND_MODEL_VERSION_URL}`,
      method: "get",
      params: request
    });
  }

  /** 发布推荐版本 */
  PublishRecommendModelVersion(
    request: UpdateRecommendModelVersionPublishRequest
  ): Promise<UpdateRecommendModelVersionPublishResponse> {
    return service<UpdateRecommendModelVersionPublishRequest, UpdateRecommendModelVersionPublishResponse>({
      url: `${RECOMMEND_MODEL_VERSION_URL}/${request.id}/publish`,
      method: "put",
      data: request
    });
  }
}

export const defRecommendModelVersionService = new RecommendModelVersionServiceImpl();
