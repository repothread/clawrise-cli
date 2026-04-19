# Notion Operation Map

Use this reference when the user describes a Notion task but does not know the exact Clawrise operation name yet.

## Search And Target Resolution

- `notion.search.query`: search pages, databases, and data sources across the visible workspace
- `notion.task.database.resolve_target`: resolve a Notion URL, raw id, page id, database id, or data source id into database and data source context

## Page Primitives

- `notion.page.create`: create one page with provider-native properties, child blocks, markdown, template, and page insertion position
- `notion.page.get`: read one page with optional property filtering
- `notion.page.property_item.get`: read one property item with pagination when a page property is partially expanded
- `notion.page.update`: update page properties, lock state, trash state, template, icon, or cover
- `notion.page.move`: move a regular page under another page or into a data source
- `notion.page.markdown.get`: read page content as markdown
- `notion.page.markdown.update`: mutate page content through the Notion markdown page API

## Page Workflows

- `notion.task.page.import_markdown`: create one child page from markdown text or one local Markdown file
- `notion.task.page.upsert_markdown_child`: find a child page by exact title under one parent and replace or create it from Markdown
- `notion.task.page.patch_section`: replace one section under a page by exact heading or heading path
- `notion.task.page.ensure_sections`: ensure one page contains a set of heading sections, creating only the missing ones
- `notion.task.page.append_under_heading`: append markdown content under one heading, optionally creating that heading first
- `notion.task.page.find_or_create_by_path`: resolve a root context and then find or create a nested page path
- `notion.task.page.read_complete`: read one page with full property items plus markdown expansion
- `notion.task.page.read_graph`: read one page plus related pages discovered through relation properties

## Block Primitives

- `notion.block.get`: read one block
- `notion.block.list_children`: list direct child blocks under one block
- `notion.block.get_descendants`: recursively collect all descendant blocks
- `notion.block.append`: append child blocks under one block
- `notion.block.update`: update one block body
- `notion.block.delete`: archive one block (`child_page` targets can archive the underlying page; use with care)

## File Uploads And Attachments

- `notion.task.block.attach_file`: upload one local file or base64 payload and append it as an image or file block in one step
- `notion.file_upload.create`: create a file upload slot in single-part, multi-part, or external URL mode
- `notion.file_upload.send`: send one upload part from a local file or base64 payload
- `notion.file_upload.complete`: finalize a multi-part upload
- `notion.file_upload.get`: inspect one file upload object
- `notion.file_upload.list`: list visible file uploads

## Database And Data Source Primitives

- `notion.database.get`: read one database and expose its child data sources
- `notion.database.create`: create one database through shorthand fields or a provider-native body
- `notion.database.update`: update one database through shorthand fields or a provider-native body
- `notion.data_source.get`: read one data source schema
- `notion.data_source.template.list`: list templates available under one data source
- `notion.data_source.create`: create one data source through a provider-native body
- `notion.data_source.update`: update one data source through a provider-native body
- `notion.data_source.query`: query rows inside one data source

## Data Source Workflows

- `notion.task.data_source.row.upsert`: find one row with a provider-native match filter and update or create it
- `notion.task.data_source.bulk_upsert`: apply row upsert semantics to many items and return per-item results
- `notion.task.data_source.schema.ensure`: add missing data source properties or missing select-like options without rewriting the whole schema

## Comments, Users, And Meeting Notes

- `notion.comment.get`: read one comment object
- `notion.comment.list`: list open comments under a page or block
- `notion.comment.create`: create a comment on a page, block, or discussion thread
- `notion.user.get`: read one user object, including `user_id=me`
- `notion.user.list`: list users visible to the current integration
- `notion.task.meeting_notes.get`: read one meeting notes block, or discover meeting notes blocks under a page and fetch summary, notes, and transcript sections

## Routing Heuristics

- Choose low-level `notion.page.*`, `notion.block.*`, `notion.database.*`, and `notion.data_source.*` operations when the user already knows the provider-native payload shape they want.
- Choose `notion.task.*` operations when the task is expressed in terms of markdown files, headings, page paths, schema sync, row upsert, file attachment, or meeting notes extraction.
- Choose `notion.task.database.resolve_target` before row or schema work when the user only has a URL, a page id, or an ambiguous database or data source reference.
