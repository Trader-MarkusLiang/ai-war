import { ref } from 'vue'

// 全局加载状态
export const loading = ref(false)
export const loadingText = ref('加载中...')

// 设置加载状态
export function setLoading(isLoading, text = '加载中...') {
  loading.value = isLoading
  loadingText.value = text
}

// 全局错误状态
export const error = ref(null)

// 设置错误信息
export function setError(errorMsg) {
  error.value = errorMsg
  if (errorMsg) {
    setTimeout(() => {
      error.value = null
    }, 5000)
  }
}
