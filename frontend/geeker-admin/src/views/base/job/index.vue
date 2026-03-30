<!-- 定时任务 -->
<template>
  <div class="table-box">
    <ProTable ref="proTable" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestBaseJobTable">
      <template #args="scope">
        <el-tag v-for="(arg, index) in scope.row.args" :key="index" class="mr-5">{{ arg.key }}={{ arg.value }}</el-tag>
      </template>

      <template #operation="scope">
        <div class="job-operation">
          <el-button
            v-if="BUTTONS['base:job:update']"
            type="primary"
            link
            :icon="EditPen"
            @click.stop="handleOpenDialog(scope.row.id)"
          >
            编辑
          </el-button>
          <el-button v-if="BUTTONS['base:job:delete']" type="danger" link :icon="Delete" @click.stop="handleDelete(scope.row)">
            删除
          </el-button>
          <el-dropdown v-if="showMoreMenu(scope.row)" @command="command => handleMenuCommand(command, scope.row)">
            <el-button type="primary" link>
              更多
              <el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item
                  v-if="scope.row.status == 1 && scope.row.entryId === 0 && BUTTONS['base:job:start']"
                  command="start"
                >
                  启动
                </el-dropdown-item>
                <el-dropdown-item
                  v-if="scope.row.status == 1 && scope.row.entryId > 0 && BUTTONS['base:job:stop']"
                  command="stop"
                >
                  停止
                </el-dropdown-item>
                <el-dropdown-item v-if="scope.row.status == 1 && BUTTONS['base:job:exec']" command="exec">
                  执行一次
                </el-dropdown-item>
                <el-dropdown-item v-if="BUTTONS['base:job:log']" command="log">日志</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </template>
    </ProTable>

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="dialog.title"
      width="1000px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      label-width="150px"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { ArrowDown, CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@/components/ProForm/interface";
import ProTable from "@/components/ProTable/index.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { defBaseJobService } from "@/api/admin/base_job";
import type { BaseJob, BaseJobArgs, BaseJobForm, PageBaseJobRequest } from "@/rpc/admin/base_job";
import router from "@/routers";
import { Status } from "@/rpc/common/enum";
import { buildPageRequest, normalizeSelectedIds } from "@/utils/proTable";

defineOptions({
  name: "BaseJob",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();

const dialog = reactive({
  title: "",
  visible: false
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
  status: Status.ENABLE
});

const rules = computed(() => ({
  name: [{ required: true, message: "请输入任务名称", trigger: "blur" }],
  cronExpression: [{ required: true, message: "请输入cron表达式", trigger: "blur" }],
  invokeTarget: [{ required: true, message: "请输入调用目标", trigger: "blur" }],
  args: {
    validator: (rule: unknown, value: BaseJobArgs[], callback: (error?: Error) => void) => {
      if (value.some(arg => !arg.key)) callback(new Error("所有参数必须填写key"));
      else callback();
    },
    trigger: "blur"
  },
  status: [{ required: true, message: "状态不能为空", trigger: "blur" }]
}));

const statusOptions: ProFormOption[] = [
  { label: "启用", value: Status.ENABLE },
  { label: "禁用", value: Status.DISABLE }
];

/** 定时任务表单字段配置。 */
const formFields: ProFormField[] = [
  { prop: "name", label: "任务名称", component: "input", props: { placeholder: "请输入任务名称" } },
  { prop: "invokeTarget", label: "调用目标", component: "input", props: { placeholder: "请输入调用目标" } },
  { prop: "cronExpression", label: "cron表达式", component: "cron-expression", props: { placeholder: "0 0 0 * * ?" } },
  {
    prop: "args",
    label: "目标参数",
    component: "kv-list",
    props: {
      keyInputProps: { placeholder: "参数" },
      valueInputProps: { placeholder: "值" },
      addText: "添加参数"
    }
  },
  { prop: "status", label: "状态", component: "radio-group", options: statusOptions }
];

/** 定时任务表格列配置。 */
const columns: ColumnProps[] = [
  { type: "selection", width: 55 },
  { prop: "name", label: "任务名称", search: { el: "input" } },
  { prop: "invokeTarget", label: "调用目标", search: { el: "input" } },
  { prop: "args", label: "参数" },
  { prop: "cronExpression", label: "cron表达式", align: "center" },
  { prop: "entryId", label: "任务id", align: "right" },
  {
    prop: "status",
    label: "状态",
    width: 100,
    search: { el: "select" },
    cellType: "status",
    statusProps: {
      activeValue: Status.ENABLE,
      inactiveValue: Status.DISABLE,
      activeText: "启用",
      inactiveText: "禁用",
      disabled: scope => (scope.row as BaseJob).entryId === 0 || !BUTTONS.value["base:job:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BaseJob)
    }
  },
  { prop: "createdAt", label: "创建时间", width: 180 },
  { prop: "updatedAt", label: "更新时间", width: 180 },
  { prop: "operation", label: "操作", width: 240, fixed: "right" }
];

/** 定时任务顶部按钮配置。 */
const headerActions: HeaderActionProps[] = [
  {
    label: "新增",
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:job:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: "删除",
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:job:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BaseJob[])
  }
];

/**
 * 判断当前任务是否需要展示“更多”操作菜单。
 */
function showMoreMenu(row: BaseJob) {
  let count = 0;
  // 脚本中读取 computed 需显式取 value，否则权限判断始终为 false。
  if (row.status === Status.ENABLE && row.entryId === 0 && BUTTONS.value["base:job:start"]) count += 1;
  if (row.status === Status.ENABLE && row.entryId > 0 && BUTTONS.value["base:job:stop"]) count += 1;
  if (row.status === Status.ENABLE && BUTTONS.value["base:job:exec"]) count += 1;
  if (BUTTONS.value["base:job:log"]) count += 1;
  return count > 0;
}

/**
 * 分发定时任务更多菜单命令。
 */
function handleMenuCommand(command: string, row: BaseJob) {
  const actions: Record<string, () => void> = {
    start: () => handleStart(row.id, row.name),
    stop: () => handleStop(row.id, row.name),
    exec: () => handleExec(row.id, row.name),
    log: () => handleOpenBaseJob(row.id, row.name)
  };
  actions[command]?.();
}

/**
 * 请求定时任务列表，并由 ProTable 统一维护分页与搜索参数。
 */
async function requestBaseJobTable(params: PageBaseJobRequest) {
  const data = await defBaseJobService.PageBaseJob(buildPageRequest(params));
  return { data };
}

/**
 * 刷新定时任务表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 打开定时任务弹窗。
 */
function handleOpenDialog(jobId?: number) {
  resetForm();
  dialog.title = jobId ? "修改定时任务" : "新增定时任务";
  dialog.visible = true;
  if (!jobId) return;

  defBaseJobService.GetBaseJob({ value: jobId }).then(data => {
    Object.assign(formData, data);
  });
}

/**
 * 关闭定时任务弹窗并恢复默认表单值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/**
 * 重置定时任务表单。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.name = "";
  formData.invokeTarget = "";
  formData.args = [];
  formData.cronExpression = "";
  formData.status = Status.ENABLE;
}

/**
 * 提交定时任务表单。
 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;

    const submitData = JSON.parse(JSON.stringify(formData)) as BaseJobForm;
    const request = submitData.id ? defBaseJobService.UpdateBaseJob(submitData) : defBaseJobService.CreateBaseJob(submitData);
    request.then(() => {
      ElMessage.success(submitData.id ? "修改成功" : "新增成功");
      handleCloseDialog();
      refreshTable();
    });
  });
}

/**
 * 在定时任务状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: BaseJob) {
  const nextStatus = row.status === Status.ENABLE ? Status.DISABLE : Status.ENABLE;
  const text = nextStatus === Status.ENABLE ? "启用" : "禁用";
  const jobName = row.name || row.invokeTarget || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(`是否确定${text}定时任务：${jobName}？`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defBaseJobService.SetBaseJobStatus({ id: row.id, status: nextStatus });
    ElMessage.success(`${text}成功`);
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除定时任务，兼容单条删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | BaseJob | BaseJob[]) {
  const jobList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as BaseJob[])
    : selected && typeof selected === "object"
      ? [selected as BaseJob]
      : [];
  const jobIds = (
    jobList.length ? jobList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!jobIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  const confirmMessage = jobList.length
    ? jobList.length === 1
      ? `是否确定删除定时任务：${jobList[0].name || jobList[0].invokeTarget || `ID:${jobList[0].id}`}？`
      : `确认删除已选中的 ${jobList.length} 个定时任务吗？`
    : "确认删除已选中的定时任务吗？";

  ElMessageBox.confirm(confirmMessage, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseJobService.DeleteBaseJob({ value: jobIds }).then(() => {
        ElMessage.success("删除成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消删除");
    }
  );
}

/**
 * 启动定时任务。
 */
function handleStart(id: number, name: string) {
  ElMessageBox.confirm(`确定启动【${name}】定时任务?`, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseJobService.StartBaseJob({ id }).then(() => {
        ElMessage.success("启动成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消启动");
    }
  );
}

/**
 * 停止定时任务。
 */
function handleStop(id: number, name: string) {
  ElMessageBox.confirm(`确定停止【${name}】定时任务?`, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseJobService.StopBaseJob({ id }).then(() => {
        ElMessage.success("停止成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消停止");
    }
  );
}

/**
 * 执行一次定时任务。
 */
function handleExec(id: number, name: string) {
  ElMessageBox.confirm(`确定执行【${name}】定时任务?`, "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(
    () => {
      defBaseJobService.ExecBaseJob({ id }).then(() => {
        ElMessage.success("执行成功");
        refreshTable();
      });
    },
    () => {
      ElMessage.info("已取消执行");
    }
  );
}

/**
 * 打开定时任务日志页面。
 */
function handleOpenBaseJob(id: number, name: string) {
  router.push({
    path: "/base/job-log",
    query: { jobId: id, title: `【${name}】定时任务日志` }
  });
}
</script>

<style scoped>
.job-operation {
  display: flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
}
</style>
