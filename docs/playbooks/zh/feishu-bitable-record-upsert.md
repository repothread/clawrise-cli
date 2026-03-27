# 创建或更新飞书多维表格记录

适用 operation：

- `feishu.bitable.record.list`
- `feishu.bitable.record.create`
- `feishu.bitable.record.update`

适用场景：

- 向多维表格写入新任务
- 根据已有记录 ID 更新状态

## 1. 先查记录

```bash
clawrise feishu.bitable.record.list --json '{
  "app_token":"app_demo",
  "table_id":"tbl_demo",
  "page_size":20
}'
```

## 2. 创建记录

先验证：

```bash
clawrise feishu.bitable.record.create --dry-run --json '{
  "app_token":"app_demo",
  "table_id":"tbl_demo",
  "fields":{
    "Title":"任务 A",
    "Status":"Todo"
  }
}'
```

再执行：

```bash
clawrise feishu.bitable.record.create --json '{
  "app_token":"app_demo",
  "table_id":"tbl_demo",
  "fields":{
    "Title":"任务 A",
    "Status":"Todo"
  }
}'
```

## 3. 更新记录

```bash
clawrise feishu.bitable.record.update --json '{
  "app_token":"app_demo",
  "table_id":"tbl_demo",
  "record_id":"rec_demo",
  "fields":{
    "Status":"Done"
  }
}'
```

## 验证建议

- 先运行 `clawrise spec get feishu.bitable.record.create`
- 不确定字段名时，先用 `record.list` 看已有记录返回结构
- 对写操作优先使用 `--dry-run`
