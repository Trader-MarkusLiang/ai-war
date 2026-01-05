# 贡献指南

感谢您对 NOFX 运维管理面板的关注！

## 如何贡献

### 报告问题

如果您发现了 bug 或有功能建议：

1. 检查是否已有相关 issue
2. 创建新 issue，描述清楚问题或建议
3. 提供复现步骤（如果是 bug）

### 提交代码

1. Fork 本项目
2. 创建功能分支：`git checkout -b feat/your-feature`
3. 提交代码：`git commit -m "feat: 添加新功能"`
4. 推送分支：`git push origin feat/your-feature`
5. 创建 Pull Request

## 开发规范

### 代码风格

**Python (后端)**
- 遵循 PEP 8 规范
- 使用类型注解
- 添加必要的文档字符串

**JavaScript (前端)**
- 使用 ES6+ 语法
- 遵循 Vue 3 最佳实践
- 使用有意义的变量名

### 提交信息规范

使用约定式提交格式：

```
<type>(<scope>): <subject>
```

**类型 (type)**
- `feat`: 新功能
- `fix`: 修复 bug
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 重构
- `test`: 测试相关
- `chore`: 构建/工具相关

**示例**
```
feat(api): 添加状态查询缓存
fix(ssh): 修复连接池泄漏问题
docs(readme): 更新部署说明
```

