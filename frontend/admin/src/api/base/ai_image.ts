import service from "@/utils/request";
import type {
  AiImageService,
  AiImageTask,
  CreateAiImageTaskRequest,
  GetAiImageTaskRequest,
  PageAiImageTasksRequest,
  PageAiImageTasksResponse,
  PolishAiImagePromptRequest,
  PolishAiImagePromptResponse,
  RetryAiImageTaskRequest
} from "@/rpc/base/v1/ai_image";

const AI_IMAGE_TASK_URL = "/v1/base/ai/image/task";
const AI_IMAGE_PROMPT_POLISH_URL = "/v1/base/ai/image/prompt/polish";

/** AI 图片公共服务。 */
export class AiImageServiceImpl implements AiImageService {
  /** 分页查询 AI 图片。 */
  PageAiImageTasks(request: PageAiImageTasksRequest): Promise<PageAiImageTasksResponse> {
    return service<PageAiImageTasksRequest, PageAiImageTasksResponse>({
      url: AI_IMAGE_TASK_URL,
      method: "get",
      params: request
    });
  }

  /** 查询 AI 图片。 */
  GetAiImageTask(request: GetAiImageTaskRequest): Promise<AiImageTask> {
    return service<GetAiImageTaskRequest, AiImageTask>({
      url: `${AI_IMAGE_TASK_URL}/${request.id}`,
      method: "get"
    });
  }

  /** 创建 AI 图片。 */
  CreateAiImageTask(request: CreateAiImageTaskRequest): Promise<AiImageTask> {
    return service<CreateAiImageTaskRequest, AiImageTask>({
      url: AI_IMAGE_TASK_URL,
      method: "post",
      data: request
    });
  }

  /** 重试 AI 图片生成。 */
  RetryAiImageTask(request: RetryAiImageTaskRequest): Promise<AiImageTask> {
    return service<RetryAiImageTaskRequest, AiImageTask>({
      url: `${AI_IMAGE_TASK_URL}/${request.id}/retry`,
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
