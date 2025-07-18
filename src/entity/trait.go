package entity

import "github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"

type ItemTraitsResp struct {
	Result interface{} `json:"result"`
}

type TraitInfo struct {
	Trait        string  `json:"trait"`
	TraitValue   string  `json:"trait_value"`
	TraitAmount  int64   `json:"trait_amount"`
	TraitPercent float64 `json:"trait_percent"`
}

type TraitCount struct {
	multi.ItemTrait
	Count int64 `json:"count"`
}
