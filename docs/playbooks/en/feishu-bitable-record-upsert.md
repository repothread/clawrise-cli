# Create or Update a Feishu Bitable Record

Operations:

- `feishu.bitable.record.list`
- `feishu.bitable.record.create`
- `feishu.bitable.record.update`

## List current records

```bash
clawrise feishu.bitable.record.list --json '{
  "app_token":"app_demo",
  "table_id":"tbl_demo",
  "page_size":20
}'
```

## Create a record

```bash
clawrise feishu.bitable.record.create --dry-run --json '{
  "app_token":"app_demo",
  "table_id":"tbl_demo",
  "fields":{
    "Title":"Task A",
    "Status":"Todo"
  }
}'
```

## Update a record

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
