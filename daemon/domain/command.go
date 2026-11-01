package domain

// Command -
type Command struct {
	VerifySource bool   `json:"verifySource,omitempty"`
	Estimated    bool   `json:"estimated,omitempty"`
	ID           string `json:"id"`
	Src          string `json:"src"`
	Dst          string `json:"dst"`
	Entry        string `json:"entry"`
	Size         uint64 `json:"size"`
	Transferred  uint64 `json:"transferred"`
	Status       int    `json:"status"`
	Reason       string `json:"reason"`
}
