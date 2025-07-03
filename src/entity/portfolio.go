package entity

import "github.com/shopspring/decimal"

type UserCollectionsParams struct {
	UserAddresses []string `json:"user_addresses"`
}
type UserCollectionsResp struct {
	Result interface{} `json:"result"`
}
type UserCollections struct {
	ChainID    int             `json:"chain_id"`
	Address    string          `json:"address"` // 合约地址
	Name       string          `json:"name"`
	Symbol     string          `json:"symbol"`
	ImageURI   string          `json:"image_uri"`
	ItemCount  int64           `json:"item_count"`
	FloorPrice decimal.Decimal `json:"floor_price"`
	ItemAmount int64           `json:"item_amount"`
}
type CollectionInfo struct {
	ChainID    int             `json:"chain_id"`
	Name       string          `json:"name"`
	Address    string          `json:"address"`
	Symbol     string          `json:"symbol"`
	ImageURI   string          `json:"image_uri"`
	ListAmount int             `json:"list_amount"`
	ItemAmount int64           `json:"item_amount"`
	FloorPrice decimal.Decimal `json:"floor_price"`
}
type ChainInfo struct {
	ChainID   int             `json:"chain_id"`
	ItemOwned int64           `json:"item_owned"`
	ItemValue decimal.Decimal `json:"item_value"`
}
type UserCollectionsData struct {
	CollectionInfos []CollectionInfo `json:"collection_info"`
	ChainInfos      []ChainInfo      `json:"chain_info"`
}
