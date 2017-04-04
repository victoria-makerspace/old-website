package site

import ()

func init() {
	init_handler("/tools", "tools", tools_handler)
}

func tools_handler(p *page) {
	p.Title = "Tools"
}
