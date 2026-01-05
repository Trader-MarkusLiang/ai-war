<template>
  <div>
    <h2>服务管理</h2>
    <el-card>
      <el-space>
        <el-button type="success" @click="action('start')" :loading="loading">启动服务</el-button>
        <el-button type="warning" @click="action('restart')" :loading="loading">重启服务</el-button>
        <el-button type="danger" @click="confirmStop" :loading="loading">停止服务</el-button>
      </el-space>
    </el-card>
    <el-card style="margin-top:20px">
      <template #header>容器状态</template>
      <el-table :data="containers">
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="status" label="状态" />
        <el-table-column prop="memory" label="内存" />
        <el-table-column prop="cpu" label="CPU" />
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../api'

const containers = ref([])
const loading = ref(false)

async function loadStatus() {
  const res = await api.get('/status')
  containers.value = res.containers
}

async function action(type) {
  loading.value = true
  try {
    await api.post(`/service/${type}`, { confirm: true })
    ElMessage.success('操作成功')
    setTimeout(loadStatus, 2000)
  } catch (e) {
    ElMessage.error('操作失败')
  } finally {
    loading.value = false
  }
}

function confirmStop() {
  ElMessageBox.confirm('确定要停止服务吗？', '警告', { type: 'warning' })
    .then(() => action('stop'))
}

onMounted(loadStatus)
</script>
