package entity

import (
	"github.com/shopspring/decimal"
)

type ItemBid struct {
	MarketplaceId     int             `json:"marketplace_id"`     //市场id
	CollectionAddress string          `json:"collection_address"` //集合地址
	TokenId           string          `json:"token_id"`           //代币id
	OrderId           string          `json:"order_id"`           //订单id
	EventTime         int64           `json:"event_time"`         //事件时间
	ExpireTime        int64           `json:"expire_time"`        //过期时间
	Price             decimal.Decimal `json:"price"`              //价格
	Salt              int64           `json:"salt"`               //盐
	BidSize           int64           `json:"bid_size"`           //出价总量
	BidUnfilled       int64           `json:"bid_unfilled"`       //未成交数量
	Bidder            string          `json:"bidder"`             //出价人
	OrderType         int64           `json:"order_type"`         //订单类型
}
