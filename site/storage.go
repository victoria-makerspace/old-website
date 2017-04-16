package site

import ()

func init() {
	init_handler("storage", storage_handler, "/member/storage")
}

func storage_handler(p *page) {
	p.Title = "Storage"
	if !p.must_authenticate() {
		return
	}
	p.Data["hall_lockers"] = p.List_storage(p.Plans["storage-locker-hallway"].ID)
	p.Data["bathroom_lockers"] = p.List_storage("storage-locker-bathroom")
}
