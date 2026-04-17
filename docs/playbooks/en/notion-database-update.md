# Update a Notion database description

Operations:

- `notion.database.update`
- `notion.database.get`
- `notion.data_source.update` (only for property-level descriptions, not the database-level description)

Use this flow when you need to:

- update a database's top-level description
- update a database title
- distinguish database-level description changes from data source property description changes

## 1. Confirm the correct entry point

```bash
clawrise spec list notion
clawrise spec list notion.database
clawrise spec get notion.database.update
```

Use `notion.database.update` for the **database-level description**.
Use `notion.data_source.update` only when patching a property's own `description` field.

## 2. Update the database description with the shorthand field

```bash
clawrise notion.database.update --dry-run --json '{
  "database_id":"db_demo",
  "description":"Content master table: one row per item, aggregating current state and rolling metrics."
}'
```

The top-level `description` field accepts a plain string and Clawrise will translate it into the rich_text array required by Notion.

## 3. Update the database description with provider-native `body.description`

```bash
clawrise notion.database.update --dry-run --json '{
  "database_id":"db_demo",
  "body":{
    "description":[
      {
        "type":"text",
        "text":{
          "content":"Content master table: one row per item, aggregating current state and rolling metrics."
        }
      }
    ]
  }
}'
```

Use this form when you are already working directly with the upstream Notion payload shape.

## 4. Read the database back

```bash
clawrise notion.database.get --json '{"database_id":"db_demo"}'
```

Focus on these fields in the response:

- `description`
- `description_rich_text`
- `raw.description`

## 5. Do not send the database description through `notion.data_source.update.body.description`

```bash
clawrise notion.data_source.update --dry-run --json '{
  "data_source_id":"ds_demo",
  "body":{
    "description":[
      {
        "type":"text",
        "text":{"content":"This will fail"}
      }
    ]
  }
}'
```

That field is not supported by Notion's data source update API. Switch back to `notion.database.update` when you need to change the database-level description.

## Verification tips

- Run `clawrise spec get notion.database.update` first
- Prefer the shorthand `description` field if you are unsure about the raw payload shape
- Use `--dry-run` before sending the live request
- Read the database back with `notion.database.get` after the write
