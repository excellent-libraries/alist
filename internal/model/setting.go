package model

const (
	SITE = iota
	STYLE
	PREVIEW
	GLOBAL
	SINGLE
	ARIA2
)

const (
	PUBLIC = iota
	PRIVATE
	READONLY
	DEPRECATED
)

type SettingItem struct {
	Key    string `json:"key" gorm:"primaryKey" binding:"required"` // unique key
	Value  string `json:"value"`                                    // value
	Help   string `json:"help"`                                     // help message
	Type   string `json:"type"`                                     // string, number, bool, select
	Values string `json:"values"`                                   // values for select
	Group  int    `json:"group"`                                    // use to group setting in frontend
	Flag   int    `json:"flag"`                                     // 0 = public, 1 = private, 2 = readonly, 3 = deprecated, etc.
}

func (s SettingItem) IsDeprecated() bool {
	return s.Flag == DEPRECATED
}
