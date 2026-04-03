# 安全写入 Notion block

适用 operation：

- `notion.page.create`
- `notion.block.append`
- `notion.block.update`
- `notion.block.get`
- `notion.page.markdown.get`

## 1. 先读再写

```bash
clawrise notion.block.get --json '{"block_id":"block_demo"}'
clawrise notion.page.markdown.get --json '{"page_id":"page_demo"}'
```

## 2. 先看清输入契约

```bash
clawrise spec get notion.block.append
clawrise spec get notion.block.update
```

## 3. 支持的 block 输入形态

Clawrise 现在同时支持两种 block 载荷风格：

- shorthand 顶层字段，例如 `text`、`rich_text`、`children`、`checked`
- provider-native 嵌套 block body，例如 `paragraph.rich_text`、`to_do.checked`

同一个 block 同时出现两种形态时，以顶层字段为准。

## 4. 用 shorthand 顶层字段做 dry-run

```bash
clawrise notion.block.append --dry-run --json '{
  "block_id":"page_demo",
  "children":[
    {"type":"heading_1","text":"周报"},
    {"type":"paragraph","text":"由 Clawrise 生成。"},
    {"type":"to_do","text":"上线修复","checked":false}
  ]
}'
```

## 5. 用 provider-native 嵌套 block body 做 dry-run

```bash
clawrise notion.block.append --dry-run --json '{
  "block_id":"page_demo",
  "children":[
    {
      "type":"heading_1",
      "heading_1":{
        "rich_text":[
          {"type":"text","text":{"content":"周报"}}
        ]
      }
    },
    {
      "type":"paragraph",
      "paragraph":{
        "rich_text":[
          {"type":"text","text":{"content":"由 Clawrise 生成。"}}
        ]
      }
    },
    {
      "type":"to_do",
      "to_do":{
        "checked":false,
        "rich_text":[
          {"type":"text","text":{"content":"上线修复"}}
        ]
      }
    }
  ]
}'
```

## 6. 用 provider-native body 更新单个 block

```bash
clawrise notion.block.update --dry-run --json '{
  "block_id":"block_demo",
  "block":{
    "type":"paragraph",
    "paragraph":{
      "color":"green_background",
      "rich_text":[
        {"type":"text","text":{"content":"由 Clawrise 更新。"}}
      ]
    }
  }
}'
```

## 7. 真正写入后做验证

```bash
clawrise notion.block.append --verify --json '{
  "block_id":"page_demo",
  "children":[
    {"type":"paragraph","text":"已验证写入"}
  ]
}'
```

## 8. 先上传文件，再写入 file block

```bash
clawrise notion.file_upload.create --json '{
  "mode":"single_part",
  "filename":"demo.txt",
  "content_type":"text/plain"
}'
```

拿到 `file_upload_id` 后：

```bash
clawrise notion.file_upload.send --json '{
  "file_upload_id":"fu_demo",
  "file_path":"/tmp/demo.txt",
  "content_type":"text/plain"
}'
```

然后把它挂到 block：

```bash
clawrise notion.block.append --json '{
  "block_id":"page_demo",
  "children":[
    {"type":"file","file_upload_id":"fu_demo"}
  ]
}'
```

## 使用建议

- 在载荷形态稳定之前，始终把 `--dry-run` 放在回路里
- 需要写后立即确认结果时，在支持的 Notion 写操作上加 `--verify`
- 需要查看最终上游请求和响应时，在支持的 Notion 写操作上加 `--debug-provider-payload`
- 需要写入 `image` 或 `file` block 时，优先先走 `notion.file_upload.*`，再把 `file_upload_id` 传给 block
- 如果你是在复用 Notion 读取结果或中间转换结果，保留 provider-native 嵌套 block body 是安全的
- 如果任务本质上是 markdown 优先的整页编辑，优先考虑 `notion.page.markdown.update`
