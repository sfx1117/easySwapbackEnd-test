package entity

import "github.com/shopspring/decimal"

type ItemPriceInfo struct {
	CollectionAddress string          `json:"collection_address"`
	TokenId           string          `json:"token_id"`
	Maker             string          `json:"maker"`
	Price             decimal.Decimal `json:"price"`
	OrderStatus       int             `json:"order_status"`
}

// item详情响应参数
type ItemDetailInfoResp struct {
	Result interface{} `json:"result"`
}

// item详情
type ItemDetailInfo struct {
	ChainID            int             `json:"chain_id"`
	Name               string          `json:"name"`
	CollectionAddress  string          `json:"collection_address"`
	CollectionName     string          `json:"collection_name"`
	CollectionImageURI string          `json:"collection_image_uri"`
	TokenID            string          `json:"token_id"`
	ImageURI           string          `json:"image_uri"`
	VideoType          string          `json:"video_type"`
	VideoURI           string          `json:"video_uri"`
	LastSellPrice      decimal.Decimal `json:"last_sell_price"`
	FloorPrice         decimal.Decimal `json:"floor_price"`
	OwnerAddress       string          `json:"owner_address"`
	MarketplaceID      int             `json:"marketplace_id"`

	ListOrderID    string          `json:"list_order_id"`
	ListTime       int64           `json:"list_time"`
	ListPrice      decimal.Decimal `json:"list_price"`
	ListExpireTime int64           `json:"list_expire_time"`
	ListSalt       int64           `json:"list_salt"`
	ListMaker      string          `json:"list_maker"`

	BidOrderID    string          `json:"bid_order_id"`
	BidTime       int64           `json:"bid_time"`
	BidExpireTime int64           `json:"bid_expire_time"`
	BidPrice      decimal.Decimal `json:"bid_price"`
	BidSalt       int64           `json:"bid_salt"`
	BidMaker      string          `json:"bid_maker"`
	BidType       int64           `json:"bid_type"`
	BidSize       int64           `json:"bid_size"`
	BidUnfilled   int64           `json:"bid_unfilled"`
}

// TopTrait过滤条件
type TopTraitFilterParams struct {
	TokenIds []string `json:"token_ids"`
	ChainID  int      `json:"chain_id"`
}

// TopTrait返回参数
type ItemTopTraitResp struct {
	Result interface{} `json:"result"`
}

// TopTrait实体类
type TraitPrice struct {
	CollectionAddress string          `json:"collection_address"`
	TokenID           string          `json:"token_id"`
	Trait             string          `json:"trait"`
	TraitValue        string          `json:"trait_value"`
	Price             decimal.Decimal `json:"price"`
}

// NFT图片信息
type ItemImage struct {
	CollectionAddress string `json:"collection_address"`
	TokenID           string `json:"token_id"`
	ImageUri          string `json:"image_uri"`
}

// NFT所有者信息
type ItemOwner struct {
	CollectionAddress string `json:"collection_address"`
	TokenID           string `json:"token_id"`
	Owner             string `json:"owner"`
}

type ItemInfo struct {
	CollectionAddress string `json:"collection_address"`
	TokenID           string `json:"token_id"`
}

type MultiChainItemInfo struct {
	ItemInfo  ItemInfo
	ChainName string
}
