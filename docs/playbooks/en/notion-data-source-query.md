# Query a Notion Data Source

Operations:

- `notion.data_source.get`
- `notion.data_source.template.list`
- `notion.data_source.query`

## Read schema

```bash
clawrise notion.data_source.get --json '{
  "data_source_id":"ds_demo"
}'
```

## List available templates

```bash
clawrise notion.data_source.template.list --json '{
  "data_source_id":"ds_demo",
  "page_size":20
}'
```

## Run a basic query

```bash
clawrise notion.data_source.query --json '{
  "data_source_id":"ds_demo",
  "page_size":20
}'
```

## Query with filter and sorts

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
