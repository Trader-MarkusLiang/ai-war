<template>
  <div>
    <h2>日志查看</h2>
    <el-card>
      <el-space>
        <el-select v-model="service" @change="loadLogs">
          <el-option label="后端" value="backend" />
          <el-option label="前端" value="frontend" />
        </el-select>
        <el-button @click="loadLogs">刷新</el-button>
      </el-space>
      <div class="log-box">
        <pre>{{ logs }}</pre>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '../api'

const service = ref('backend')
const logs = ref('')

async function loadLogs() {
  const res = await api.get(`/logs/${service.value}?lines=200`)
  logs.value = res.logs.join('\n')
}

onMounted(loadLogs)
</script>

<style scoped>
.log-box { margin-top: 15px; background: #1e1e1e; color: #d4d4d4; padding: 15px; border-radius: 4px; height: 500px; overflow: auto; }
pre { margin: 0; font-size: 12px; white-space: pre-wrap; word-break: break-all; }
</style>
