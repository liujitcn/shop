import service from "@/utils/request";
import type {
  AiImageService,
  GenerateAiImageRequest,
  GenerateAiImageResponse,
  PolishAiImagePromptRequest,
  PolishAiImagePromptResponse
} from "@/rpc/base/v1/ai_image";

const AI_IMAGE_URL = "/v1/base/ai/image/generation";
const AI_IMAGE_PROMPT_POLISH_URL = "/v1/base/ai/image/prompt/polish";

/** AI 图片生成提交参数。 */
export type GenerateAiImagePayload = Pick<
  GenerateAiImageRequest,
  "prompt" | "model" | "size" | "quality" | "background" | "output_format" | "n" | "save_output" | "polish_prompt"
>;

/** AI 图片公共服务。 */
export class AiImageServiceImpl implements AiImageService {
  /** 生成 AI 图片。 */
  GenerateAiImage(request: GenerateAiImagePayload): Promise<GenerateAiImageResponse> {
    // 图片生成耗时不稳定，前端不主动用固定时长截断请求。
    return service<GenerateAiImagePayload, GenerateAiImageResponse>({
      url: AI_IMAGE_URL,
      method: "post",
      data: request,
      timeout: 0
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
