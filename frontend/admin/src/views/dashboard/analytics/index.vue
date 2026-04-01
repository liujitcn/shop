<template>
  <div class="dashboard-container">
    <!-- 数据卡片 -->
    <el-row :gutter="10" class="mt-3">
      <!-- 用户 -->
      <el-col :xs="24" :sm="12" :lg="6">
        <el-card shadow="never">
          <template #header>
            <div class="flex items-center justify-between">
              <span class="text-[var(--el-text-color-secondary)]">新增用户</span>
              <el-tag type="success">日</el-tag>
            </div>
          </template>

          <div class="flex items-center justify-between mt-5">
            <div class="text-lg text-right">
              {{ dashboardCountUser.newNum }}
            </div>
            <svg-icon icon-class="visit" size="2em" />
          </div>

          <div
            class="flex items-center justify-between mt-5 text-sm text-[var(--el-text-color-secondary)]"
          >
            <span>总用户数</span>
            <span>{{ dashboardCountUser.totalNum }}</span>
          </div>
        </el-card>
      </el-col>

      <!--商品-->
      <el-col :xs="24" :sm="12" :lg="6">
        <el-card shadow="never">
          <template #header>
            <div class="flex items-center justify-between">
              <span class="text-[var(--el-text-color-secondary)]">新增商品</span>
              <el-tag type="success">日</el-tag>
            </div>
          </template>

          <div class="flex items-center justify-between mt-5">
            <div class="text-lg text-right">
              {{ dashboardCountGoods.newNum }}
            </div>
            <svg-icon icon-class="ip" size="2em" />
          </div>

          <div
            class="flex items-center justify-between mt-5 text-sm text-[var(--el-text-color-secondary)]"
          >
            <span>总商品数</span>
            <span>{{ dashboardCountGoods.totalNum }}</span>
          </div>
        </el-card>
      </el-col>

      <!--订单量-->
      <el-col :xs="24" :sm="12" :lg="6">
        <el-card shadow="never">
          <template #header>
            <div class="flex items-center justify-between">
              <span class="text-[var(--el-text-color-secondary)]">订单量</span>
              <el-tag type="success">日</el-tag>
            </div>
          </template>

          <div class="flex items-center justify-between mt-5">
            <div class="text-lg text-right">
              {{ dashboardCountOrder.newNum }}
            </div>
            <svg-icon icon-class="order" size="2em" />
          </div>

          <div
            class="flex items-center justify-between mt-5 text-sm text-[var(--el-text-color-secondary)]"
          >
            <span>总订单量</span>
            <span>{{ dashboardCountOrder.totalNum }}</span>
          </div>
        </el-card>
      </el-col>

      <!--销售额-->
      <el-col :xs="24" :sm="12" :lg="6">
        <el-card shadow="never">
          <template #header>
            <div class="flex items-center justify-between">
              <span class="text-[var(--el-text-color-secondary)]">销售额</span>
              <el-tag type="success">日</el-tag>
            </div>
          </template>

          <div class="flex items-center justify-between mt-5">
            <div class="text-lg text-right">
              {{ formatPrice(dashboardCountOrder.newNum) }}
            </div>
            <svg-icon icon-class="money" size="2em" />
          </div>

          <div
            class="flex items-center justify-between mt-5 text-sm text-[var(--el-text-color-secondary)]"
          >
            <span>总销售额</span>
            <span>{{ formatPrice(dashboardCountOrder.totalNum) }}</span>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Echarts 图表 -->
    <el-row :gutter="10" class="mt-3">
      <el-col :sm="24" :lg="12" class="mb-2">
        <OrderBarChart class="bg-[var(--el-bg-color-overlay)]" />
      </el-col>

      <el-col :sm="24" :lg="12" class="mb-2">
        <GoodsBarChart class="bg-[var(--el-bg-color-overlay)]" />
      </el-col>

      <el-col :sm="24" :lg="12" class="mb-2">
        <GoodsPieChart class="bg-[var(--el-bg-color-overlay)]" />
      </el-col>

      <el-col :sm="24" :lg="12" class="mb-2">
        <OrderRadarChart class="bg-[var(--el-bg-color-overlay)]" />
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import OrderBarChart from "./components/OrderBarChart.vue";
import GoodsBarChart from "./components/GoodsBarChart.vue";
import GoodsPieChart from "./components/GoodsPieChart.vue";
import OrderRadarChart from "./components/OrderRadarChart.vue";
import { useUserStore } from "@/store/modules/user";
import { defAnalyticsService } from "@/api/admin/analytics";
import type { AnalyticsCountResponse } from "@/rpc/admin/analytics";
import { AnalyticsTimeType } from "@/rpc/admin/analytics";
import { formatPrice } from "@/utils/utils";

defineOptions({
  name: "Analytics",
  inheritAttrs: false,
});

const dashboardCountUser = reactive<AnalyticsCountResponse>({
  /** 新增数量 */
  newNum: 0,
  /** 总数量 */
  totalNum: 0,
});
const dashboardCountGoods = reactive<AnalyticsCountResponse>({
  /** 新增数量 */
  newNum: 0,
  /** 总数量 */
  totalNum: 0,
});
const dashboardCountOrder = reactive<AnalyticsCountResponse>({
  /** 新增数量 */
  newNum: 0,
  /** 总数量 */
  totalNum: 0,
});
const dashboardCountSale = reactive<AnalyticsCountResponse>({
  /** 新增数量 */
  newNum: 0,
  /** 总数量 */
  totalNum: 0,
});

async function handleQuery() {
  const user = await defAnalyticsService.AnalyticsCountUser({
    timeType: AnalyticsTimeType.DAY,
  });
  Object.assign(dashboardCountUser, user);

  const goods = await defAnalyticsService.AnalyticsCountGoods({
    timeType: AnalyticsTimeType.DAY,
  });
  Object.assign(dashboardCountGoods, goods);

  const order = await defAnalyticsService.AnalyticsCountOrder({
    timeType: AnalyticsTimeType.DAY,
  });
  Object.assign(dashboardCountOrder, order);

  const sale = await defAnalyticsService.AnalyticsCountSale({
    timeType: AnalyticsTimeType.DAY,
  });
  Object.assign(dashboardCountSale, sale);
}

onMounted(() => {
  handleQuery();
});
</script>

<style lang="scss" scoped>
.dashboard-container {
  position: relative;
  padding: 24px;
}
</style>
