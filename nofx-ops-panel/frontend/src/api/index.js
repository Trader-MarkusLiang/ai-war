import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000, // 30秒超时
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器
api.interceptors.request.use(
  config => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  error => {
    console.error('请求错误:', error)
    return Promise.reject(error)
  }
)

// 响应拦截器（优化错误处理）
api.interceptors.response.use(
  res => res.data,
  err => {
    const status = err.response?.status
    const message = err.response?.data?.detail || err.message

    // 处理不同的错误状态
    if (status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    } else if (status === 403) {
      console.error('权限不足')
    } else if (status === 404) {
      console.error('资源不存在')
    } else if (status >= 500) {
      console.error('服务器错误:', message)
    } else if (err.code === 'ECONNABORTED') {
      console.error('请求超时')
    }

    return Promise.reject(err)
  }
)

export default api
