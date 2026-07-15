package handler

import (
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
)

// mtsBrowsePageSize is the client's fixed ITC page block: CITCWnd_List::
// ChangeCategorySub (v83 0x5BDD12) builds the page selector as
// (m_nCurrentCategoryItemCnt + 15) / 16, so every browse view pages in 16s
// and the GetItcListDone categoryItemCnt field must carry the TOTAL match
// count while the packet's item list carries one 16-item window.
const mtsBrowsePageSize = 16

// mtsPageWindow returns the requested 16-item window of items — the page the
// GetItcListDone packet carries. Out-of-range pages yield an empty window
// (the client clamps its selector to ceil(total/16), so this only happens on
// a stale request racing a shrinking result set).
func mtsPageWindow(items []fieldcb.MtsItem, page uint32) []fieldcb.MtsItem {
	start := int(page) * mtsBrowsePageSize
	if start < 0 || start >= len(items) {
		return []fieldcb.MtsItem{}
	}
	end := start + mtsBrowsePageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}
