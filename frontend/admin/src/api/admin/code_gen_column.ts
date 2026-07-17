import service from "@/utils/request";
import {
  type CodeGenColumnService,
  type ListCodeGenDatabaseColumnsRequest,
  type ListCodeGenDatabaseColumnsResponse,
  type ListCodeGenColumnsRequest,
  type ListCodeGenColumnsResponse,
  type SaveCodeGenColumnsRequest
} from "@/rpc/admin/v1/code_gen_column";
import type { Empty } from "@/rpc/google/protobuf/empty";

const CODE_GEN_DATABASE_TABLE_URL = "/v1/admin/code-gen/database/table";
const CODE_GEN_TABLE_URL = "/v1/admin/code-gen/table";

/** Admin代码生成字段服务。 */
export class CodeGenColumnServiceImpl implements CodeGenColumnService {
  /** 查询数据库表字段列表。 */
  ListCodeGenDatabaseColumns(request: ListCodeGenDatabaseColumnsRequest): Promise<ListCodeGenDatabaseColumnsResponse> {
    return service<ListCodeGenDatabaseColumnsRequest, ListCodeGenDatabaseColumnsResponse>({
      url: `${CODE_GEN_DATABASE_TABLE_URL}/${request.table_name}/column`,
      method: "get"
    });
  }

  /** 查询代码生成字段配置。 */
  ListCodeGenColumns(request: ListCodeGenColumnsRequest): Promise<ListCodeGenColumnsResponse> {
    return service<ListCodeGenColumnsRequest, ListCodeGenColumnsResponse>({
      url: `${CODE_GEN_TABLE_URL}/${request.table_id}/column`,
      method: "get"
    });
  }

  /** 保存代码生成字段配置。 */
  SaveCodeGenColumns(request: SaveCodeGenColumnsRequest): Promise<Empty> {
    return service<SaveCodeGenColumnsRequest, Empty>({
      url: `${CODE_GEN_TABLE_URL}/${request.table_id}/column`,
      method: "put",
      data: request
    });
  }
}

export const defCodeGenColumnService = new CodeGenColumnServiceImpl();
