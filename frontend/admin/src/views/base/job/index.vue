<!-- 定时任务 -->
<template>
  <div class="app-container">
    <div class="search-bar">
      <el-form ref="queryFormRef" :model="queryParams" :inline="true">
        <el-form-item label="任务名称" prop="name">
          <el-input
            v-model="queryParams.name"
            placeholder="任务名称"
            clearable
            @keyup.enter="handleQuery"
          />
        </el-form-item>
        <el-form-item label="调用目标" prop="invokeTarget">
          <el-input
            v-model="queryParams.invokeTarget"
            placeholder="调用目标"
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
          v-hasPerm="['base:job:create']"
          type="success"
          icon="plus"
          @click="handleOpenDialog()"
        >
          新增
        </el-button>
        <el-button
          v-hasPerm="['base:job:delete']"
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
        <el-table-column label="任务名称" prop="name" />
        <el-table-column label="调用目标" prop="invokeTarget" />
        <el-table-column label="参数" show-overflow-tooltip>
          <template #default="scope">
            <el-tag v-for="(arg, index) in scope.row.args" :key="index" class="mr-5">
              {{ arg.key }}={{ arg.value }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="cron表达式" prop="cronExpression" align="center" />
        <el-table-column label="任务id" prop="entryId" align="right" />
        <el-table-column label="状态" align="center" prop="status" width="80">
          <template #default="scope">
            <el-switch
              v-model="scope.row.status"
              inline-prompt
              :active-value="Status.ENABLE"
              :inactive-value="Status.DISABLE"
              active-text="启用"
              inactive-text="禁用"
              :disabled="scope.row.entryId || !useUserStore().hasPerm('base:job:status')"
              @change="handleSetStatus(scope.row)"
            />
          </template>
        </el-table-column>
        <el-table-column label="创建时间" prop="createdAt" align="center" />
        <el-table-column label="更新时间" prop="updatedAt" align="center" />
        <el-table-column fixed="right" label="操作" align="center" width="220">
          <template #default="scope">
            <div style="display: flex; align-items: center; gap: var(--button-gap)">
              <el-button
                v-hasPerm="['base:job:update']"
                type="primary"
                link
                size="small"
                icon="edit"
                style="margin: 0; padding: 0 2px"
                @click.stop="handleOpenDialog(scope.row.id)"
              >
                编辑
              </el-button>
              <el-button
                v-hasPerm="['base:job:delete']"
                type="danger"
                link
                size="small"
                icon="delete"
                style="margin: 0; padding: 0 2px"
                @click.stop="handleDelete(scope.row.id)"
              >
                删除
              </el-button>
              <!-- 更多操作下拉菜单 -->
              <el-dropdown
                v-if="showMoreMenu(scope.row)"
                style="margin: 0"
                @command="(cmd) => handleMenuCommand(cmd, scope.row)"
              >
                <el-button type="primary" link size="small" style="margin: 0; padding: 0 2px">
                  更多
                  <el-icon class="el-icon--right"><arrow-down /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item
                      v-if="
                        scope.row.status == 1 &&
                        !scope.row.entryId &&
                        useUserStore().hasPerm('base:job:start')
                      "
                      command="start"
                    >
                      启动
                    </el-dropdown-item>
                    <el-dropdown-item
                      v-if="
                        scope.row.status == 1 &&
                        scope.row.entryId &&
                        useUserStore().hasPerm('base:job:stop')
                      "
                      command="stop"
                    >
                      停止
                    </el-dropdown-item>
                    <el-dropdown-item
                      v-if="scope.row.status == 1 && useUserStore().hasPerm('base:job:exec')"
                      command="exec"
                    >
                      执行一次
                    </el-dropdown-item>
                    <el-dropdown-item v-if="useUserStore().hasPerm('base:job:log')" command="log">
                      日志
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
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

    <!--定时任务弹窗-->
    <el-dialog
      v-model="dialog.visible"
      :title="dialog.title"
      width="1000px"
      @close="handleCloseDialog"
    >
      <el-form ref="dataFormRef" :model="formData" :rules="rules" label-width="150px">
        <el-card shadow="never">
          <el-form-item label="任务名称" prop="name">
            <el-input v-model="formData.name" placeholder="请输入任务名称" />
          </el-form-item>

          <el-form-item label="调用目标" prop="invokeTarget">
            <el-input v-model="formData.invokeTarget" placeholder="请输入调用目标" />
          </el-form-item>

          <el-form-item label="cron表达式" prop="cronExpression">
            <el-input v-model="formData.cronExpression" placeholder="0 0 0 * * ? " />
          </el-form-item>

          <el-form-item label="目标参数" prop="args">
            <div v-for="(arg, index) in formData.args" :key="index" class="flex mb-10px">
              <el-input v-model="arg.key" placeholder="参数" class="mr-10px" style="width: 200px" />
              <el-input v-model="arg.value" placeholder="值" class="mr-10px" style="width: 200px" />
              <el-button type="danger" circle icon="Delete" @click="removeArg(index)" />
            </div>
            <el-button type="primary" icon="Plus" @click="addArg">添加参数</el-button>
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
  name: "BaseJob",
  inheritAttrs: false,
});

import { defBaseJobService } from "@/api/admin/base_job";
import { BaseJob, BaseJobArgs, BaseJobForm, PageBaseJobRequest } from "@/rpc/admin/base_job";

import router from "@/router";
import { Status } from "@/rpc/common/enum";

const queryFormRef = ref(ElForm);
const dataFormRef = ref(ElForm);

const loading = ref(false);
const ids = ref<number[]>([]);
const total = ref(0);

const queryParams = reactive<PageBaseJobRequest>({
  /** 任务名称 */
  name: "",
  /** 调用目标 */
  invokeTarget: "",
  /** 状态 */
  status: undefined,
  /** 当前页码 */
  pageNum: 0,
  /** 每一页的行数 */
  pageSize: 10,
});

const tableData = ref<BaseJob[]>();

const dialog = reactive({
  title: "",
  visible: false,
});

const formData = reactive<BaseJobForm>({
  /** 定时任务ID */
  id: 0,
  /** 任务名称 */
  name: "",
  /** 调用目标 */
  invokeTarget: "",
  /** 目标参数 */
  args: [],
  /** cron表达式 */
  cronExpression: "",
  /** 状态 */
  status: Status.ENABLE,
});

const rules = computed(() => {
  const rules: Partial<Record<string, any>> = {
    name: [{ required: true, message: "请输入任务名称", trigger: "blur" }],
    cronExpression: [{ required: true, message: "请输入cron表达式", trigger: "blur" }],
    invokeTarget: [{ required: true, message: "请输入调用目标", trigger: "blur" }],
    args: {
      validator: (rule: any, value: BaseJobArgs[], callback: any) => {
        if (value.some((arg) => !arg.key)) {
          callback(new Error("所有参数必须填写key"));
        } else {
          callback();
        }
      },
      trigger: "blur",
    },
    status: [{ required: true, message: "状态不能为空", trigger: "blur" }],
  };
  return rules;
});

// 添加参数
const addArg = () => {
  formData.args.push({ key: "", value: "" });
};

// 删除参数
const removeArg = (index: number) => {
  formData.args.splice(index, 1);
};

// // 判断是否显示更多菜单
function showMoreMenu(row: BaseJob) {
  let res = 0;
  if (row.status === 1 && row.entryId === 0 && useUserStore().hasPerm("base:job:start")) {
    res += 1;
  }
  if (row.status == 1 && row.entryId > 0 && useUserStore().hasPerm("base:job:stop")) {
    res += 1;
  }
  if (row.status == 1 && useUserStore().hasPerm("base:job:exec")) {
    res += 1;
  }
  if (useUserStore().hasPerm("base:job:log")) {
    res += 1;
  }
  return res;
}

const handleMenuCommand = (command: string, row: BaseJob) => {
  const actions: { [key: string]: () => void } = {
    start: () => handleStart(row.id, row.name),
    stop: () => handleStop(row.id, row.name),
    exec: () => handleExec(row.id, row.name),
    log: () => handleOpenBaseJob(row.id, row.name),
  };
  actions[command]?.();
};

// 查询
function handleQuery() {
  loading.value = true;
  defBaseJobService
    .PageBaseJob(queryParams)
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

/**
 * 打开弹窗
 *
 * @param jobId
 */
function handleOpenDialog(jobId?: number) {
  dialog.visible = true;
  if (jobId) {
    dialog.title = "修改定时任务";
    defBaseJobService
      .GetBaseJob({
        value: jobId,
      })
      .then((data) => {
        Object.assign(formData, { ...data });
      });
  } else {
    dialog.title = "新增定时任务";
  }
}

// 关闭弹窗
function handleCloseDialog() {
  dialog.visible = false;
  dataFormRef.value.resetFields();
  dataFormRef.value.clearValidate();

  formData.id = 0;
}

// 提交定时任务表单
function handleSubmitClick() {
  dataFormRef.value.validate((isValid: boolean) => {
    if (isValid) {
      const jobId = formData.id;
      loading.value = true;
      if (jobId) {
        defBaseJobService
          .UpdateBaseJob(formData)
          .then(() => {
            ElMessage.success("修改成功");
            handleCloseDialog();
            handleQuery();
          })
          .finally(() => (loading.value = false));
      } else {
        defBaseJobService
          .CreateBaseJob(formData)
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

// 设置定时任务状态
function handleSetStatus(row: BaseJob) {
  let text = row.status === Status.ENABLE ? "启用" : "禁用";
  ElMessageBox.confirm(`是否确定${text}定时任务为：${row.name}?`, "提示", {
    confirmButtonText: "确认",
    cancelButtonText: "取消",
    type: "warning",
  })
    .then(() => {
      defBaseJobService.SetBaseJobStatus({ id: row.id, status: row.status }).then(() => {
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
 * 删除定时任务
 *
 * @param jobId 定时任务ID
 */
function handleDelete(jobId?: number) {
  const jobIds = [jobId || ids.value].join(",");
  if (!jobIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }
  ElMessageBox.confirm("确认删除已选中的数据项?", "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning",
  }).then(
    () => {
      defBaseJobService
        .DeleteBaseJob({
          value: jobIds,
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

/**
 * 启动定时任务
 *
 * @param id 定时任务ID
 * @param name 定时任务名称
 */
function handleStart(id: number, name: string) {
  ElMessageBox.confirm(`确定启动【${name}】定时任务?`, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning",
  }).then(
    () => {
      defBaseJobService
        .StartBaseJob({
          id: id,
        })
        .then(() => {
          ElMessage.success("启动成功");
          handleResetQuery();
        });
    },
    () => {
      ElMessage.info("已取消启动");
    }
  );
}

/**
 * 停止定时任务
 *
 * @param id 定时任务ID
 * @param name 定时任务名称
 */
function handleStop(id: number, name: string) {
  ElMessageBox.confirm(`确定停止【${name}】定时任务?`, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning",
  }).then(
    () => {
      defBaseJobService
        .StopBaseJob({
          id: id,
        })
        .then(() => {
          ElMessage.success("停止成功");
          handleResetQuery();
        });
    },
    () => {
      ElMessage.info("已取消停止");
    }
  );
}

/**
 * 执行定时任务
 *
 * @param id 定时任务ID
 * @param name 定时任务名称
 */
function handleExec(id: number, name: string) {
  ElMessageBox.confirm(`确定执行【${name}】定时任务?`, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning",
  }).then(
    () => {
      defBaseJobService
        .ExecBaseJob({
          id: id,
        })
        .then(() => {
          ElMessage.success("执行成功");
          handleResetQuery();
        });
    },
    () => {
      ElMessage.info("已取消执行");
    }
  );
}

// 打开定时任务数据
function handleOpenBaseJob(id: number, name: string) {
  router.push({
    path: "/base/job-log",
    query: { jobId: id, title: `【${name}】定时任务日志` },
  });
}

onMounted(() => {
  handleQuery();
});
</script>
<style>
:root {
  --button-gap: 12px; /* 通过CSS变量控制间距 */
}
</style>
