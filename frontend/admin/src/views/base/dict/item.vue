<!-- 字典数据 -->
<template>
  <div class="app-container">
    <div class="search-bar mt-5">
      <el-form ref="queryFormRef" :model="queryParams" :inline="true">
        <el-form-item label="字典标签" prop="label">
          <el-input
            v-model="queryParams.label"
            placeholder="字典标签"
            clearable
            @keyup.enter="handleQuery"
          />
        </el-form-item>
        <el-form-item label="状态" prop="status">
          <Dict v-model="queryParams.status" code="status" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="handleQuery()">搜索</el-button>
          <el-button icon="refresh" @click="handleResetQuery()">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <el-card shadow="never">
      <div class="mb-[10px]">
        <el-button
          v-hasPerm="['base:dict-item:create']"
          type="success"
          icon="plus"
          @click="handleOpenDialog()"
        >
          新增
        </el-button>
        <el-button
          v-hasPerm="['base:dict-item:delete']"
          type="danger"
          :disabled="ids.length === 0"
          icon="delete"
          @click="handleDelete()"
        >
          删除
        </el-button>
      </div>

      <el-table
        v-loading="loading"
        highlight-current-row
        :data="tableData"
        border
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" align="center" />
        <el-table-column label="字典标签" prop="label" />
        <el-table-column label="字典值" prop="value" />
        <el-table-column label="排序" prop="sort" />
        <el-table-column label="状态" align="center" prop="status" width="80">
          <template #default="scope">
            <el-switch
              v-model="scope.row.status"
              inline-prompt
              :active-value="Status.ENABLE"
              :inactive-value="Status.DISABLE"
              active-text="启用"
              inactive-text="禁用"
              :disabled="!useUserStore().hasPerm('base:dict-item:status')"
              @change="handleSetStatus(scope.row)"
            />
          </template>
        </el-table-column>
        <el-table-column label="创建时间" prop="createdAt" align="center" />
        <el-table-column label="更新时间" prop="updatedAt" align="center" />
        <el-table-column fixed="right" label="操作" align="center" width="200">
          <template #default="scope">
            <el-button
              v-hasPerm="['base:dict-item:update']"
              type="primary"
              link
              size="small"
              icon="edit"
              @click.stop="handleOpenDialog(scope.row.id)"
            >
              编辑
            </el-button>
            <el-button
              v-hasPerm="['base:dict-item:delete']"
              type="danger"
              link
              size="small"
              icon="delete"
              @click.stop="handleDelete(scope.row.id)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <pagination
        v-if="total > 0"
        v-model:total="total"
        v-model:page="queryParams.pageNum"
        v-model:limit="queryParams.pageSize"
        @pagination="handleQuery"
      />
    </el-card>

    <!--字典弹窗-->
    <el-dialog
      v-model="dialog.visible"
      :title="dialog.title"
      width="820px"
      @close="handleCloseDialog"
    >
      <el-form ref="dataFormRef" :model="formData" :rules="rules" label-width="100px">
        <el-card shadow="never">
          <el-form-item label="字典标签" prop="label">
            <el-input v-model="formData.label" placeholder="请输入字典标签" />
          </el-form-item>
          <el-form-item label="字典值" prop="value">
            <el-input v-model="formData.value" placeholder="请输入字典值" />
          </el-form-item>
          <el-form-item label="状态" prop="status">
            <el-switch
              v-model="formData.status"
              inline-prompt
              active-text="启用"
              inactive-text="禁用"
              :active-value="Status.ENABLE"
              :inactive-value="Status.DISABLE"
            />
          </el-form-item>
          <el-form-item label="排序">
            <el-input-number
              v-model="formData.sort"
              controls-position="right"
              :min="1"
              :precision="0"
              :step="1"
            />
          </el-form-item>
          <el-form-item label="标签类型">
            <el-tag v-if="formData.tagType" :type="formData.tagType as 'success' | 'info' | 'warning' | 'primary' | 'danger'" class="mr-2">
              {{ formData.label }}
            </el-tag>
            <el-radio-group v-model="formData.tagType">
              <el-radio value="success" border size="small">success</el-radio>
              <el-radio value="warning" border size="small">warning</el-radio>
              <el-radio value="info" border size="small">info</el-radio>
              <el-radio value="primary" border size="small">primary</el-radio>
              <el-radio value="danger" border size="small">danger</el-radio>
              <el-radio value="" border size="small">清空</el-radio>
            </el-radio-group>
          </el-form-item>
        </el-card>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button type="primary" @click="handleSubmitClick">确 定</el-button>
          <el-button @click="handleCloseDialog">取 消</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { useUserStore } from "@/store";

defineOptions({
  name: "BaseDictItem",
  inheritAttrs: false,
});

import { defBaseDictService } from "@/api/admin/base_dict";
import type {
  BaseDictItem,
  BaseDictItemForm,
  PageBaseDictItemRequest,
} from "@/rpc/admin/base_dict";
import { Status } from "@/rpc/common/enum";
import { formatJson } from "@/utils/utils";

const route = useRoute();

const dictId = ref(route.query.dictId as unknown as number);

const queryFormRef = ref(ElForm);
const dataFormRef = ref(ElForm);

const loading = ref(false);
const ids = ref<number[]>([]);
const total = ref(0);

const queryParams = reactive<PageBaseDictItemRequest>({
  /** 字典id */
  dictId: dictId.value,
  /** 字典属性名称 */
  label: "",
  /** 状态 */
  status: undefined,
  /** 当前页码 */
  pageNum: 0,
  /** 每一页的行数 */
  pageSize: 10,
});

const tableData = ref<BaseDictItem[]>();

const dialog = reactive({
  title: "",
  visible: false,
});

const formData = reactive<BaseDictItemForm>({
  /** 字典ID */
  id: 0,
  /** 字典ID */
  dictId: dictId.value,
  /** 字典值 */
  value: "",
  /** 字典项标签 */
  label: "",
  /** 标签类型，用于前端样式展示（如success、warning等） */
  tagType: "" as "success" | "info" | "warning" | "primary" | "danger" | "",
  /** 排序 */
  sort: 1,
  /** 状态 */
  status: Status.ENABLE,
});

// 监听路由参数变化，更新字典数据
watch(
  () => [route.query.dictId],
  ([newDictId]) => {
    queryParams.dictId = newDictId as unknown as number;
    dictId.value = newDictId as unknown as number;
    handleQuery();
  }
);
const rules = computed(() => {
  const rules: Partial<Record<string, any>> = {
    value: [{ required: true, message: "请输入字典值", trigger: "blur" }],
    label: [{ required: true, message: "请输入字典标签", trigger: "blur" }],
    status: [{ required: true, message: "状态不能为空", trigger: "blur" }],
  };
  return rules;
});

// 查询
function handleQuery() {
  loading.value = true;
  defBaseDictService
    .PageBaseDictItem(queryParams)
    .then((data) => {
      tableData.value = data.list;
      total.value = data.total;
    })
    .finally(() => {
      loading.value = false;
    });
}

// 重置查询
function handleResetQuery() {
  queryFormRef.value.resetFields();
  queryParams.pageNum = 1;
  handleQuery();
}

// 行选择
function handleSelectionChange(selection: any) {
  ids.value = selection.map((item: any) => item.id);
}

// 打开弹窗
function handleOpenDialog(dictItemId?: number) {
  dialog.visible = true;
  if (dictItemId) {
    dialog.title = "修改字典数据";
    defBaseDictService
      .GetBaseDictItem({
        value: dictItemId,
      })
      .then((data) => {
        Object.assign(formData, { ...data });
      });
  } else {
    dialog.title = "新增字典数据";
  }
}

// 提交表单
function handleSubmitClick() {
  dataFormRef.value.validate((isValid: boolean) => {
    if (isValid) {
      loading.value = true;
      const id = formData.id;
      formData.dictId = dictId.value;
      if (id) {
        defBaseDictService
          .UpdateBaseDictItem(formData)
          .then(() => {
            ElMessage.success("修改成功");
            handleCloseDialog();
            handleQuery();
          })
          .finally(() => (loading.value = false));
      } else {
        defBaseDictService
          .CreateBaseDictItem(formData)
          .then(() => {
            ElMessage.success("新增成功");
            handleCloseDialog();
            handleQuery();
          })
          .finally(() => (loading.value = false));
      }
    }
  });
}

// 关闭弹窗
function handleCloseDialog() {
  dialog.visible = false;

  dataFormRef.value.resetFields();
  dataFormRef.value.clearValidate();

  formData.id = 0;
}

// 设置字典状态
function handleSetStatus(row: BaseDictItem) {
  let text = row.status === Status.ENABLE ? "启用" : "禁用";
  ElMessageBox.confirm(`是否确定${text}字典属性为：${row.label}?`, "提示", {
    confirmButtonText: "确认",
    cancelButtonText: "取消",
    type: "warning",
  })
    .then(() => {
      defBaseDictService.SetBaseDictItemStatus({ id: row.id, status: row.status }).then(() => {
        handleQuery();
        ElMessage.success(`${text}成功`);
      });
    })
    .catch(() => {
      if (row.status == 0) {
        row.status = 1;
      } else {
        row.status = 0;
      }
    });
}

/**
 * 删除字典
 *
 * @param dictItemId 字典ID
 */
function handleDelete(dictItemId?: number) {
  const dictItemIds = [dictItemId || ids.value].join(",");
  if (!dictItemIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }
  ElMessageBox.confirm("确认删除已选中的数据项?", "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning",
  }).then(
    () => {
      defBaseDictService
        .DeleteBaseDictItem({
          value: dictItemIds,
        })
        .then(() => {
          ElMessage.success("删除成功");
          handleResetQuery();
        });
    },
    () => {
      ElMessage.info("已取消删除");
    }
  );
}

onMounted(() => {
  handleQuery();
});
</script>
