package entity

import "github.com/shopspring/decimal"

// 集合详情返回参数
type CollectionDetailRes struct {
	Result interface{} `json:"result"`
}

// 集合详情
type CollectionDetail struct {
	ImageUri       string          `json:"image_uri"`
	Name           string          `json:"name"`
	Address        string          `json:"address"`
	ChainId        int             `json:"chain_id"`
	FloorPrice     decimal.Decimal `json:"floor_price"`
	SellPrice      string          `json:"sell_price"`
	VolumeTotal    decimal.Decimal `json:"volume_total"`
	Volume24h      decimal.Decimal `json:"volume_24h"`
	Sold24h        int64           `json:"sold_24h"`
	ListAmount     int64           `json:"list_amount"`
	TotalSupply    int64           `json:"total_supply"`
	OwnerAmount    int64           `json:"owner_amount"`
	RoyaltyFeeRate string          `json:"royalty_fee_rate"`
}

// CollectionBid查询参数
type CollectionBidFilterParam struct {
	ChainId  int `json:"chain_id"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// CollectionBid返回参数
type CollectionBidsRes struct {
	Result interface{} `json:"result"`
	Count  int64       `json:"count"`
}

// CollectionBid
type CollectionBids struct {
	Price   decimal.Decimal `json:"price"`
	Size    int             `json:"size"`
	Total   decimal.Decimal `json:"total"`
	Bidders int             `json:"bidders"`
}
