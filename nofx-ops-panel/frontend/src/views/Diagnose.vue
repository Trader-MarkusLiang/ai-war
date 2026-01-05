<template>
  <div>
    <h2>系统诊断</h2>
    <el-button @click="runDiagnose" :loading="loading">运行诊断</el-button>
    <el-card style="margin-top:20px">
      <el-table :data="checks">
        <el-table-column prop="name" label="检查项" width="150" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'ok' ? 'success' : (row.status === 'warning' ? 'warning' : 'danger')">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="detail" label="详情" />
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '../api'

const checks = ref([])
const loading = ref(false)

async function runDiagnose() {
  loading.value = true
  try {
    const res = await api.get('/diagnose')
    checks.value = res.checks
  } finally {
    loading.value = false
  }
}

onMounted(runDiagnose)
</script>
