# Clawrise MVP Operation Spec

See the Chinese version at [../zh/mvp-operation-spec.md](../zh/mvp-operation-spec.md).

## 1. Purpose

This document defines the first batch of operations at an implementation-ready level.

It should now be read as the original MVP baseline, not as the complete list of currently implemented operations.

The current runtime already exceeds this baseline in several areas, including:

- additional Feishu calendar operations
- Feishu Bitable record operations
- additional Notion page and data source operations
- Notion comment operations
- a broader Notion block subset

It answers:

- which operations belong to MVP
- what JSON each operation accepts
- which fields are required
- which subject types are allowed
- which write operations require idempotency
- what normalized success output should contain

## 2. Shared Conventions

All operations use JSON input.

Field naming uses `snake_case`.

Time values use `RFC3339`.

List operations use:

- `page_size`
- `page_token`

List outputs normalize to:

- `items`
- `next_page_token`
- `has_more`

Write operations require idempotency in MVP.

Current subject rules:

- Feishu operations allow `bot`
- Notion operations allow `integration`

Additional note:

- the current runtime also supports selected Feishu document and wiki operations under `subject=user`
- this document still describes the MVP baseline rather than every current runtime extension

## 3. MVP Scope

Baseline operations:

Feishu:

- `feishu.calendar.event.create`
- `feishu.calendar.event.list`
- `feishu.docs.document.get`
- `feishu.docs.document.list_blocks`
- `feishu.docs.block.get`
- `feishu.docs.block.list_children`
- `feishu.docs.block.get_descendants`
- `feishu.docs.block.update`
- `feishu.docs.block.batch_delete`
- `feishu.wiki.space.list`
- `feishu.wiki.node.list`
- `feishu.wiki.node.create`
- `feishu.docs.document.append_blocks`
- `feishu.docs.document.get_raw_content`
- `feishu.docs.document.create`
- `feishu.docs.document.edit`

Notion:

- `notion.search.query`
- `notion.data_source.query`
- `notion.page.create`
- `notion.page.get`
- `notion.page.markdown.get`
- `notion.page.markdown.update`
- `notion.block.get`
- `notion.block.list_children`
- `notion.block.append`
- `notion.block.update`
- `notion.block.delete`

Supplementary read operations:

Feishu:

- `feishu.contact.user.get`

Notion:

- `notion.user.get`

## 4. Feishu Operations

### feishu.calendar.event.create

Purpose:

- create calendar events
- validate write path, idempotency, and time normalization

Required fields:

- `calendar_id`
- `summary`
- `start_at`
- `end_at`

Optional fields:

- `description`
- `location`
- `reminders`
- `timezone`

Allowed subject:

- `bot`

Local validation:

- `end_at` must be later than `start_at`
- `summary` must not be empty
- `attendees` are not supported by the current real implementation

Success output should include:

- `event_id`
- `calendar_id`
- `summary`
- `start_at`
- `end_at`
- `html_url`

Currently implemented request subset:

- `calendar_id`
- `summary`
- `start_at`
- `end_at`
- `description`
- `location`
- `reminders`
- `timezone`

Note:

- `attendees` are not created together with the event. According to the official Feishu API, attendees must be handled through a separate attendee API.

### feishu.calendar.event.list

Purpose:

- list events in a time range
- validate normalized list output

Required fields:

- `calendar_id`

Optional fields:

- `start_at_from`
- `start_at_to`
- `page_size`
- `page_token`

Allowed subject:

- `bot`

Success output should include:

- `items`
- `next_page_token`
- `has_more`

The Feishu document capability should use generic operation names, while the
actual actor is selected through `subject` and `profile` context:

- `feishu.docs.document.create`
- `feishu.docs.document.edit`

Recommended defaults:

- use `subject=user` for creation when user visibility matters
- use `subject=bot` for ongoing automated edits when attribution separation matters

### feishu.docs.document.create

Purpose:

- create an empty document
- recommended with `subject=user` so the resource is naturally visible in the user's scope

Required fields:

- `title`

Optional fields:

- `folder_token`

Allowed subject:

- `user`
- `bot`

MVP limitation:

- create the document only
- body editing is deferred to a future operation
- bot access still needs to be granted before the bot can edit the document

Visibility note:

- because the resource is created under user identity, it is more naturally placed in the user's visible scope
- this is exactly why creation and bot editing should be split into different operations

Success output should include:

- `document_id`
- `title`
- `folder_token`
- `url`

Recommended future extensions:

- add document sharing operations
- add editing support for existing shared documents

### feishu.wiki.space.list

Purpose:

- list wiki spaces visible to the current execution identity
- verify whether the current execution identity has already been added to the target knowledge base

Allowed subject:

- `bot`
- `user`

Notes:

- the official Feishu API does not return "My Library" in this list
- an empty result can mean the current execution identity has no wiki space access yet, not necessarily that the API failed

### feishu.wiki.node.list

Purpose:

- list child nodes under a wiki space or parent node
- discover parent node tokens and existing document nodes

Required fields:

- `space_id`

Optional fields:

- `parent_node_token`
- `page_size`
- `page_token`

Allowed subject:

- `bot`
- `user`

### feishu.wiki.node.create

Purpose:

- create a `docx` child node under a wiki parent node

Required fields:

- `space_id`

Optional fields:

- `parent_node_token`
- `title`
- `obj_type`
- `node_type`

Allowed subject:

- `bot`
- `user`

Current implementation notes:

- defaults to `docx`
- defaults to `node_type=origin`
- returns both the wiki node token and the resulting `document_id`

Precondition:

- the current execution identity must have container edit permission on the parent node

### feishu.docs.document.edit

Recommended execution modes:

- `subject=bot`: recommended default automation edit mode
- `subject=user`: use only when user attribution is explicitly desired

### feishu.docs.document.get

Purpose:

- read basic document metadata
- discover latest revision and title before structured reads

Required fields:

- `document_id`

Allowed subject:

- `bot`

Success output should include:

- `document_id`
- `revision_id`
- `title`

### feishu.docs.document.list_blocks

Purpose:

- list all blocks in a docx document
- support structured content traversal for agent workflows

Required fields:

- `document_id`

Optional fields:

- `page_size`
- `page_token`
- `document_revision_id`

Allowed subject:

- `bot`

Success output should include:

- `items`
- `next_page_token`
- `has_more`

### feishu.docs.block.get

Purpose:

- read one docx block
- support precise structured content inspection

Required fields:

- `document_id`
- `block_id`

Optional fields:

- `document_revision_id`

Allowed subject:

- `bot`

Success output should include:

- `block_id`
- `block_type`
- `block_type_name`
- `plain_text`

### feishu.docs.block.list_children

Purpose:

- list children under one docx block
- support structured traversal by subtree

Required fields:

- `document_id`
- `block_id`

Optional fields:

- `page_size`
- `page_token`
- `document_revision_id`
- `with_descendants`

Allowed subject:

- `bot`

Success output should include:

- `items`
- `next_page_token`
- `has_more`

### feishu.docs.block.get_descendants

Purpose:

- read all descendants under a docx block
- support subtree traversal in one operation

Required fields:

- `document_id`
- `block_id`

Optional fields:

- `page_size`
- `page_token`
- `document_revision_id`

Allowed subject:

- `bot`

Success output should include:

- `items`
- `next_page_token`
- `has_more`

### feishu.docs.block.update

Purpose:

- update one docx block
- support precise structured content editing

Required fields:

- `document_id`
- `block_id`

Current implemented input subset:

- text content update through `text`
- explicit `update_task` passthrough

Optional fields:

- `document_revision_id`

Allowed subject:

- `bot`

Success output should include:

- `block_id`
- `block_type_name`
- `plain_text`
- `document_revision_id`

### feishu.docs.block.batch_delete

Purpose:

- delete a range of child blocks under a parent block
- support structured subtree cleanup

Required fields:

- `document_id`
- `block_id`
- `start_index`
- `end_index`

Optional fields:

- `document_revision_id`

Allowed subject:

- `bot`

Success output should include:

- `document_id`
- `block_id`
- `start_index`
- `end_index`
- `document_revision_id`

### feishu.docs.document.append_blocks

Purpose:

- append block content to an existing docx document

Required fields:

- `document_id`
- `blocks`

Optional fields:

- `block_id`

Allowed subject:

- `bot`

Currently implemented block subset:

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

Implementation notes:

- appends to the document root by default
- when `block_id` is omitted, `document_id` is used as the root block id

### feishu.docs.document.get_raw_content

Purpose:

- read pure text content from a docx document

Required fields:

- `document_id`

Allowed subject:

- `bot`

### feishu.contact.user.get

Supplementary read operation.

Required fields:

- `user_id`

Allowed subject:

- `bot`

## 5. Notion Operations

### notion.page.create

Purpose:

- create a page
- validate structured content writes and idempotency

Required fields:

- `parent.type`
- `parent.id`
- `title`

Optional fields:

- `properties`
- `children`

Notes:

- `children` accepts both shorthand top-level block fields and provider-native nested block bodies
- when both shapes are present on the same block, the top-level fields take precedence

Allowed subject:

- `integration`

MVP block subset:

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

Success output should include:

- `page_id`
- `title`
- `parent`
- `url`

### notion.search.query

Purpose:

- search pages and data sources visible to the integration
- provide a general-purpose discovery entry for agent workflows

Optional fields:

- `query`
- `filter`
- `sort`
- `page_size`
- `page_token`

Allowed subject:

- `integration`

Success output should include:

- `items`
- `next_page_token`
- `has_more`

### notion.data_source.query

Purpose:

- query pages and nested data sources under one data source
- support structured filtering and sorting workflows

Required fields:

- `data_source_id`

Optional fields:

- `filter`
- `sorts`
- `page_size`
- `page_token`
- `filter_properties`

Allowed subject:

- `integration`

Success output should include:

- `data_source_id`
- `items`
- `next_page_token`
- `has_more`

### notion.page.get

Purpose:

- read page details
- validate normalized object output

Required fields:

- `page_id`

Allowed subject:

- `integration`

Success output should include:

- `page_id`
- `title`
- `url`
- `archived`
- `properties`

### notion.page.markdown.get

Purpose:

- read enhanced markdown for a page
- support agent-friendly content inspection

Required fields:

- `page_id`

Optional fields:

- `include_transcript`

Allowed subject:

- `integration`

Success output should include:

- `page_id`
- `markdown`
- `truncated`
- `unknown_block_ids`

### notion.page.markdown.update

Purpose:

- update page content through enhanced markdown commands
- support agent-friendly content editing

Required fields:

- `page_id`
- one markdown update command

Supported command types:

- `update_content`
- `replace_content`
- `insert_content`
- `replace_content_range`

Allowed subject:

- `integration`

Success output should include:

- `page_id`
- `markdown`
- `truncated`
- `unknown_block_ids`

### notion.block.append

Purpose:

- append blocks to a page or block
- validate structured writes and idempotency

Required fields:

- `block_id`
- `children`

Notes:

- `children` accepts both shorthand top-level block fields and provider-native nested block bodies
- when both shapes are present on the same block, the top-level fields take precedence

Allowed subject:

- `integration`

Success output should include:

- `block_id`
- `appended_count`
- appended child identifiers

### notion.block.get

Purpose:

- read a single block
- normalize block-level content metadata

Required fields:

- `block_id`

Allowed subject:

- `integration`

Success output should include:

- `block_id`
- `type`
- `has_children`
- `plain_text`

### notion.block.list_children

Purpose:

- list direct children under a block
- support structured content traversal

Required fields:

- `block_id`

Optional fields:

- `page_size`
- `page_token`

Allowed subject:

- `integration`

Success output should include:

- `items`
- `next_page_token`
- `has_more`

### notion.block.update

Purpose:

- update one block in place
- support precise structured content editing

Required fields:

- `block_id`
- block payload

Notes:

- the block payload may be provided directly at the top level or under `block`
- textual and structured block fields accept both shorthand top-level fields and provider-native nested block bodies
- when both shapes are present on the same block, the top-level fields take precedence

Allowed subject:

- `integration`

Success output should include:

- `block_id`
- `type`
- `plain_text`

### notion.block.delete

Purpose:

- archive one block
- support structured content removal

Required fields:

- `block_id`

Optional fields:

- `allow_child_page_delete`

Allowed subject:

- `integration`

Safety notes:

- deleting a `child_page` block can archive/trash the underlying Notion page, not merely remove one visual entry
- keep `allow_child_page_delete` unset unless you intentionally want to archive that page
- if the integration lacks Notion read content capability, the adapter cannot inspect the target type and will require `allow_child_page_delete=true` before deleting

Success output should include:

- `block_id`
- `archived`
- `in_trash`

### notion.user.get

Supplementary read operation.

Required fields:

- `user_id`

Allowed subject:

- `integration`

## 6. Out of Scope

The following are out of MVP scope:

- full rich-text styling coverage
- file upload
- image upload
- table blocks
- complex Feishu document body editing
- full Notion database query DSL mapping
- transactional multi-write flows
- cross-platform workflow orchestration
