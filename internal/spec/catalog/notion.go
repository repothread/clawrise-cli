package catalog

func notionEntries() []Entry {
	return entriesFromOperations([]string{
		"notion.block.append",
		"notion.block.delete",
		"notion.block.get",
		"notion.block.list_children",
		"notion.block.update",
		"notion.data_source.query",
		"notion.page.create",
		"notion.page.get",
		"notion.page.markdown.get",
		"notion.page.markdown.update",
		"notion.search.query",
		"notion.user.get",
	})
}
