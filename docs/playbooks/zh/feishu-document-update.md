# 更新飞书文档

适用 operation：

- `feishu.docs.document.edit`
- `feishu.docs.document.get`
- `feishu.docs.document.get_raw_content`

适用场景：

- 用任务导向方式替换整篇文档内容
- 向现有文档末尾追加结构化 block
- 在真正写入前先读取当前文档内容

## 1. 先看文档当前内容

```bash
clawrise feishu.docs.document.get_raw_content --json '{"document_id":"doxcnDemo"}'
```

## 2. 用 `replace_all` 全量替换

先验证：

```bash
clawrise feishu.docs.document.edit --dry-run --json '{
  "document_id":"doxcnDemo",
  "mode":"replace_all",
  "blocks":[
    {"type":"heading_1","text":"周报"},
    {"type":"paragraph","text":"由 Clawrise 生成。"}
  ]
}'
```

再执行：

```bash
clawrise feishu.docs.document.edit --json '{
  "document_id":"doxcnDemo",
  "mode":"replace_all",
  "blocks":[
    {"type":"heading_1","text":"周报"},
    {"type":"paragraph","text":"由 Clawrise 生成。"}
  ]
}'
```

## 3. 用 `append` 追加内容

```bash
clawrise feishu.docs.document.edit --json '{
  "document_id":"doxcnDemo",
  "mode":"append",
  "blocks":[
    {"type":"paragraph","text":"新增一段正文。"}
  ]
}'
```

## 验证建议

- 先运行 `clawrise spec get feishu.docs.document.edit`
- 写入前优先使用 `--dry-run`
- 写入后再调用一次 `feishu.docs.document.get_raw_content`

## 注意事项

- `create` 只保证创建成功，不默认保证目标用户马上可见
- `replace_all` 会替换文档根节点下的直接子节点
- 若文档由 bot 创建，后续仍可能需要单独分享或授权
