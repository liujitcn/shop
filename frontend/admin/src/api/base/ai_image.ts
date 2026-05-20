import service from "@/utils/request";
import type {
  AiImage,
  AiImageService,
  CreateAiImageRequest,
  DeleteAiImageRequest,
  GetAiImageRequest,
  PageAiImagesRequest,
  PageAiImagesResponse,
  PolishAiImagePromptRequest,
  PolishAiImagePromptResponse,
  RetryAiImageRequest
} from "@/rpc/base/v1/ai_image";
import type { Empty } from "@/rpc/google/protobuf/empty";

const AI_IMAGE_URL = "/v1/base/ai/image";
const AI_IMAGE_PROMPT_POLISH_URL = "/v1/base/ai/image/prompt/polish";

/** AI 图片公共服务。 */
export class AiImageServiceImpl implements AiImageService {
  /** 分页查询 AI 图片。 */
  PageAiImages(request: PageAiImagesRequest): Promise<PageAiImagesResponse> {
    return service<PageAiImagesRequest, PageAiImagesResponse>({
      url: AI_IMAGE_URL,
      method: "get",
      params: request
    });
  }

  /** 查询 AI 图片。 */
  GetAiImage(request: GetAiImageRequest): Promise<AiImage> {
    return service<GetAiImageRequest, AiImage>({
      url: `${AI_IMAGE_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 创建 AI 图片。 */
  CreateAiImage(request: CreateAiImageRequest): Promise<AiImage> {
    return service<CreateAiImageRequest, AiImage>({
      url: AI_IMAGE_URL,
      method: "post",
      data: request
    });
  }

  /** 删除 AI 图片。 */
  DeleteAiImage(request: DeleteAiImageRequest): Promise<Empty> {
    return service<DeleteAiImageRequest, Empty>({
      url: `${AI_IMAGE_URL}/${request.ids}`,
      method: "delete"
    });
  }

  /** 重试 AI 图片生成。 */
  RetryAiImage(request: RetryAiImageRequest): Promise<AiImage> {
    return service<RetryAiImageRequest, AiImage>({
      url: `${AI_IMAGE_URL}/${request.id}/retry`,
      method: "post",
      data: request
    });
  }

  /** 润色 AI 图片提示词。 */
  PolishAiImagePrompt(request: PolishAiImagePromptRequest): Promise<PolishAiImagePromptResponse> {
    return service<PolishAiImagePromptRequest, PolishAiImagePromptResponse>({
      url: AI_IMAGE_PROMPT_POLISH_URL,
      method: "post",
      data: request
    });
  }
}

export const defAiImageService = new AiImageServiceImpl();
