package domain

// Item -
type Item struct {
	Estimated     bool   `json:"estimated,omitempty"`
	TransferSize  uint64 `json:"transferSize,omitempty"`
	RequiredSpace uint64 `json:"requiredSpace,omitempty"`
	Name          string `json:"name"`
	Size          uint64 `json:"size"`
	Path          string `json:"path"`
	Location      string `json:"location"`
	BlocksUsed    uint64 `json:"blocksUsed"`
}

// // BySize -
// type BySize []*Item

// func (s BySize) Len() int           { return len(s) }
// func (s BySize) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
// func (s BySize) Less(i, j int) bool { return s[i].Size > s[j].Size }
