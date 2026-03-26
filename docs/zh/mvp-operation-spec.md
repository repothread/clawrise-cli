# Clawrise MVP Operation 规格

英文版见 [../en/mvp-operation-spec.md](../en/mvp-operation-spec.md)。

授权模型见 [auth-model.md](auth-model.md)。

## 1. 文档目的

这份文档把 MVP 阶段的 operation 收敛到“可直接编码”的粒度。

它回答以下问题：

- MVP 到底做哪些 operation
- 每个 operation 接受什么 JSON 输入
- 哪些字段必填
- 哪些主体允许调用
- 哪些写操作必须启用幂等
- 成功时标准化输出应包含哪些核心字段

## 2. 统一约定

### 2.1 输入格式

所有执行命令统一使用 JSON 输入：

```bash
clawrise feishu.calendar.event.create --input @event.json
clawrise notion.page.create --json '{"title":"项目记录"}'
```

### 2.2 字段命名

统一使用 `snake_case`。

示例：

- `calendar_id`
- `page_id`
- `start_at`
- `page_size`

### 2.3 时间格式

所有时间字段统一使用 `RFC3339` 字符串。

示例：

```text
2026-03-30T10:00:00+08:00
2026-03-30T02:00:00Z
```

### 2.4 分页格式

所有列表型命令统一使用：

- `page_size`
- `page_token`

统一输出：

- `items`
- `next_page_token`
- `has_more`

### 2.5 幂等约定

规则如下：

- 读操作默认不要求 `idempotency_key`
- 写操作在 MVP 默认要求幂等
- 如果用户未显式传入，运行时可按稳定输入计算指纹

### 2.6 调用主体约定

每个 operation 在运行时都必须校验允许的主体类型。

当前 MVP 规则：

- Feishu operation 允许 `bot`
- Notion operation 允许 `integration`

### 2.7 Notion Block 子集

为了控制范围，MVP 只支持以下 block 类型：

- `paragraph`
- `heading_1`
- `heading_2`
- `heading_3`
- `bulleted_list_item`
- `numbered_list_item`
- `to_do`
- `quote`
- `code`
- `divider`

## 3. MVP 范围

### 3.1 P0 必做

Feishu：

- `feishu.calendar.event.create`
- `feishu.calendar.event.list`
- `feishu.wiki.space.list`
- `feishu.wiki.node.list`
- `feishu.wiki.node.create`
- `feishu.docs.document.append_blocks`
- `feishu.docs.document.get_raw_content`
- `feishu.docs.document.create`
- `feishu.docs.document.edit`

Notion：

- `notion.page.create`
- `notion.page.get`
- `notion.block.get`
- `notion.block.list_children`
- `notion.block.append`
- `notion.block.update`
- `notion.block.delete`

### 3.2 P1 可后置

Feishu：

- `feishu.contact.user.get`

Notion：

- `notion.user.get`

## 4. Feishu Operation 规格

### 4.1 feishu.calendar.event.create

用途：

- 创建日历事件
- 验证写操作、幂等、时间标准化

必填字段：

- `calendar_id`
- `summary`
- `start_at`
- `end_at`

可选字段：

- `description`
- `location`
- `reminders`
- `timezone`

允许主体：

- `bot`

本地校验：

- `end_at` 必须晚于 `start_at`
- `summary` 不能为空
- 当前真实实现不支持 `attendees`

成功输出建议包含：

- `event_id`
- `calendar_id`
- `summary`
- `start_at`
- `end_at`
- `html_url`

当前真实实现支持的创建字段子集：

- `calendar_id`
- `summary`
- `start_at`
- `end_at`
- `description`
- `location`
- `reminders`
- `timezone`

说明：

- `attendees` 不会在创建日程时一并写入。根据飞书官方 API 说明，参与人需要通过单独的日程参与人接口处理。

### 4.2 feishu.calendar.event.list

用途：

- 按时间窗口查询事件
- 验证统一列表输出

必填字段：

- `calendar_id`

可选字段：

- `start_at_from`
- `start_at_to`
- `page_size`
- `page_token`

允许主体：

- `bot`

成功输出建议包含：

- `items`
- `next_page_token`
- `has_more`

### 4.3 feishu.docs.document.create

用途：

- 创建空文档
- 推荐在 `subject=user` 时使用，以便资源天然位于用户可见范围内

必填字段：

- `title`

可选字段：

- `folder_token`

允许主体：

- `user`
- `bot`

MVP 限制：

- 只创建文档本体
- 不在本 operation 中直接写入正文
- 后续仍需单独为 bot 授权，才能让 bot 编辑该文档

可见性说明：

- 由于资源由用户身份创建，因此默认更容易处于用户可见范围
- 这正是推荐把文档创建与 bot 编辑拆开的原因

成功输出建议包含：

- `document_id`
- `title`
- `folder_token`
- `url`

未来扩展建议：

- 增加文档共享类 operation
- 增加对已有共享文档的编辑能力

### 4.4 feishu.wiki.space.list

用途：

- 列出当前 bot 可访问的知识空间
- 验证 bot 是否已经被加入目标知识库

允许主体：

- `bot`

说明：

- 飞书官方说明，该接口不会返回“我的文档库”
- 如果返回空列表，不一定表示接口异常，也可能表示 bot 还没有任何知识空间访问权限

### 4.5 feishu.wiki.node.list

用途：

- 列出指定知识空间下的子节点
- 用于定位父节点 token 和已有文档节点

必填字段：

- `space_id`

可选字段：

- `parent_node_token`
- `page_size`
- `page_token`

允许主体：

- `bot`

### 4.6 feishu.wiki.node.create

用途：

- 在知识库指定父节点下创建一个 `docx` 子节点

必填字段：

- `space_id`

可选字段：

- `parent_node_token`
- `title`
- `obj_type`
- `node_type`

允许主体：

- `bot`

当前实现说明：

- 默认创建 `docx`
- 默认 `node_type=origin`
- 成功后会返回知识库节点 token 和对应的 `document_id`

前置条件：

- bot 对父节点具备容器编辑权限

### 4.7 feishu.docs.document.edit

推荐执行模式：

- `subject=bot`：推荐默认自动化编辑模式
- `subject=user`：仅在确实需要用户归因时使用

### 4.8 feishu.docs.document.append_blocks

用途：

- 向已有 docx 文档追加内容块

必填字段：

- `document_id`
- `blocks`

可选字段：

- `block_id`

允许主体：

- `bot`

当前实现支持的块类型子集：

- `paragraph`
- `heading_1`
- `heading_2`
- `heading_3`
- `bulleted_list_item`
- `numbered_list_item`
- `quote`
- `to_do`
- `code`
- `divider`

实现说明：

- 默认向文档根节点追加
- 如不传 `block_id`，会使用 `document_id` 作为根块

### 4.9 feishu.docs.document.get_raw_content

用途：

- 读取 docx 文档纯文本内容

必填字段：

- `document_id`

允许主体：

- `bot`

### 4.10 feishu.contact.user.get

P1 读操作。

必填字段：

- `user_id`

允许主体：

- `bot`

## 5. Notion Operation 规格

### 5.1 notion.page.create

用途：

- 创建页面
- 验证结构化内容写入与幂等

必填字段：

- `parent.type`
- `parent.id`
- `title`

可选字段：

- `properties`
- `children`

允许主体：

- `integration`

成功输出建议包含：

- `page_id`
- `title`
- `parent`
- `url`

### 5.2 notion.page.get

用途：

- 读取页面详情
- 验证对象标准化输出

必填字段：

- `page_id`

允许主体：

- `integration`

成功输出建议包含：

- `page_id`
- `title`
- `url`
- `archived`
- `properties`

### 5.3 notion.block.append

用途：

- 向页面或 block 追加内容
- 验证结构化写入和幂等

必填字段：

- `block_id`
- `children`

允许主体：

- `integration`

成功输出建议包含：

- `block_id`
- `appended_count`
- 追加后的子 block 标识

### 5.4 notion.block.get

用途：

- 读取单个 block
- 验证 block 级内容标准化输出

必填字段：

- `block_id`

允许主体：

- `integration`

成功输出建议包含：

- `block_id`
- `type`
- `has_children`
- `plain_text`

### 5.5 notion.block.list_children

用途：

- 列出指定 block 下的直接子 block
- 支持结构化正文遍历

必填字段：

- `block_id`

可选字段：

- `page_size`
- `page_token`

允许主体：

- `integration`

成功输出建议包含：

- `items`
- `next_page_token`
- `has_more`

### 5.6 notion.block.update

用途：

- 原地更新单个 block
- 支持精细化结构化编辑

必填字段：

- `block_id`
- block 内容载荷

允许主体：

- `integration`

成功输出建议包含：

- `block_id`
- `type`
- `plain_text`

### 5.7 notion.block.delete

用途：

- 归档单个 block
- 支持结构化内容删除

必填字段：

- `block_id`

允许主体：

- `integration`

成功输出建议包含：

- `block_id`
- `archived`
- `in_trash`

### 5.8 notion.user.get

P1 读操作。

必填字段：

- `user_id`

允许主体：

- `integration`

## 6. MVP 暂不支持

以下能力不纳入 MVP：

- 富文本全量样式映射
- 文件上传
- 图片上传
- 表格 block
- 飞书文档复杂正文编辑
- Notion 数据库查询 DSL 的完整映射
- 事务型批量写入
- 跨平台工作流编排

## 7. 推荐实施顺序

1. 统一输入读取、输出包络和错误模型
2. `feishu.calendar.event.create`
3. `feishu.calendar.event.list`
4. `notion.page.create`
5. `notion.page.get`
6. 幂等与审计存储
7. `notion.block.append`
8. `feishu.docs.document.create`
9. P1 读操作
