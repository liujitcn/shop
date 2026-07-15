import service from "@/utils/request";
import {
  type CodeGenColumnService,
  type ListCodeGenDatabaseColumnsRequest,
  type ListCodeGenDatabaseColumnsResponse
} from "@/rpc/admin/v1/code_gen_column";

const CODE_GEN_DATABASE_TABLE_URL = "/v1/admin/code-gen/database/table";

/** Admin代码生成字段服务。 */
export class CodeGenColumnServiceImpl implements CodeGenColumnService {
  /** 查询数据库表字段列表。 */
  ListCodeGenDatabaseColumns(request: ListCodeGenDatabaseColumnsRequest): Promise<ListCodeGenDatabaseColumnsResponse> {
    return service<ListCodeGenDatabaseColumnsRequest, ListCodeGenDatabaseColumnsResponse>({
      url: `${CODE_GEN_DATABASE_TABLE_URL}/${request.table_name}/column`,
      method: "get"
    });
  }
}

export const defCodeGenColumnService = new CodeGenColumnServiceImpl();
