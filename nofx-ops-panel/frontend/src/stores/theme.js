import { ref, watch } from 'vue'

const isDark = ref(localStorage.getItem('theme') === 'dark')

watch(isDark, (val) => {
  localStorage.setItem('theme', val ? 'dark' : 'light')
  document.documentElement.setAttribute('data-theme', val ? 'dark' : 'light')
})

// 初始化
if (isDark.value) {
  document.documentElement.setAttribute('data-theme', 'dark')
}

export function useTheme() {
  const toggleTheme = () => {
    isDark.value = !isDark.value
  }
  return { isDark, toggleTheme }
}
