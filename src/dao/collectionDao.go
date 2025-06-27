package dao

import (
	"context"
	"fmt"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"time"
)

var collectionDetailFields = []string{"id", "chain_id", "token_standard", "name", "address", "image_uri", "floor_price", "sale_price", "item_amount", "owner_amount"}

// 查询指定链上的NFT集合信息
func (dao *Dao) QueryCollectionInfo(ctx context.Context, chain, address string) (*multi.Collection, error) {
	var collection multi.Collection
	err := dao.DB.WithContext(ctx).Table(multi.CollectionTableName(chain)).
		Select(collectionDetailFields).
		Where("address = ?", address).
		Find(&collection).
		Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on get collection info")
	}
	return &collection, nil
}

// 查询NFT集合的地板价
func (dao *Dao) QueryFloorPrice(ctx context.Context, chain, collectionAddr string) (decimal.Decimal, error) {
	// SQL解释:
	// 1. 从Item表(ci)和订单表(co)联表查询
	// 2. 选择字段:co.price作为地板价
	// 3. 关联条件:集合地址和tokenID都相同
	// 4. WHERE条件:
	//    - 指定集合地址
	//    - 订单类型为listing(OrderType=1)
	//    - 订单状态为active(OrderStatus=0)
	//    - 卖家是NFT当前所有者
	//    - 排除marketplace_id=1的订单
	// 5. 按价格升序排序,取第一条记录(即最低价)
	sql := fmt.Sprintf(`SELECT
	co.price AS price 
FROM
	%s AS ci
	JOIN %s co ON ci.collection_address = co.collection_address 
	AND ci.token_id = co.token_id 
WHERE
	(
	co.collection_address = ? 
	AND co.order_type = ? 
	AND co.order_status =? 
	AND ci.OWNER = co.maker 
	AND co.marketplace_id != ? 
	) 
ORDER BY
	co.price ASC 
	LIMIT 1`, multi.ItemTableName(chain), multi.OrderTableName(chain))

	var price decimal.Decimal
	err := dao.DB.WithContext(ctx).Raw(sql, collectionAddr, OrderType, OrderStatus, 1).Scan(&price).Error
	if err != nil {
		return decimal.Zero, errors.Wrap(err, "failed on get collection floor price")
	}
	return price, nil
}

// 查询指定NFT集合的最高卖单价格
func (dao *Dao) QueryCollectionSellPrice(ctx context.Context, chain, collectionAddr string) (*multi.Collection, error) {
	var collection multi.Collection
	sql := fmt.Sprintf(`SELECT
	co.collection_address AS address,
	co.price AS sale_price 
FROM
	%s AS co 
WHERE
	co.collection_address = ? 
	AND order_status = ? 
	AND co.order_type = ? 
	AND co.quantity_remaining > 0 
	AND co.expire_time > ? 
ORDER BY
	co.price DESC 
	LIMIT 1`, multi.OrderTableName(chain))

	err := dao.DB.WithContext(ctx).Raw(
		sql,
		collectionAddr,
		multi.OrderStatusActive,
		multi.CollectionBidOrder,
		time.Now().Unix()).
		Scan(&collection).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on get collection sell price")
	}
	return &collection, nil
}
