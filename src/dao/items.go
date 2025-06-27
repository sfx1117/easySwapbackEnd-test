package dao

import (
	"EasySwapBackend-test/src/entity"
	"context"
	"fmt"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"time"
)

const OrderType = 1
const OrderStatus = 0

/*
*
查询集合中已上架NFT的数量
*/
func (dao *Dao) QueryListedAmount(ctx context.Context, chain, collectionAddr string) (int64, error) {
	// SQL解释:
	// 1. 从Item表(ci)和订单表(co)联表查询
	// 2. 关联条件:集合地址和tokenID都相同
	// 3. 使用distinct去重统计不同的tokenID数量
	// 4. WHERE条件:
	//   - 指定集合地址
	//   - 订单类型为listing(OrderType=1)
	//   - 订单状态为active(OrderStatus=0)
	//   - 卖家是NFT当前所有者
	//   - 排除marketplace_id=1的订单
	sql := fmt.Sprintf(`SELECT
	count( DISTINCT ( co.token_id ) ) AS counts 
FROM
	%s AS ci
	JOIN %s co ON ci.collection_address = co.collection_address 
	AND ci.token_id = co.token_id 
WHERE
	(
	co.collection_address = ?
	AND co.order_type = ?
	AND co.order_status =?
	AND ci.owner = co.maker 
	AND co.marketplace_id != ?
	)`, multi.ItemTableName(chain), multi.OrderTableName(chain))

	var counts int64
	err := dao.DB.WithContext(ctx).Raw(sql, collectionAddr, OrderType, OrderStatus, 1).Scan(&counts).Error
	if err != nil {
		return 0, errors.Wrap(err, "failed on get listed item amount")
	}
	return counts, nil
}

// QueryCollectionBids 查询NFT集合的出价信息
// 该函数主要用于获取某个NFT集合的所有有效出价信息,包括出价数量、价格、总价值和出价人数等
func (dao *Dao) QueryCollectionBids(ctx context.Context, chain, collectionAddr string, page, pageSize int) ([]entity.CollectionBids, int64, error) {
	// 1、统计总记录数
	// SQL解释:统计订单表中符合条件的记录数
	// 条件:1.指定集合地址 2.订单类型为出价单 3.订单状态为活跃 4.未过期
	// 按价格分组统计不同价格的出价数量
	var count int64
	err := dao.DB.WithContext(ctx).Table(multi.OrderTableName(chain)).
		Where("collection_address = ? and order_type = ? and order_status = ? and expire_time > ?",
			collectionAddr, multi.CollectionBidOrder, multi.OrderStatusActive, time.Now().Unix()).
		Group("price").
		Count(&count).Error
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed on count user items")
	}
	//2、分页查询出价详情
	var bids []entity.CollectionBids
	err = dao.DB.WithContext(ctx).Table(multi.OrderTableName(chain)).
		Select(`
			sum(quantity_remaining) AS size, 
			price,
			sum(quantity_remaining)*price as total,
			COUNT(DISTINCT maker) AS bidders`).
		Where("collection_address = ? and order_type = ? and order_status = ? and expire_time > ?",
			collectionAddr, multi.CollectionBidOrder, multi.OrderStatusActive, time.Now().Unix()).
		Group("price").
		Limit(pageSize).
		Offset(pageSize * (page - 1)).
		Scan(&bids).Error
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed on count user items")
	}
	return bids, count, nil
}

// QueryItemBids 查询Item的出价信息
func (dao *Dao) QueryItemBids(ctx context.Context, chain, collectionAddr, tokenId string, page, pageSize int) ([]entity.ItemBid, int64, error) {
	// 构建SQL查询
	// 查询字段包括:市场ID、集合地址、代币ID、订单ID、盐值、事件时间、过期时间、 价格、出价人、订单类型、未成交数量、出价总量
	// 查询条件1:集合级别的出价 - 匹配集合地址,订单类型为集合出价,状态为活跃,未过期且有剩余数量
	// 查询条件2:Item级别的出价 - 匹配集合地址和代币ID,订单类型为Item出价,其他条件同上
	db := dao.DB.WithContext(ctx).Table(multi.OrderTableName(chain)).
		Select("marketplace_id, collection_address, token_id, order_id, salt, "+
			"event_time, expire_time, price, maker as bidder, order_type, "+
			"quantity_remaining as bid_unfilled, size as bid_size").
		Where("collection_address = ? and order_type = ? and order_status = ? "+
			"and expire_time > ? and quantity_remaining > 0",
			collectionAddr, multi.CollectionBidOrder, multi.OrderStatusActive, time.Now().Unix()).
		Or("collection_address = ? and token_id=? and order_type = ? and order_status = ? "+
			"and expire_time > ? and quantity_remaining > 0",
			collectionAddr, tokenId, multi.ItemBidOrder, multi.OrderStatusActive, time.Now().Unix())

	//1、统计总记录数
	var count int64
	//创建一个新的会话(Session)
	countTxDB := db.Session(&gorm.Session{})
	err := countTxDB.Count(&count).Error
	if err != nil {
		return nil, 0, errors.Wrap(db.Error, "failed on count user items")
	}

	//2、如果没有记录直接返回
	var itemBids []entity.ItemBid
	if count == 0 {
		return itemBids, count, nil
	}
	//3、有记录，则进行分页查询
	err = db.Order("price desc").
		Limit(pageSize).
		Offset(pageSize * (page - 1)).
		Scan(&itemBids).Error
	if err != nil {
		return nil, 0, errors.Wrap(db.Error, "failed on count user items")
	}
	return itemBids, count, nil
}
