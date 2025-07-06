package dao

import (
	"EasySwapBackend-test/src/entity"
	"context"
	"fmt"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
	"time"
)

const OrderType = 1
const OrderStatus = 0
const (
	BuyNow   = 1
	HasOffer = 2
	All      = 3
)
const (
	listTime      = 0
	listPriceAsc  = 1
	listPriceDesc = 2
	salePriceDesc = 3
	salePriceAsc  = 4
)

type CollectionItem struct {
	multi.Item
	MarketID       int    `json:"market_id"`
	Listing        bool   `json:"listing"`
	OrderID        string `json:"order_id"`
	OrderStatus    int    `json:"order_status"`
	ListMaker      string `json:"list_maker"`
	ListTime       int64  `json:"list_time"`
	ListExpireTime int64  `json:"list_expire_time"`
	ListSalt       int64  `json:"list_salt"`
}

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

// QueryCollectionItemOrder 查询集合内NFT Item的订单信息
func (dao *Dao) QueryCollectionItemOrder(ctx context.Context, chain, collectionAddr string, filter entity.CollectionItemFilterParam) ([]*CollectionItem, int64, error) {
	//1、如果未指定市场，则默认使用OrderBookDex
	if len(filter.Markets) == 0 {
		filter.Markets = []int{multi.OrderBookDex}
	}
	//2、初始化数据库查询
	db := dao.DB.WithContext(ctx).Table(fmt.Sprintf("%s as ci", multi.ItemTableName(chain)))
	coTableName := multi.OrderTableName(chain)

	//3、根据状态过滤查询 组装sql
	// status: 1-buy now(立即购买), 2-has offer(有报价), 3-all(所有)
	if len(filter.Status) == 1 { // filter.Status=[1] 或者 filter.Status=[2]
		// 构建基础SELECT语句  1. 关联订单表和Item表
		db.Select(
			"ci.id as id, ci.chain_id as chain_id, " +
				"ci.collection_address as collection_address,ci.token_id as token_id, " +
				"ci.name as name, ci.owner as owner, " +
				"min(co.price) as list_price, " +
				"SUBSTRING_INDEX(GROUP_CONCAT(co.marketplace_id ORDER BY co.price,co.marketplace_id),',', 1) AS market_id, " +
				"min(co.price) != 0 as listing").
			Joins(fmt.Sprintf("join %s co on co.collection_address=ci.collection_address "+
				"and co.token_id=ci.token_id", coTableName))

		//处理立即购买状态
		if filter.Status[0] == BuyNow {
			// 2. 条件:集合地址匹配、订单类型为listing、订单状态active、卖家是Item所有者
			db.Where("co.collection_address = ? and co.order_type = ? and co.order_status=? and co.maker = ci.owner",
				collectionAddr, multi.ListingOrder, multi.OrderStatusActive)

		} else if filter.Status[0] == HasOffer { //处理立即购买状态
			// 2. 条件:集合地址匹配、订单类型为offer、订单状态active
			db.Where("co.collection_address = ? and co.order_type = ? and co.order_status=?",
				collectionAddr, multi.OfferOrder, multi.OrderStatusActive)
		}

		//根据市场id过滤
		if len(filter.Markets) == 1 {
			db.Where("co.marketplace_id = ?", filter.Markets[0])
		} else if len(filter.Markets) != 5 {
			db.Where("co.marketplace_id in (?)", filter.Markets)
		}
		//根据tokenId过滤
		if filter.TokenID != "" {
			db.Where("co.token_id = ?", filter.TokenID)
		}
		//根据用户地址过滤
		if filter.UserAddress != "" {
			db.Where("ci.owner = ?", filter.UserAddress)
		}
		//分组条件
		db.Group("co.token_id")

	} else if len(filter.Status) == 2 { // 处理同时有买卖订单的情况 即 filter.Status=[1,2]
		// SQL解释:
		// 1. 关联订单表和Item表
		// 2. 条件:订单状态active、卖家是Item所有者
		// 3. 分组后需同时存在listing和offer订单
		db.Select(
			"ci.id as id, ci.chain_id as chain_id,"+
				"ci.collection_address as collection_address,ci.token_id as token_id, "+
				"ci.name as name, ci.owner as owner, "+
				"min(co.price) as list_price, "+
				"SUBSTRING_INDEX(GROUP_CONCAT(co.marketplace_id ORDER BY co.price,co.marketplace_id),',', 1) AS market_id").
			Joins(fmt.Sprintf(
				"join %s co on co.collection_address=ci.collection_address and co.token_id=ci.token_id",
				coTableName)).
			Where(
				"co.collection_address = ? and co.order_status=? and co.maker = ci.owner",
				collectionAddr, multi.OrderStatusActive)
		//根据市场id过滤
		if len(filter.Markets) == 1 {
			db.Where("co.marketplace_id = ?", filter.Markets[0])
		} else if len(filter.Markets) != 5 {
			db.Where("co.marketplace_id in (?)", filter.Markets)
		}
		//根据tokenId过滤
		if filter.TokenID != "" {
			db.Where("co.token_id = ?", filter.TokenID)
		}
		//根据用户地址过滤
		if filter.UserAddress != "" {
			db.Where("ci.owner = ?", filter.UserAddress)
		}
		//分组条件
		db.Group("co.token_id").
			Having("min(co.order_type) = ? and max(co.order_type) = ?", multi.ListingOrder, multi.OfferOrder)
	} else { // 处理所有状态 即 filter.Status=[3]
		// 1. 子查询获取每个token的最低listing价格
		// 2. 左连接子查询结果到Item表
		// 3. 根据条件过滤
		subQuery := dao.DB.WithContext(ctx).Table(
			fmt.Sprintf("%s as cis", multi.ItemTableName(chain))).
			Select(
				"cis.id as item_id,cis.collection_address as collection_address,"+
					"cis.token_id as token_id, cis.owner as owner, cos.order_id as order_id, "+
					"min(cos.price) as list_price, "+
					"SUBSTRING_INDEX(GROUP_CONCAT(cos.marketplace_id ORDER BY cos.price,cos.marketplace_id),',', 1) AS market_id, "+
					"min(cos.price) != 0 as listing").
			Joins(fmt.Sprintf(
				"join %s cos on cos.collection_address=cis.collection_address and cos.token_id=cis.token_id",
				coTableName)).
			Where(
				"cos.collection_address = ? and cos.order_type = ? and cos.order_status=? "+
					"and cos.maker = cis.owner",
				collectionAddr, multi.ListingOrder, multi.OrderStatusActive)

		if len(filter.Markets) == 1 {
			subQuery.Where("cos.marketplace_id = ?", filter.Markets[0])
		} else if len(filter.Markets) != 5 {
			subQuery.Where("cos.marketplace_id in (?)", filter.Markets)
		}
		subQuery.Group("cos.token_id")

		db.Select("ci.id as id, ci.chain_id as chain_id,"+
			"ci.collection_address as collection_address, ci.token_id as token_id, "+
			"ci.name as name, ci.owner as owner, "+
			"co.list_price as list_price, co.market_id as market_id, co.listing as listing").
			Joins("left join (?) co on co.collection_address=ci.collection_address and co.token_id=ci.token_id", subQuery).
			Where("ci.collection_address = ?", collectionAddr)
		//根据tokenId过滤
		if filter.TokenID != "" {
			db.Where("co.token_id = ?", filter.TokenID)
		}
		//根据用户地址过滤
		if filter.UserAddress != "" {
			db.Where("ci.owner = ?", filter.UserAddress)
		}
	}
	//4、统计总记录数
	var count int64
	countSessionTx := db.Session(&gorm.Session{})
	err := countSessionTx.Count(&count).Error
	if err != nil {
		return nil, 0, errors.Wrap(db.Error, "failed on count items")
	}

	//5、处理排序
	if len(filter.Status) == 0 {
		db.Order("listing desc")
	}
	if filter.Sort == 0 {
		filter.Sort = listPriceAsc
	}
	//5.1、根据不同排序条件设置ORDER BY
	switch filter.Sort {
	case listTime:
		db.Order("list_time desc,ci.id asc")
	case listPriceAsc:
		db.Order("list_price asc, ci.id asc")
	case listPriceDesc:
		db.Order("list_price desc,ci.id asc")
	case salePriceDesc:
		db.Order("sale_price desc,ci.id asc")
	case salePriceAsc:
		db.Order("sale_price = 0,sale_price asc,ci.id asc")
	}

	//6、分页查询
	var items []*CollectionItem
	err = db.Limit(filter.PageSize).
		Offset(filter.PageSize * (filter.Page - 1)).
		Scan(&items).Error
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed on get query items info")
	}
	return items, count, nil
}

// QueryListingInfo 查询订单上架信息
// 该函数主要功能:
// 1、根据传入的价格信息列表查询对应的订单信息
// 2、每个价格信息包含：集合地址，代币id，创建者，价格，订单状态
// 3、返回的订单信息包含：集合地址，代币id，订单id，创建时间，过期时间，盐
func (dao *Dao) QueryListingInfo(ctx context.Context, chain string, itemPrice []entity.ItemPriceInfo) ([]multi.Order, error) {
	//1、构建查询条件
	var conditions []clause.Expr
	for _, item := range itemPrice {
		conditions = append(conditions,
			gorm.Expr("(?, ?, ?, ?, ?)",
				item.CollectionAddress,
				item.TokenId,
				item.Maker,
				item.OrderStatus,
				item.Price,
			))
	}
	//2、sql查询
	var ordersInfo []multi.Order
	err := dao.DB.WithContext(ctx).Table(multi.OrderTableName(chain)).
		Select("collection_address,token_id,order_id,event_time,expire_time,salt,maker").
		Where("(collection_address,token_id,maker,order_status,price) in (?)", conditions).
		Scan(&ordersInfo).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on query items order id")
	}
	return ordersInfo, nil
}

type UserItemCount struct {
	Owner  string `json:"owner"`
	Counts int64  `json:"counts"`
}

// QueryUserItemCount 查询用户持有NFT数量统计
// 该函数主要功能:
// 1. 根据链名称、集合地址和用户地址列表查询每个用户持有的NFT数量
// 2. 返回用户地址和对应的NFT持有数量
func (dao *Dao) QueryUserItemCount(ctx context.Context, chain, collectionAddr string, owners []string) ([]UserItemCount, error) {
	var userItemCount []UserItemCount
	err := dao.DB.WithContext(ctx).
		Table(multi.ItemTableName(chain)).
		Select("owner,count(*) as counts").
		Where("collection_address = ? and owner in (?)", collectionAddr, owners).
		Group("owner").
		Scan(&userItemCount).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on get user item count")
	}
	return userItemCount, nil
}

// QueryLastSalePrice 查询NFT最近的销售价格
// 该函数主要功能:
// 1. 根据链名称、集合地址和代币ID列表查询每个NFT最近一次的销售价格
// 2. 返回NFT的集合地址、代币ID和对应的销售价格
func (dao *Dao) QueryLastSalePrice(ctx context.Context, chain, collectionAddr string, owners []string) ([]multi.Activity, error) {
	var lastSales []multi.Activity
	sql := fmt.Sprintf(`
		SELECT a.collection_address, a.token_id, a.price
		FROM %s a
		INNER JOIN (
			SELECT collection_address,token_id, 
				MAX(event_time) as max_event_time
			FROM %s
			WHERE collection_address = ?
				AND token_id IN (?)
				AND activity_type = ?
			GROUP BY collection_address,token_id
		) groupedA 
		ON a.collection_address = groupedA.collection_address
		AND a.token_id = groupedA.token_id
		AND a.event_time = groupedA.max_event_time
		AND a.activity_type = ?`,
		multi.ActivityTableName(chain),
		multi.ActivityTableName(chain))
	err := dao.DB.Raw(sql, collectionAddr, owners, multi.Sale, multi.Sale).
		Scan(&lastSales).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on get item last sale price")
	}
	return lastSales, nil
}

// QueryBestBids 查询NFT的最佳出价信息
// 该函数主要功能:
// 1. 根据链名称、用户地址、集合地址和代币ID列表查询NFT的出价信息
// 2. 返回符合条件的出价订单列表
// 3. 如果指定了用户地址,则排除该用户的出价
func (dao *Dao) QueryBestBids(ctx context.Context, chain, collectionAddr, userAddr string, tokenIds []string) ([]multi.Order, error) {
	var bestBids []multi.Order
	var sql string
	// SQL解释:
	// 1. 查询订单表中符合条件的出价记录
	// 2. 条件包括:
	//    - 指定集合地址
	//    - 指定代币ID列表
	//    - 订单类型为出价单
	//    - 订单状态为激活
	//    - 未过期
	//    - 剩余数量大于0
	//    - 如果指定用户地址,则排除该用户的出价
	if userAddr == "" {
		sql = fmt.Sprintf(`
			SELECT order_id, token_id, event_time, price, salt, 
				expire_time, maker, order_type, quantity_remaining, size   
			FROM %s
			WHERE collection_address = ?
				AND token_id IN (?)
				AND order_type = ?
				AND order_status = ?
				AND expire_time > ?
				AND quantity_remaining > 0
		`, multi.OrderTableName(chain))
	} else {
		sql = fmt.Sprintf(`
			SELECT order_id, token_id, event_time, price, salt, 
				expire_time, maker, order_type, quantity_remaining, size   
			FROM %s
			WHERE collection_address = ?
				AND token_id IN (?)
				AND order_type = ?
				AND order_status = ?
				AND expire_time > ?
				AND quantity_remaining > 0
				AND maker != '%s'
		`, multi.OrderTableName(chain), userAddr)
	}
	err := dao.DB.Raw(sql, collectionAddr, tokenIds, multi.ItemBidOrder, multi.OrderStatusActive, time.Now().Unix()).
		Scan(&bestBids).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on get item best bids")
	}
	return bestBids, nil
}

// QueryCollectionBestBid 查询集合最高出价信息
// 该函数主要功能:
// 1. 根据链名称、用户地址和集合地址查询该集合的最高出价订单
// 2. 如果指定了用户地址,则排除该用户的出价
// 3. 返回价格最高的一个有效订单(未过期且有剩余数量)
func (dao *Dao) QueryCollectionBestBid(ctx context.Context, chain, collectionAddr, userAddr string) (multi.Order, error) {
	var bestBid multi.Order
	var sql string

	if userAddr == "" {
		sql = fmt.Sprintf(`SELECT order_id, price, event_time, expire_time, salt, maker, 
				order_type, quantity_remaining, size  
			FROM %s
			WHERE collection_address = ?
			AND order_type = ?
			AND order_status = ?
			AND quantity_remaining > 0
			AND expire_time > ? 
			ORDER BY price DESC 
			LIMIT 1`,
			multi.OrderTableName(chain))
	} else {
		sql = fmt.Sprintf(`SELECT order_id, price, event_time, expire_time, salt, maker, 
				order_type, quantity_remaining, size  
			FROM %s
			WHERE collection_address = ?
			AND order_type = ?
			AND order_status = ?
			AND quantity_remaining > 0
			AND expire_time > ? 
			AND maker != '%s'
			ORDER BY price DESC 
			LIMIT 1`,
			multi.OrderTableName(chain),
			userAddr)
	}
	err := dao.DB.Raw(sql, collectionAddr, multi.ItemBidOrder, multi.OrderStatusActive, time.Now().Unix()).
		Scan(&bestBid).Error
	if err != nil {
		return bestBid, errors.Wrap(err, "failed on get item best bids")
	}
	return bestBid, nil
}

// QueryItemInfo 查询单个NFT Item的详细信息
func (dao *Dao) QueryItemInfo(ctx context.Context, chain, collectionAddr, tokenId string) (*multi.Item, error) {
	var item multi.Item
	// 构建SQL查询
	// 从items表中查询指定NFT的信息
	err := dao.DB.WithContext(ctx).
		Table(fmt.Sprintf("%s as ci", multi.ItemTableName(chain))).
		Select("ci.id as id, "+
			"ci.chain_id as chain_id, "+
			"ci.collection_address as collection_address, "+
			"ci.token_id as token_id, "+
			"ci.name as name, "+
			"ci.owner as owner").
		Where("ci.collection_address =? and ci.token_id = ? ", collectionAddr, tokenId).
		Scan(&item).Error
	if err != nil {
		return nil, errors.Wrap(err, "")
	}
	return &item, nil
}

// QueryItemListInfo 查询单个NFT的挂单信息
// 主要功能:
// 1. 查询NFT基本信息(ID、稀有度等)和挂单信息(价格、市场等)
// 2. 如果有挂单,则查询挂单的详细信息(订单ID、过期时间等)
func (dao *Dao) QueryItemListInfo(ctx context.Context, chain, collectionAddr, tokenId string) (*CollectionItem, error) {
	var collectionItem CollectionItem
	db := dao.DB.WithContext(ctx).Table(fmt.Sprintf("%s as ci", multi.ItemTableName(chain)))
	coTableName := multi.OrderTableName(chain)
	// SQL解释:
	// 1. 从items表和orders表联表查询
	// 2. 选择NFT基本信息和挂单信息
	// 3. 按价格升序,取最低价的市场ID
	// 4. 过滤条件:匹配NFT、活跃订单、owner是卖家
	err := db.Select("ci.id as id, ci.chain_id as chain_id, "+
		"ci.collection_address as collection_address,ci.token_id as token_id, "+
		"ci.name as name, ci.owner as owner, "+
		"min(co.price) as list_price, "+
		"SUBSTRING_INDEX(GROUP_CONCAT(co.marketplace_id ORDER BY co.price,co.marketplace_id),',', 1) AS market_id, "+
		"min(co.price) != 0 as listing").
		Joins(fmt.Sprintf("join %s co on co.collection_address=ci.collection_address "+
			"and co.token_id=ci.token_id", coTableName)).
		Where("ci.collection_address =? and ci.token_id = ? and co.order_type = ? and co.order_status=? "+
			"and co.maker = ci.owner", collectionAddr, tokenId, multi.ListingOrder, multi.OrderStatusActive).
		Group("ci.collection_address,ci.token_id").
		Scan(&collectionItem).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on query user items list info")
	}
	//如果没有挂单，则直接返回
	if !collectionItem.Listing {
		return &collectionItem, nil
	}
	//如果有挂单，则查询挂单详情
	var listOrder multi.Order
	// SQL解释:
	// 如果有挂单,查询订单详细信息
	// 1. 从orders表查询订单ID、过期时间等信息
	// 2. 匹配NFT、卖家、状态和价格
	err = dao.DB.WithContext(ctx).Table(multi.OrderTableName(chain)).
		Select("order_id, expire_time, maker, salt, event_time").
		Where("collection_address=? and token_id=? and maker=? and order_status=? and price = ?",
			collectionItem.CollectionAddress, collectionItem.TokenId, collectionItem.Owner,
			multi.OrderStatusActive, collectionItem.ListPrice).
		Scan(&listOrder).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on query item order id")
	}
	collectionItem.OrderID = listOrder.OrderID
	collectionItem.ListSalt = listOrder.Salt
	collectionItem.ListExpireTime = listOrder.ExpireTime
	collectionItem.ListMaker = listOrder.Maker
	collectionItem.ListTime = listOrder.EventTime
	return &collectionItem, nil
}

// QueryTraitsPrice 查询NFT Trait的价格信息
// 主要功能:
// 1. 查询指定NFT集合中特定token id的 Trait价格
// 2. 通过关联订单表和 Trait表,找出每个 Trait对应的最低挂单价格
// 3. 返回 Trait价格列表
func (dao *Dao) QueryTraitPrice(ctx context.Context, chain, collectionAddr string, tokenIds []string) ([]entity.TraitPrice, error) {
	var traitPrice []entity.TraitPrice
	//构建子查询,查询指定item的trait信息
	listSubQuery := dao.DB.WithContext(ctx).
		Table(fmt.Sprintf("%s as gf_order", multi.OrderTableName(chain))).
		// 查询字段: Trait名称、 Trait值、最低价格
		Select("gf_attribute.trait,gf_attribute.trait_value,min(gf_order.price) as price").
		// 条件1:匹配集合地址、订单类型为挂单、订单状态为活跃
		Where("gf_order.collection_address=? and gf_order.order_type=? and gf_order.order_status = ?",
			collectionAddr, multi.ListingOrder, multi.OrderStatusActive).
		// 条件2: Trait必须在指定token的 Trait列表中
		Where("(gf_attribute.trait,gf_attribute.trait_value) in (?)",
			dao.DB.WithContext(ctx).
				Table(fmt.Sprintf("%s as gf_attr", multi.ItemTraitTableName(chain))).
				Select("gf_attr.trait, gf_attr.trait_value").
				Where("gf_attr.collection_address=? and gf_attr.token_id in (?)", collectionAddr, tokenIds))
	// 关联 Trait表,按 Trait分组查询
	err := listSubQuery.Joins(fmt.Sprintf("join %s as gf_attribute on gf_order.collection_address = gf_attribute.collection_address "+
		"and gf_order.token_id=gf_attribute.token_id", multi.ItemTraitTableName(chain))).
		Group("gf_attribute.trait, gf_attribute.trait_value").
		Scan(&traitPrice).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on query trait price")
	}
	return traitPrice, nil
}

// 更新NFT所有者
func (dao *Dao) UpdateItemOwner(ctx context.Context, chain, collectionAddr, tokenId, owner string) error {
	err := dao.DB.WithContext(ctx).
		Table(multi.ItemTableName(chain)).
		Update("owner", owner).
		Where("collection_address = ? and token_id = ?", collectionAddr, tokenId).
		Error
	if err != nil {
		return errors.Wrap(err, "failed on get user item count")
	}
	return nil
}

// 查询多个集合中已上架NFT的数量
func (dao *Dao) QueryListedAmountEachCollection(ctx context.Context, chain string, collectionAddrs, userAddrs []string) ([]entity.CollectionInfo, error) {
	var counts []entity.CollectionInfo
	// SQL解释:
	// 1. 从Item表(ci)和订单表(co)联表查询
	// 2. 选择字段:
	//    - ci.collection_address 作为 address
	//    - count(distinct co.token_id) 作为 list_amount,统计每个集合中不重复的tokenID数量
	// 3. 关联条件:集合地址和tokenID都相同
	// 4. WHERE条件:
	//    - 集合地址在给定列表中
	//    - NFT所有者在给定用户列表中
	//    - 订单类型为listing(OrderType=1)
	//    - 订单状态为active(OrderStatus=0)
	//    - 卖家是NFT当前所有者
	//    - 排除marketplace_id=1的订单
	// 5. 按集合地址分组,获取每个集合的统计结果
	sql := fmt.Sprintf(`SELECT  ci.collection_address as address, count(distinct (co.token_id)) as list_amount
			FROM %s as ci
					join %s co on co.collection_address = ci.collection_address and co.token_id = ci.token_id
			WHERE (co.collection_address in (?) and ci.owner in (?) and co.order_type = ? and
				co.order_status = ? and co.maker = ci.owner and co.marketplace_id != ?) group by ci.collection_address`,
		multi.ItemTableName(chain), multi.CollectionTableName(chain))
	err := dao.DB.WithContext(ctx).
		Raw(sql, collectionAddrs, userAddrs, OrderType, OrderStatus, 1).
		Scan(&counts).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on get listed item amount")
	}
	return counts, nil
}

/*
*
查询多个集合的最高出价信息
// 该函数主要功能:
// 1. 根据链名称、用户地址和集合地址列表查询每个集合的最高出价订单
// 2. 如果指定了用户地址,则排除该用户的出价
// 3. 返回每个集合中价格最高的有效订单(未过期且有剩余数量)
*/
func (dao *Dao) QueryCollectionsBestBid(ctx context.Context, chain string, userAddr string, collectionAddrs []string) ([]*multi.Order, error) {
	var bestBids []*multi.Order
	// 1. 主查询:从订单表中查询订单详细信息
	sql := fmt.Sprintf("SELECT collection_address, order_id, price,event_time, expire_time, salt, maker, order_type, quantity_remaining, size  FROM %s ", multi.OrderTableName(chain))
	// 2. 子查询:获取每个集合的最高出价
	sql += fmt.Sprintf("where (collection_address,price) in (SELECT collection_address, max(price) as price FROM %s ", multi.OrderTableName(chain))
	// 3. 子查询条件:
	//   - 集合地址在给定列表中
	//   - 订单类型为集合出价单
	//   - 订单状态为活跃
	//   - 剩余数量大于0
	//   - 未过期
	//   - 如果指定用户地址,则排除该用户
	sql += "where collection_address in (?) and order_type = ? and order_status = ? and quantity_remaining > 0 and expire_time > ? "
	if userAddr != "" {
		sql += fmt.Sprintf("and maker != '%s' ", userAddr)
	}
	sql += "group by collection_address ) "
	// 4. 主查询条件:与子查询条件相同
	sql += "and order_type = ? and order_status = ? and quantity_remaining > 0 and expire_time > ? "
	if userAddr != "" {
		sql += fmt.Sprintf("and maker != '%s' ", userAddr)
	}
	//5、执行查询sql
	err := dao.DB.WithContext(ctx).
		Raw(sql, collectionAddrs, multi.CollectionBidOrder, multi.OrderStatusActive, time.Now().Unix(),
			multi.CollectionBidOrder, multi.OrderStatusActive, time.Now().Unix()).
		Scan(&bestBids).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on get item best bids")
	}
	return bestBids, nil
}

/*
*
查询多个NFT Item的最高出价信息
// 主要功能:
// 1. 根据链名称、用户地址和Itemem信息列表查询ItemItem的最高出价订单
// 2. 如果指定了用户地址,则排除该用户的出价
// 3. 返回所有符合条件的有效订单(未过期且有剩余数量)
*/
func (dao *Dao) QueryItemsBestBids(ctx context.Context, chain string, userAddr string, itemInfos []entity.ItemInfo) ([]multi.Order, error) {
	var bestBids []multi.Order
	sql := ""
	//构建查询条件，将每个item的集合地址和tokenid拼装成(addr,tokenId)形式
	var conditions []clause.Expr
	for _, item := range itemInfos {
		conditions = append(conditions, gorm.Expr("(?,?)", item.CollectionAddress, item.TokenID))
	}
	//根据是否指定用户地址构建不同的SQL
	if userAddr == "" {
		sql += fmt.Sprintf(`
SELECT order_id, token_id, event_time, price, salt, expire_time, maker, order_type, quantity_remaining, size
    FROM %s
    WHERE (collection_address,token_id) IN (?)
      AND order_type = ?
      AND order_status = ?
	  AND quantity_remaining > 0
      AND expire_time > ?
`, multi.OrderTableName(chain))
	} else {
		sql += fmt.Sprintf(`
SELECT order_id, token_id, event_time, price, salt, expire_time, maker, order_type, quantity_remaining,size 
    FROM %s
    WHERE (collection_address,token_id) IN (?)
      AND order_type = ?
      AND order_status = ?
	  AND quantity_remaining > 0
      AND expire_time > ?
	  AND maker != '%s'
`, multi.OrderTableName(chain))
	}
	//执行sql查询
	err := dao.DB.WithContext(ctx).
		Raw(sql, conditions, multi.ItemBidOrder, multi.OrderStatusActive, time.Now().Unix()).
		Scan(&bestBids).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on get item best bids")
	}
	return bestBids, nil
}

/*
*
查询多条链上用户NFT Item的挂单信息
// 主要功能:
// 1. 根据用户地址列表和Item信息列表查询每个Item的挂单状态
// 2. 支持跨链查询,按链名称分组处理
// 3. 返回每个Item的挂单价格、市场ID等信息
*/
func (dao *Dao) QueryMultiChainUserItemsListInfo(ctx context.Context, userAddrs []string, itemInfos []entity.MultiChainItemInfo) ([]*CollectionItem, error) {
	var collectionItems []*CollectionItem
	// 1、构建用户地址参数字符串: 'addr1','addr2',...
	var userAddrParam string
	for i, addr := range userAddrs {
		userAddrParam += fmt.Sprintf("'%s'", addr)
		if i < len(userAddrs)-1 {
			userAddrParam += ","
		}
	}
	//2、按链名称对Item信息分组
	chainItemMap := make(map[string][]entity.ItemInfo)
	for _, item := range itemInfos {
		items, ok := chainItemMap[strings.ToLower(item.ChainName)]
		if ok {
			items = append(items, item.ItemInfo)
			chainItemMap[strings.ToLower(item.ChainName)] = items
		} else {
			chainItemMap[strings.ToLower(item.ChainName)] = []entity.ItemInfo{item.ItemInfo}
		}
	}
	//3、构建查询sql
	//3.1、构建sql头
	sqlHead := "select * from ("
	//3.2、构建sql中部
	sqlMid := ""
	for chain, items := range chainItemMap {
		if sqlMid != "" {
			sqlMid += " union all "
		}
		// 构建子查询SQL
		sqlMid += "(select ci.id as id, ci.chain_id as chain_id,ci.collection_address as collection_address," +
			"ci.token_id as token_id, ci.name as name, ci.owner as owner,min(co.price) as list_price, " +
			"SUBSTRING_INDEX(GROUP_CONCAT(co.marketplace_id ORDER BY co.price,co.marketplace_id),',', 1) AS market_id, " +
			"min(co.price) != 0 as listing "
		// 关联Item表和订单表
		sqlMid += fmt.Sprintf("from %s as ci ", multi.ItemTableName(chain))
		sqlMid += fmt.Sprintf("join %s as co ", multi.OrderTableName(chain))
		sqlMid += "on co.collection_address=ci.collection_address and co.token_id=ci.token_id "
		// 查询条件:匹配集合地址和tokenID、订单类型为listing、状态为active、卖家是Item所有者
		sqlMid += "where (co.collection_address,co.token_id) in "
		sqlMid += fmt.Sprintf("(('%s','%s')", items[0].CollectionAddress, items[0].TokenID)
		for i := 1; i < len(items); i++ {
			sqlMid += fmt.Sprintf(",('%s','%s')", items[i].CollectionAddress, items[i].TokenID)
		}
		sqlMid += ") "
		sqlMid += fmt.Sprintf("and co.order_type = %d and co.order_status=%d and co.maker = ci.owner and co.maker in (%s) ",
			multi.ListingOrder, multi.OrderStatusActive, userAddrParam)
		//分组条件
		sqlMid += "group by co.collection_address,co.token_id)"
	}
	//3.3、构建sql尾部
	sqlTail := ") as combined"
	//3.4、组合sql
	sql := sqlHead + sqlMid + sqlTail

	//4、执行查询sql
	err := dao.DB.WithContext(ctx).Raw(sql).Scan(&collectionItems).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on query user multi chain items list info")
	}
	return collectionItems, nil
}

/*
*
查询多条链上的NFT挂单信息
*/
func (dao *Dao) QueryMultiChainListingInfo(ctx context.Context, priceInfos []entity.MultiChainItemPriceInfo) ([]multi.Order, error) {
	var orders []multi.Order

	// 1、按链名称对价格信息分组
	chainPriceMap := make(map[string][]entity.MultiChainItemPriceInfo)
	for _, price := range priceInfos {
		items, ok := chainPriceMap[strings.ToLower(price.ChainName)]
		if ok {
			items = append(items, price)
			chainPriceMap[strings.ToLower(price.ChainName)] = items
		} else {
			chainPriceMap[strings.ToLower(price.ChainName)] = []entity.MultiChainItemPriceInfo{price}
		}
	}
	//2、构建查询sql
	//2.1 构建SQL头
	sqlHead := "select * from ("
	//2.2 构建sql中部
	sqlMid := ""
	for chain, prices := range chainPriceMap {
		if sqlMid == "" {
			sqlMid += " union all "
		}

		sqlMid += "(select collection_address,token_id,order_id,salt,event_time,expire_time,maker "
		sqlMid += fmt.Sprintf("from %s ", multi.OrderTableName(chain))
		sqlMid += "where (collection_address,token_id,maker,order_status,price) in "
		// 构建IN查询条件: (('addr1','id1','maker1',status1,price1),...)
		sqlMid += fmt.Sprintf("(('%s','%s','%s',%d, %s)", prices[0].CollectionAddress, prices[0].TokenId, prices[0].Maker, prices[0].OrderStatus, prices[0].Price.String())
		for i := 1; i < len(prices); i++ {
			sqlMid += fmt.Sprintf(",('%s','%s','%s',%d, %s)", prices[i].CollectionAddress, prices[i].TokenId, prices[i].Maker, prices[i].OrderStatus, prices[i].Price.String())
		}
		sqlMid += "))"
	}
	//2.3 构建sql尾部
	sqlTail := ") as combined"
	//2.4 组合sql
	sql := sqlHead + sqlMid + sqlTail

	//3、执行sql
	err := dao.DB.WithContext(ctx).Raw(sql).Scan(&orders).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on get multi chain listing info")
	}
	return orders, nil
}

/*
*
//  查询多条链上用户Item的过期挂单信息
// 主要功能:
// 1. 根据用户地址列表和Item信息列表查询每个Item的挂单状态
// 2. 支持查询多条链上的Item信息
// 3. 返回Item的基本信息和挂单信息(价格、市场等)
*/
func (d *Dao) QueryMultiChainUserItemsExpireListInfo(ctx context.Context, userAddrs []string,
	itemInfos []entity.MultiChainItemInfo) ([]*CollectionItem, error) {
	var collectionItems []*CollectionItem

	// 构建用户地址参数字符串: 'addr1','addr2',...
	var userAddrsParam string
	for i, addr := range userAddrs {
		userAddrsParam += fmt.Sprintf(`'%s'`, addr)
		if i < len(userAddrs)-1 {
			userAddrsParam += ","
		}
	}

	// SQL语句组成部分
	sqlHead := "SELECT * FROM (" // 外层查询开始
	sqlTail := ") as combined"   // 外层查询结束
	var sqlMids []string         // 存储每个Item的子查询

	// 构建IN查询条件: (('addr1','id1'),('addr2','id2'),...)
	tmpStat := fmt.Sprintf("(('%s','%s')", itemInfos[0].ItemInfo.CollectionAddress, itemInfos[0].ItemInfo.TokenID)
	for i := 1; i < len(itemInfos); i++ {
		tmpStat += fmt.Sprintf(",('%s','%s')", itemInfos[i].ItemInfo.CollectionAddress, itemInfos[i].ItemInfo.TokenID)
	}
	tmpStat += ") "

	// 遍历每个Item构建子查询
	for _, info := range itemInfos {
		sqlMid := "("
		// 选择字段:Item基本信息、最低挂单价格、市场ID等
		sqlMid += "select ci.id as id, ci.chain_id as chain_id,"
		sqlMid += "ci.collection_address as collection_address,ci.token_id as token_id, " +
			"ci.name as name, ci.owner as owner,"
		sqlMid += "min(co.price) as list_price, " +
			"SUBSTRING_INDEX(GROUP_CONCAT(co.marketplace_id ORDER BY co.price,co.marketplace_id),',', 1) " +
			"AS market_id, min(co.price) != 0 as listing "

		// 关联Item表和订单表
		sqlMid += fmt.Sprintf("from %s as ci ", multi.ItemTableName(info.ChainName))
		sqlMid += fmt.Sprintf("join %s co ", multi.OrderTableName(info.ChainName))
		sqlMid += "on co.collection_address=ci.collection_address and co.token_id=ci.token_id "

		// 查询条件:
		// 1. 匹配集合地址和tokenID
		// 2. 订单类型为listing
		// 3. 订单状态为active或expired
		// 4. 卖家是Item所有者且在用户列表中
		sqlMid += "where (co.collection_address,co.token_id) in "
		sqlMid += tmpStat
		sqlMid += fmt.Sprintf("and co.order_type = %d and (co.order_status=%d or co.order_status=%d) "+
			"and co.maker = ci.owner and co.maker in (%s) ",
			multi.ListingOrder, multi.OrderStatusActive, multi.OrderStatusExpired, userAddrsParam)
		sqlMid += "group by co.collection_address,co.token_id"
		sqlMid += ")"

		sqlMids = append(sqlMids, sqlMid)
	}

	// 使用UNION ALL组合所有子查询
	sql := sqlHead
	for i := 0; i < len(sqlMids); i++ {
		if i != 0 {
			sql += " UNION ALL " // 使用UNION ALL合并结果集
		}
		sql += sqlMids[i]
	}
	sql += sqlTail

	// 执行SQL查询
	if err := d.DB.WithContext(ctx).Raw(sql).Scan(&collectionItems).Error; err != nil {
		return nil, errors.Wrap(err, "failed on query user multi chain items list info")
	}

	return collectionItems, nil
}

/*
*
查询集合中前N个最高出价订单
// 1. 查询指定集合中的最高出价订单
// 2. 根据剩余数量展开订单
// 3. 返回指定数量的订单记录
*/
func (dao *Dao) QueryCollectionTopNBid(ctx context.Context, chain string,
	userAddr string, collectionAddr string, num int) ([]multi.Order, error) {
	var bestBids []multi.Order
	var sql string

	if userAddr == "" {
		// SQL解释:
		// 1. 查询订单基本信息(订单ID、价格、时间、过期时间等)
		// 2. 条件:
		//   - 指定集合地址
		//   - 订单类型为集合出价单
		//   - 订单状态为活跃
		//   - 剩余数量大于0
		//   - 未过期
		// 3. 按价格降序排序并限制返回记录数
		sql = fmt.Sprintf(`
			SELECT order_id, price, event_time, expire_time, salt, maker, 
				order_type, quantity_remaining, size 
			FROM %s
			WHERE collection_address = ?
				AND order_type = ?
				AND order_status = ?
				AND quantity_remaining > 0
				AND expire_time > ? 
			ORDER BY price DESC 
			LIMIT %d
		`, multi.OrderTableName(chain), num)
	} else {
		// SQL与上面类似,增加了排除指定用户的条件(maker != userAddr)
		sql = fmt.Sprintf(`
			SELECT order_id, price, event_time, expire_time, salt, maker, 
				order_type, quantity_remaining, size
			FROM %s
			WHERE collection_address = ?
				AND order_type = ?
				AND order_status = ?
				AND quantity_remaining > 0
				AND expire_time > ? 
				AND maker != '%s'
			ORDER BY price DESC 
			LIMIT %d
		`, multi.OrderTableName(chain), userAddr, num)
	}
	//执行sql查询
	err := dao.DB.WithContext(ctx).
		Raw(sql, multi.CollectionBidOrder, multi.OrderStatusActive, time.Now().Unix()).
		Scan(&bestBids).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on get item best bids")
	}
	//根据剩余数量展开订单
	var result []multi.Order
	for i := 0; i < len(bestBids); i++ {
		for j := 0; j < int(bestBids[i].QuantityRemaining); j++ {
			result = append(result, bestBids[i])
		}
	}
	//返回指定数量的订单
	if num < len(result) {
		result = result[:num]
	}
	return result, nil
}
