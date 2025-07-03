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

// 指定collection的item查询参数
type CollectionItemFilterParam struct {
	Sort        int    `json:"sort"`    //1- listing_price  2-listing_time 3-sale_price
	Status      []int  `json:"status"`  // 1 buy now  2 has offer  3 全选
	Markets     []int  `json:"markets"` // 0:ns 1:os 2:looksrare 3:x2y2
	TokenID     string `json:"token_id"`
	UserAddress string `json:"user_address"`
	ChainID     int    `json:"chain_id"`
	Page        int    `json:"page"`
	PageSize    int    `json:"page_size"`
}

// NFT返回参数
type NFTListInfoRes struct {
	Result interface{} `json:"result"`
	Count  int64       `json:"count"`
}

// NFTListInfo
type NFTListInfo struct {
	Name              string      `json:"name"`
	ImageURI          string      `json:"image_uri"`
	VideoType         string      `json:"video_type"`
	VideoURI          string      `json:"video_uri"`
	CollectionAddress string      `json:"collection_address"`
	TokenID           string      `json:"token_id"`
	OwnerAddress      string      `json:"owner_address"`
	Traits            []ItemTrait `json:"traits"`

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

	MarketID int `json:"market_id"`

	LastSellPrice    decimal.Decimal `json:"last_sell_price"`
	OwnerOwnedAmount int64           `json:"owner_owned_amount"`
}
type ItemTrait struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// 历史销售价格返回参数
type HistorySalesPriceRes struct {
	Result interface{} `json:"result"`
}

// 历史销售价格
type HistorySalesPriceInfo struct {
	Price     decimal.Decimal `json:"price"`
	TokenID   string          `json:"token_id"`
	TimeStamp int64           `json:"timeStamp"`
}
type CommonResp struct {
	Result interface{} `json:"result"`
}
type RefreshItem struct {
	ChainId           int64  `json:"chain_id"`
	CollectionAddress string `json:"collection_address"`
	TokenId           string `json:"token_id"`
}

// 集合排名信息
type CollectionRankingInfo struct {
	ImageUri    string          `json:"image_uri"`
	Name        string          `json:"name"`
	Address     string          `json:"address"`
	FloorPrice  string          `json:"floor_price"`
	FloorChange string          `json:"floor_price_change"`
	SellPrice   string          `json:"sell_price"`
	Volume      decimal.Decimal `json:"volume"`
	ItemNum     int64           `json:"item_num"`
	ItemOwner   int64           `json:"item_owner"`
	ItemSold    int64           `json:"item_sold"`
	ListAmount  int             `json:"list_amount"`
	ChainID     int             `json:"chain_id"`
}

// 集合上架数量
type CollectionListed struct {
	CollectionAddr string `json:"collection_address"`
	Count          int    `json:"count"`
}
