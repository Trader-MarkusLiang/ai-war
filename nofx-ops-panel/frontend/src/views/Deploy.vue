<template>
  <div>
    <h2>部署管理</h2>
    <el-card>
      <el-space>
        <el-button type="primary" @click="restart" :loading="loading">重启服务</el-button>
      </el-space>
      <div v-if="output" class="output-box">
        <pre>{{ output }}</pre>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'

const loading = ref(false)
const output = ref('')

async function restart() {
  loading.value = true
  output.value = '正在重启服务...\n'
  try {
    const res = await api.post('/service/restart', {})
    output.value += res.output || '重启完成'
    ElMessage.success('操作成功')
  } catch (e) {
    output.value += '操作失败'
    ElMessage.error('操作失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.output-box { margin-top: 15px; background: #1e1e1e; color: #0f0; padding: 15px; border-radius: 4px; max-height: 400px; overflow: auto; }
pre { margin: 0; font-size: 12px; }
</style>
