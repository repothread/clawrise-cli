# 查询 Notion data source

适用 operation：

- `notion.data_source.get`
- `notion.data_source.query`

适用场景：

- 读取 data source schema
- 分页查询记录
- 在正式构造 filter 和 sorts 前先验证数据结构

## 1. 先看 schema

```bash
clawrise notion.data_source.get --json '{
  "data_source_id":"ds_demo"
}'
```

## 2. 基础查询

```bash
clawrise notion.data_source.query --json '{
  "data_source_id":"ds_demo",
  "page_size":20
}'
```

## 3. 带筛选和排序的查询

```bash
clawrise notion.data_source.query --json '{
  "data_source_id":"ds_demo",
  "filter":{
    "property":"Status",
    "status":{"equals":"In Progress"}
  },
  "sorts":[
    {"property":"Last edited time","direction":"descending"}
  ],
  "page_size":20
}'
```

## 验证建议

- 先运行 `clawrise spec get notion.data_source.query`
- 先 `get` 再 `query`，避免 property 名写错
- 查询是读操作，不需要显式 `idempotency_key`
