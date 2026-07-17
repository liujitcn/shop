import service from "@/utils/request";
import {
  type CodeGenProtoService,
  type ListCodeGenProtosRequest,
  type ListCodeGenProtosResponse,
  type SaveCodeGenProtosRequest
} from "@/rpc/admin/v1/code_gen_proto";
import type { Empty } from "@/rpc/google/protobuf/empty";

const CODE_GEN_TABLE_URL = "/v1/admin/code-gen/table";

/** Admin代码生成Proto接口配置服务。 */
export class CodeGenProtoServiceImpl implements CodeGenProtoService {
  /** 查询代码生成Proto接口配置。 */
  ListCodeGenProtos(request: ListCodeGenProtosRequest): Promise<ListCodeGenProtosResponse> {
    return service<ListCodeGenProtosRequest, ListCodeGenProtosResponse>({
      url: `${CODE_GEN_TABLE_URL}/${request.table_id}/proto`,
      method: "get"
    });
  }

  /** 保存代码生成Proto接口配置。 */
  SaveCodeGenProtos(request: SaveCodeGenProtosRequest): Promise<Empty> {
    return service<SaveCodeGenProtosRequest, Empty>({
      url: `${CODE_GEN_TABLE_URL}/${request.table_id}/proto`,
      method: "put",
      data: request
    });
  }
}

export const defCodeGenProtoService = new CodeGenProtoServiceImpl();
