# 创建或更新飞书日历事件

适用 operation：

- `feishu.calendar.event.create`
- `feishu.calendar.event.get`
- `feishu.calendar.event.update`

适用场景：

- 创建日程
- 根据 `event_id` 更新标题、时间或地点

## 1. 创建事件

先验证：

```bash
clawrise feishu.calendar.event.create --dry-run --json '{
  "calendar_id":"cal_demo",
  "summary":"Weekly sync",
  "start_at":"2026-03-30T10:00:00+08:00",
  "end_at":"2026-03-30T11:00:00+08:00"
}'
```

再执行：

```bash
clawrise feishu.calendar.event.create --json '{
  "calendar_id":"cal_demo",
  "summary":"Weekly sync",
  "start_at":"2026-03-30T10:00:00+08:00",
  "end_at":"2026-03-30T11:00:00+08:00"
}'
```

## 2. 查询并更新事件

```bash
clawrise feishu.calendar.event.get --json '{
  "calendar_id":"cal_demo",
  "event_id":"evt_demo"
}'
```

```bash
clawrise feishu.calendar.event.update --json '{
  "calendar_id":"cal_demo",
  "event_id":"evt_demo",
  "summary":"Updated weekly sync",
  "location":"Meeting Room A"
}'
```

## 验证建议

- 时间字段使用 RFC3339
- 先运行 `clawrise spec get feishu.calendar.event.create`
- 写操作优先 `--dry-run`
