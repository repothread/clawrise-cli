# 更新 Notion database 顶层 description

适用 operation：

- `notion.database.update`
- `notion.database.get`
- `notion.data_source.update`（仅用于 property-level description，不用于 database 顶层 description）

适用场景：

- 更新 database 顶层 description
- 更新 database 标题
- 明确区分 database 顶层 description 与 data source property description

## 1. 先确认正确入口

```bash
clawrise spec list notion
clawrise spec list notion.database
clawrise spec get notion.database.update
```

如果你要改的是 **database 顶层 description**，请使用 `notion.database.update`。
如果你要改的是某个 property 的 `description`，请继续使用 `notion.data_source.update` 并传完整 property schema patch。

## 2. 用简写字段更新 database description

```bash
clawrise notion.database.update --dry-run --json '{
  "database_id":"db_demo",
  "description":"内容主表：一条内容一行；聚合当前状态与近窗指标。"
}'
```

这里的 `description` 可以直接传字符串，Clawrise 会自动转成 Notion 需要的 rich_text 数组。

## 3. 用 provider-native body.description 更新

```bash
clawrise notion.database.update --dry-run --json '{
  "database_id":"db_demo",
  "body":{
    "description":[
      {
        "type":"text",
        "text":{
          "content":"内容主表：一条内容一行；聚合当前状态与近窗指标。"
        }
      }
    ]
  }
}'
```

当你已经在上游 payload 级别工作，或者需要自己控制 rich_text 结构时，使用这种写法。

## 4. 回读确认

```bash
clawrise notion.database.get --json '{"database_id":"db_demo"}'
```

返回结果里可以重点看：

- `description`
- `description_rich_text`
- `raw.description`

## 5. 不要把 database description 写到 notion.data_source.update.body.description

```bash
clawrise notion.data_source.update --dry-run --json '{
  "data_source_id":"ds_demo",
  "body":{
    "description":[
      {
        "type":"text",
        "text":{"content":"这会失败"}
      }
    ]
  }
}'
```

这个字段不被 Notion 的 data source update API 支持。若要更新 database 顶层 description，请切回 `notion.database.update`。

## 验证建议

- 先运行 `clawrise spec get notion.database.update`
- 不确定 payload 形状时，优先用顶层简写 `description`
- 真正写入前先加 `--dry-run`
- 写入后用 `notion.database.get` 回读确认
