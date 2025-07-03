package dao

import (
	"EasySwapBackend-test/src/entity"
	"context"
	"fmt"
	"github.com/ProjectsTask/EasySwapBase/ordermanager"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"strings"
	"time"
)

const MaxBatchReadCollections = 500
const MaxRetries = 3
const QueryTimeout = time.Second * 30

var collectionDetailFields = []string{"id", "chain_id", "token_standard", "name", "address", "image_uri", "floor_price", "sale_price", "item_amount", "owner_amount"}

// 查询所有集合信息
func (dao *Dao) QueryAllCollectionInfo(ctx context.Context, chain string) ([]multi.Collection, error) {
	//设置超时时间
	ctx, cancelFunc := context.WithTimeout(ctx, QueryTimeout)
	defer cancelFunc()
	//开启事务
	tx := dao.DB.WithContext(ctx).Begin()
	//捕获异常
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback() //回滚事务
			panic(err)
		}
	}()

	var allCollections []multi.Collection
	cursor := int64(0) // 游标
	// 循环分页查询所有集合信息
	for {
		var collections []multi.Collection
		// 最多重试MaxRetries次查询
		for i := 0; i < MaxRetries; i++ {
			// 查询大于当前cursor的MaxBatchReadCollections条记录
			err := tx.Table(multi.CollectionTableName(chain)).
				Select(collectionDetailFields).
				Where("id > ?", cursor).
				Limit(MaxBatchReadCollections).
				Order("id asc").
				Scan(&collections).Error
			//如果查询成功，则跳出当前重试循环
			if err == nil {
				break
			}
			// 达到最大重试次数仍失败,则回滚事务并返回错误
			if i == MaxRetries-1 {
				tx.Rollback()
				return nil, errors.Wrap(err, "failed on get collections info")
			}
			// 重试间隔时间递增
			time.Sleep(time.Duration(i+1) * time.Second)
		}
		//将本次查询结果追加到总结果中
		allCollections = append(allCollections, collections...)
		//如果本次查询结果数小于批次大小,说明已经查完所有记录
		if len(collections) < MaxBatchReadCollections {
			break
		}
		// 更新游标为最后一条记录的ID
		cursor = collections[len(collections)-1].Id
	}
	return allCollections, nil
}

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

// 查询所有集合的最高卖单价格
func (dao *Dao) QueryCollectionsSellPrice(ctx context.Context, chain string) ([]multi.Collection, error) {
	var collections []multi.Collection
	sql := fmt.Sprintf(`SELECT
	co.collection_address AS address,
	max(co.price) AS sale_price 
FROM
	%s AS co 
WHERE
	order_status = ? 
	AND co.order_type = ? 
	AND co.expire_time > ? 
group by collection_address`, multi.OrderTableName(chain))

	err := dao.DB.WithContext(ctx).Raw(
		sql,
		multi.OrderStatusActive,
		multi.CollectionBidOrder,
		time.Now().Unix()).
		Scan(&collections).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on get collection sell price")
	}
	return collections, nil
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

// 查询指定时间段 NFT的历史销售价格
func (dao *Dao) QueryHistorySalesPriceInfo(ctx context.Context, chain, collectionAddr string, durationTimeStamp int64) ([]multi.Activity, error) {
	var historySalesPrice []multi.Activity
	now := time.Now().Unix()

	err := dao.DB.WithContext(ctx).
		Table(multi.ActivityTableName(chain)).
		Select("price,token_id,event_time").
		Where("activity_type = ? and collection_address = ? and event_time >= ? and event_time <= ?",
			multi.Sale, collectionAddr, now-durationTimeStamp, now).
		Scan(&historySalesPrice).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on get history sales info")
	}
	return historySalesPrice, nil
}

// 查询集合地板价变化情况
func (dao *Dao) QueryCollectionFloorChange(chain string, timeDiff int64) (map[string]float64, error) {
	collectionFloorChange := make(map[string]float64)
	var collectionPrices []multi.CollectionFloorPrice
	// 这个SQL语句用于查询NFT集合的地板价变化情况:
	// 1. 从集合地板价表中选择collection_address(集合地址)、price(价格)和event_time(事件时间)
	// 2. WHERE子句包含两个条件:
	//    a) 查询每个集合的最新地板价记录(通过GROUP BY和MAX(event_time)获取)
	//    b) 查询每个集合在指定时间段之前的最新地板价记录(通过WHERE event_time <= UNIX_TIMESTAMP() - ? 筛选)
	// 3. 最后按集合地址和时间降序排序,这样可以方便计算价格变化率
	rawSql := fmt.Sprintf(`SELECT collection_address, price, event_time 
		FROM %s 
		WHERE (collection_address, event_time) IN (
			SELECT collection_address, MAX(event_time)
			FROM %s
			GROUP BY collection_address
		) OR (collection_address, event_time) IN (
			SELECT collection_address, MAX(event_time)
			FROM %s 
			WHERE event_time <= UNIX_TIMESTAMP() - ? 
			GROUP BY collection_address
		) 
		ORDER BY collection_address,event_time DESC`,
		multi.CollectionFloorPriceTableName(chain),
		multi.CollectionFloorPriceTableName(chain),
		multi.CollectionFloorPriceTableName(chain))

	err := dao.DB.Raw(rawSql, timeDiff).Scan(&collectionPrices).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on get collection floor change")
	}
	// 这个循环用于计算每个NFT集合的地板价变化率:
	// 1. 遍历collectionPrices数组,每个元素包含集合地址和对应时间点的地板价
	// 2. 对于每个集合:
	//    - 如果当前元素和下一个元素是同一个集合的记录(CollectionAddress相同)
	//    - 且下一个元素的价格大于0
	//    则:
	//    - 计算价格变化率 = (当前价格 - 历史价格) / 历史价格
	//    - 使用Price.Sub()计算价格差
	//    - 使用Div()计算变化率
	//    - 使用InexactFloat64()转换为float64类型
	//    - i++跳过下一个元素(因为已经使用过了)
	// 3. 如果不满足条件,则将该集合的变化率设为0
	// 4. 最终得到一个从集合地址到其地板价变化率的映射
	for i := 0; i < len(collectionPrices); i++ {
		if i < len(collectionPrices)-1 &&
			collectionPrices[i].CollectionAddress == collectionPrices[i+1].CollectionAddress &&
			collectionPrices[i+1].Price.GreaterThan(decimal.Zero) {
			collectionFloorChange[strings.ToLower(collectionPrices[i].CollectionAddress)] = collectionPrices[i].Price.
				Sub(collectionPrices[i+1].Price).
				Div(collectionPrices[i+1].Price).
				InexactFloat64()
		} else {
			collectionFloorChange[strings.ToLower(collectionPrices[i].CollectionAddress)] = 0.0
		}
	}
	return collectionFloorChange, nil
}

// 查询集合上架数量
func (dao *Dao) QueryCollectionsListed(chain string, collectionAddrs []string) ([]entity.CollectionListed, error) {
	var collectionListed []entity.CollectionListed
	if len(collectionAddrs) == 0 {
		return collectionListed, nil
	}
	for _, address := range collectionAddrs {
		count, err := dao.KvStore.GetInt(ordermanager.GenCollectionListedKey(chain, address))
		if err != nil {
			return nil, errors.Wrap(err, "failed on set collection listed count")
		}
		collectionListed = append(collectionListed, entity.CollectionListed{
			CollectionAddr: address,
			Count:          count,
		})
	}
	return collectionListed, nil
}

// 查询用户在多条链上的collection基本信息
func (dao *Dao) QueryMultiChainUserCollectionInfos(ctx context.Context, chainIds []int, chainNames, userAddrs []string) ([]entity.UserCollections, error) {
	var userCollections []entity.UserCollections

	//1、构建查询sql头部
	sqlHead := "select * from ("
	//2、构建查询sql中部 - 使用UNION ALL合并多个链的查询
	sqlMid := ""
	for _, chain := range chainNames {
		//为每个链构建子查询
		if sqlMid != "" {
			sqlMid += " union all "
		}
		//查询字段
		sqlMid += "(select " +
			"gc.address as address, " +
			"gc.name as name, " +
			"gc.floor_price as floor_price, " +
			"gc.chain_id as chain_id, " +
			"gc.item_amount as item_amount, " +
			"gc.symbol as symbol, " +
			"gc.image_uri as image_uri, " +
			"count(*) as item_count "
		//从Collection表和Item表联表查询
		sqlMid += fmt.Sprintf("from %s as gc join %s as gi ", multi.CollectionTableName(chain), multi.ItemTableName(chain))
		sqlMid += "on gc.address = gi.collection_address "
		// 过滤指定用户持有的Item
		sqlMid += fmt.Sprintf("where gi.owner in ('%s' ", userAddrs[0])
		for i := 1; i < len(userAddrs); i++ {
			sqlMid += fmt.Sprintf(", '%s' ", userAddrs[i])
		}
		sqlMid += ") group by gc.address) "
	}
	//3、构建查询sql尾部
	// 按照地板价*持有数量降序排序
	sqlTail := ") as combined ORDER BY combined.floor_price * CAST(combined.item_count AS DECIMAL) DESC"
	//4、组合查询sql
	sql := sqlHead + sqlMid + sqlTail
	//5、执行sql
	err := dao.DB.Raw(sql).Scan(&userCollections).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on get user multi chain collection infos")
	}
	return userCollections, nil
}
