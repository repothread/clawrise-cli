# Create or Update a Feishu Calendar Event

Operations:

- `feishu.calendar.event.create`
- `feishu.calendar.event.get`
- `feishu.calendar.event.update`

## Create an event

```bash
clawrise feishu.calendar.event.create --dry-run --json '{
  "calendar_id":"cal_demo",
  "summary":"Weekly sync",
  "start_at":"2026-03-30T10:00:00+08:00",
  "end_at":"2026-03-30T11:00:00+08:00"
}'
```

## Update an event

```bash
clawrise feishu.calendar.event.update --json '{
  "calendar_id":"cal_demo",
  "event_id":"evt_demo",
  "summary":"Updated weekly sync",
  "location":"Meeting Room A"
}'
```
