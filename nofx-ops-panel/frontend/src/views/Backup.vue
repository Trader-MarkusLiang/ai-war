<template>
  <div>
    <h2>备份管理</h2>
    <el-button type="primary" @click="createBackup" :loading="creating">创建备份</el-button>
    <el-card style="margin-top:20px">
      <el-table :data="backups">
        <el-table-column prop="file" label="文件名" />
        <el-table-column prop="size" label="大小" width="100" />
        <el-table-column prop="date" label="日期" width="150" />
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../api'

const backups = ref([])
const creating = ref(false)

async function loadBackups() {
  const res = await api.get('/backup/list')
  backups.value = res.backups
}

async function createBackup() {
  creating.value = true
  try {
    await api.post('/backup/create')
    ElMessage.success('备份创建成功')
    loadBackups()
  } catch (e) {
    ElMessage.error('备份失败')
  } finally {
    creating.value = false
  }
}

onMounted(loadBackups)
</script>
