import service from "@/utils/request";
import type {
  OptionCategoriesRequest,
  OptionCategoriesResponse,
  ConfigResponse,
  DeleteItemRequest,
  DeleteUserRequest,
  ListDashboardItemsRequest,
  ListDashboardItemsResponse,
  ExportDataRequest,
  ExportDataResponse,
  Feedback,
  FeedbackResponse,
  GetConfigRequest,
  GetItemRequest,
  GetUserRequest,
  ImportDataRequest,
  ImportDataResponse,
  ItemListResponse,
  ListTasksResponse,
  GetTimeSeriesRequest,
  Item,
  PageItemsRequest,
  PageItemsResponse,
  ListTasksRequest,
  PageUsersRequest,
  PageUsersResponse,
  PreviewExternalRequest,
  PreviewExternalResponse,
  PreviewRankerPromptRequest,
  PreviewRankerPromptResponse,
  RecommendGorseService,
  ResetConfigRequest,
  SaveConfigRequest,
  Task,
  GetUserRecommendRequest,
  GetItemSimilarRequest,
  GetUserSimilarRequest,
  TimeSeriesResponse,
  GetUserFeedbackRequest,
  UserResponse,
  UserSimilarResponse
} from "@/rpc/admin/v1/recommend_gorse";
import type { Empty } from "@/rpc/google/protobuf/empty";

const RECOMMEND_GORSE_URL = "/v1/admin/recommend/gorse";

/** 将未知值安全转换成普通对象。 */
function toRecord(value: unknown) {
  return typeof value === "object" && value !== null && !Array.isArray(value) ? (value as Record<string, unknown>) : {};
}

/** 按候选字段名读取对象值，兼容 protojson 与Gorse原始字段命名差异。 */
function readField(record: Record<string, unknown>, ...fieldNames: string[]) {
  for (const fieldName of fieldNames) {
    if (Object.prototype.hasOwnProperty.call(record, fieldName)) {
      return record[fieldName];
    }
  }
  return undefined;
}

/** 将未知值安全转换成字符串。 */
function toStringValue(value: unknown) {
  return typeof value === "string" ? value : value === null || value === undefined ? "" : String(value);
}

/** 将未知值安全转换成数字。 */
function toNumberValue(value: unknown) {
  return typeof value === "number" ? value : Number(value || 0);
}

/** 将未知值安全转换成布尔值。 */
function toBooleanValue(value: unknown) {
  if (typeof value === "boolean") return value;
  if (typeof value === "string") return value.toLowerCase() === "true";
  return Boolean(value);
}

/** 从接口响应中读取数组字段，兼容“对象包裹数组”与“直接返回数组”两种格式。 */
function toArrayField(value: unknown, ...fieldNames: string[]) {
  if (Array.isArray(value)) return value;
  const record = toRecord(value);
  for (const fieldName of fieldNames) {
    const fieldValue = readField(record, fieldName);
    if (Array.isArray(fieldValue)) return fieldValue;
  }
  return [];
}

/** 将原始商品标签规范化为前端使用结构。 */
function normalizeItemLabel(value: unknown): Item["labels"] {
  const record = toRecord(value);
  return {
    desc: toStringValue(readField(record, "desc")),
    discount_price: toNumberValue(readField(record, "discount_price")),
    inventory: toNumberValue(readField(record, "inventory")),
    price: toNumberValue(readField(record, "price")),
    status: toNumberValue(readField(record, "status"))
  };
}

/** 将原始商品数据规范化为前端使用结构。 */
function normalizeItem(value: unknown): Item {
  const record = toRecord(value);
  const categories = readField(record, "Categories", "categories");
  return {
    item_id: toStringValue(readField(record, "ItemId", "item_id")),
    is_hidden: toBooleanValue(readField(record, "IsHidden", "is_hidden")),
    categories: Array.isArray(categories) ? categories.map(item => toStringValue(item)) : [],
    timestamp: toStringValue(readField(record, "Timestamp", "timestamp")),
    labels: normalizeItemLabel(readField(record, "Labels", "labels")),
    comment: toStringValue(readField(record, "Comment", "comment")),
    score: toNumberValue(readField(record, "Score", "score"))
  };
}

/** 将原始用户标签规范化为前端使用结构。 */
function normalizeUserLabel(value: unknown): UserResponse["labels"] {
  const record = toRecord(value);
  return {
    dept_id: toNumberValue(readField(record, "dept_id")),
    gender: toNumberValue(readField(record, "gender")),
    role_id: toNumberValue(readField(record, "role_id")),
    status: toNumberValue(readField(record, "status"))
  };
}

/** 将原始用户数据规范化为前端使用结构。 */
function normalizeUser(value: unknown): UserResponse {
  const record = toRecord(value);
  return {
    user_id: toStringValue(readField(record, "UserId", "user_id")),
    labels: normalizeUserLabel(readField(record, "Labels", "labels")),
    comment: toStringValue(readField(record, "Comment", "comment")),
    last_active_time: toStringValue(readField(record, "LastActiveTime", "last_active_time")),
    last_update_time: toStringValue(readField(record, "LastUpdateTime", "last_update_time")),
    score: toNumberValue(readField(record, "Score", "score"))
  };
}

/** 将原始任务数据规范化为前端使用结构。 */
function normalizeTask(value: unknown): Task {
  const record = toRecord(value);
  return {
    tracer: toStringValue(readField(record, "Tracer", "tracer")),
    name: toStringValue(readField(record, "Name", "name")),
    status: toStringValue(readField(record, "Status", "status")),
    error: toStringValue(readField(record, "Error", "error")),
    count: toNumberValue(readField(record, "Count", "count")),
    total: toNumberValue(readField(record, "Total", "total")),
    start_time: toStringValue(readField(record, "StartTime", "start_time")),
    finish_time: toStringValue(readField(record, "FinishTime", "finish_time"))
  };
}

/** 将原始反馈数据规范化为前端使用结构。 */
function normalizeFeedback(value: unknown): Feedback {
  const record = toRecord(value);
  return {
    feedback_type: toStringValue(readField(record, "FeedbackType", "feedback_type")),
    user_id: toStringValue(readField(record, "UserId", "user_id")),
    item: normalizeItem(readField(record, "Item", "item")),
    value: toNumberValue(readField(record, "Value", "value")),
    timestamp: toStringValue(readField(record, "Timestamp", "timestamp")),
    comment: toStringValue(readField(record, "Comment", "comment"))
  };
}

/** Gorse 推荐管理服务。 */
export class RecommendGorseServiceImpl implements RecommendGorseService {
  /** 查询 Gorse 推荐时间序列。 */
  GetTimeSeries(request: GetTimeSeriesRequest): Promise<TimeSeriesResponse> {
    return service<GetTimeSeriesRequest, TimeSeriesResponse>({
      url: `${RECOMMEND_GORSE_URL}/timeseries/${encodeURIComponent(request.name)}`,
      method: "get",
      params: {
        begin: request.begin,
        end: request.end
      }
    });
  }

  /** 查询 Gorse 推荐分类。 */
  OptionCategories(request: OptionCategoriesRequest): Promise<OptionCategoriesResponse> {
    return service<OptionCategoriesRequest, OptionCategoriesResponse>({
      url: `${RECOMMEND_GORSE_URL}/category/option`,
      method: "get",
      params: request
    });
  }

  /** 查询 Gorse 推荐仪表盘推荐商品。 */
  async ListDashboardItems(request: ListDashboardItemsRequest): Promise<ListDashboardItemsResponse> {
    const data = await service<ListDashboardItemsRequest, unknown>({
      url: `${RECOMMEND_GORSE_URL}/dashboard`,
      method: "get",
      params: request
    });
    const record = toRecord(data);
    return {
      items: toArrayField(record, "Items", "items").map(normalizeItem),
      last_modified: toStringValue(readField(record, "LastModified", "last_modified"))
    };
  }

  /** 查询 Gorse 推荐配置。 */
  GetConfig(request: GetConfigRequest): Promise<ConfigResponse> {
    return service<GetConfigRequest, ConfigResponse>({
      url: `${RECOMMEND_GORSE_URL}/config`,
      method: "get",
      params: request
    });
  }

  /** 保存 Gorse 推荐配置。 */
  SaveConfig(request: SaveConfigRequest): Promise<ConfigResponse> {
    return service<ConfigResponse | undefined, ConfigResponse>({
      url: `${RECOMMEND_GORSE_URL}/config`,
      method: "post",
      data: request.config
    });
  }

  /** 重置 Gorse 推荐配置。 */
  ResetConfig(request: ResetConfigRequest): Promise<Empty> {
    return service<ResetConfigRequest, Empty>({
      url: `${RECOMMEND_GORSE_URL}/config`,
      method: "delete",
      params: request
    });
  }

  /** 预览 Gorse 推荐外部推荐脚本。 */
  PreviewExternal(request: PreviewExternalRequest): Promise<PreviewExternalResponse> {
    return service<PreviewExternalRequest, PreviewExternalResponse>({
      url: `${RECOMMEND_GORSE_URL}/external/preview`,
      method: "post",
      data: request
    });
  }

  /** 预览 Gorse 推荐排序提示词。 */
  PreviewRankerPrompt(request: PreviewRankerPromptRequest): Promise<PreviewRankerPromptResponse> {
    return service<PreviewRankerPromptRequest, PreviewRankerPromptResponse>({
      url: `${RECOMMEND_GORSE_URL}/ranker/prompt`,
      method: "post",
      data: request
    });
  }

  /** 查询 Gorse 推荐任务状态。 */
  async ListTasks(request: ListTasksRequest): Promise<ListTasksResponse> {
    const data = await service<ListTasksRequest, unknown>({
      url: `${RECOMMEND_GORSE_URL}/task`,
      method: "get",
      params: request
    });
    return {
      tasks: toArrayField(data, "Tasks", "tasks").map(normalizeTask)
    };
  }

  /** 查询 Gorse 推荐用户列表。 */
  async PageUsers(request: PageUsersRequest): Promise<PageUsersResponse> {
    const data = await service<PageUsersRequest, unknown>({
      url: `${RECOMMEND_GORSE_URL}/user`,
      method: "get",
      params: request
    });
    const record = toRecord(data);
    return {
      cursor: toStringValue(readField(record, "Cursor", "cursor")),
      users: toArrayField(record, "Users", "users").map(normalizeUser)
    };
  }

  /** 查询 Gorse 推荐用户。 */
  async GetUser(request: GetUserRequest): Promise<UserResponse> {
    const data = await service<GetUserRequest, unknown>({
      url: `${RECOMMEND_GORSE_URL}/user/${encodeURIComponent(request.id)}`,
      method: "get"
    });
    return normalizeUser(data);
  }

  /** 删除 Gorse 推荐用户。 */
  DeleteUser(request: DeleteUserRequest): Promise<Empty> {
    return service<DeleteUserRequest, Empty>({
      url: `${RECOMMEND_GORSE_URL}/user/${encodeURIComponent(request.id)}`,
      method: "delete"
    });
  }

  /** 查询 Gorse 推荐相似用户。 */
  async GetUserSimilar(request: GetUserSimilarRequest): Promise<UserSimilarResponse> {
    const data = await service<GetUserSimilarRequest, unknown>({
      url: `${RECOMMEND_GORSE_URL}/user/${encodeURIComponent(request.id)}/similar`,
      method: "get",
      params: {
        recommender: request.recommender,
        category: request.category
      }
    });
    const list = toArrayField(data, "Users", "users");
    return { users: list.map(normalizeUser) };
  }

  /** 查询 Gorse 推荐用户反馈。 */
  async GetUserFeedback(request: GetUserFeedbackRequest): Promise<FeedbackResponse> {
    const data = await service<GetUserFeedbackRequest, unknown>({
      url: `${RECOMMEND_GORSE_URL}/user/${encodeURIComponent(request.id)}/feedback`,
      method: "get",
      params: {
        feedback_type: request.feedback_type,
        offset: request.offset,
        n: request.n
      }
    });
    const list = toArrayField(data, "Feedback", "feedback");
    return { feedback: list.map(normalizeFeedback) };
  }

  /** 查询 Gorse 推荐用户推荐结果。 */
  async GetUserRecommend(request: GetUserRecommendRequest): Promise<ItemListResponse> {
    const data = await service<GetUserRecommendRequest, unknown>({
      url: `${RECOMMEND_GORSE_URL}/user/${encodeURIComponent(request.id)}/recommend`,
      method: "get",
      params: {
        recommender: request.recommender,
        category: request.category,
        n: request.n
      }
    });
    const list = toArrayField(data, "Items", "items");
    return { items: list.map(normalizeItem) };
  }

  /** 查询 Gorse 推荐商品列表。 */
  async PageItems(request: PageItemsRequest): Promise<PageItemsResponse> {
    const data = await service<PageItemsRequest, unknown>({
      url: `${RECOMMEND_GORSE_URL}/item`,
      method: "get",
      params: request
    });
    const record = toRecord(data);
    return {
      cursor: toStringValue(readField(record, "Cursor", "cursor")),
      items: toArrayField(record, "Items", "items").map(normalizeItem)
    };
  }

  /** 查询 Gorse 推荐商品。 */
  async GetItem(request: GetItemRequest): Promise<Item> {
    const data = await service<GetItemRequest, unknown>({
      url: `${RECOMMEND_GORSE_URL}/item/${encodeURIComponent(request.id)}`,
      method: "get"
    });
    return normalizeItem(data);
  }

  /** 删除 Gorse 推荐商品。 */
  DeleteItem(request: DeleteItemRequest): Promise<Empty> {
    return service<DeleteItemRequest, Empty>({
      url: `${RECOMMEND_GORSE_URL}/item/${encodeURIComponent(request.id)}`,
      method: "delete"
    });
  }

  /** 查询 Gorse 推荐相似商品。 */
  async GetItemSimilar(request: GetItemSimilarRequest): Promise<ItemListResponse> {
    const data = await service<GetItemSimilarRequest, unknown>({
      url: `${RECOMMEND_GORSE_URL}/item/${encodeURIComponent(request.id)}/similar`,
      method: "get",
      params: {
        recommender: request.recommender,
        category: request.category
      }
    });
    const list = toArrayField(data, "Items", "items");
    return { items: list.map(normalizeItem) };
  }

  /** 导出 Gorse 推荐数据。 */
  ExportData(request: ExportDataRequest): Promise<ExportDataResponse> {
    return service<ExportDataRequest, ExportDataResponse>({
      url: `${RECOMMEND_GORSE_URL}/export`,
      method: "get",
      params: request
    });
  }

  /** 导入 Gorse 推荐数据。 */
  ImportData(request: ImportDataRequest): Promise<ImportDataResponse> {
    return service<ImportDataRequest, ImportDataResponse>({
      url: `${RECOMMEND_GORSE_URL}/import`,
      method: "post",
      data: request
    });
  }
}

export const defRecommendGorseService = new RecommendGorseServiceImpl();
