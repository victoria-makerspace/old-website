package site

import ()

func init() {
	init_handler("tools", tools_handler, "/tools")
}

func tools_handler(p *page) {
	p.Title = "Tools"
}
