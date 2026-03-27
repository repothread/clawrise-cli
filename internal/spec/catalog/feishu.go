package catalog

func feishuEntries() []Entry {
	return entriesFromOperations([]string{
		"feishu.calendar.event.create",
		"feishu.calendar.event.list",
		"feishu.contact.user.get",
		"feishu.docs.block.batch_delete",
		"feishu.docs.block.get",
		"feishu.docs.block.get_descendants",
		"feishu.docs.block.list_children",
		"feishu.docs.block.update",
		"feishu.docs.document.append_blocks",
		"feishu.docs.document.create",
		"feishu.docs.document.get",
		"feishu.docs.document.get_raw_content",
		"feishu.docs.document.list_blocks",
		"feishu.wiki.node.create",
		"feishu.wiki.node.list",
		"feishu.wiki.space.list",
	})
}
