# difflite

终端里的语法高亮 `git diff` 查看器。管道即用，无需配置。

## 安装

```bash
go install github.com/freedom090/difflite@latest
```

## 使用

```bash
git diff | difflite
git diff --cached | difflite
git diff main..feature | difflite
git show | difflite --no-collapse
```

- 新增行：绿色背景 + 语法高亮
- 删除行：红色背景 + 语法高亮
- 上下文行：深色背景，长段自动折叠
- `--no-collapse`：显示全部上下文，不折叠
