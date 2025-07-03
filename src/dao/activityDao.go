package dao

import (
	"EasySwapBackend-test/src/entity"
	"EasySwapBackend-test/src/utils"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strconv"
	"strings"
	"sync"
)

var eventTypesToID = map[string]int{
	"sale":                  multi.Sale,
	"transfer":              multi.Transfer,
	"offer":                 multi.MakeOffer,
	"cancel_offer":          multi.CancelOffer,
	"cancel_list":           multi.CancelListing,
	"list":                  multi.Listing,
	"mint":                  multi.Mint,
	"buy":                   multi.Buy,
	"collection_bid":        multi.CollectionBid,
	"item_bid":              multi.ItemBid,
	"cancel_collection_bid": multi.CancelCollectionBid,
	"cancel_item_bid":       multi.CancelItemBid,
}

var idToEventTypes = map[int]string{
	multi.Sale:                "sale",
	multi.Transfer:            "transfer",
	multi.MakeOffer:           "offer",
	multi.CancelOffer:         "cancel_offer",
	multi.CancelListing:       "cancel_list",
	multi.Listing:             "list",
	multi.Mint:                "mint",
	multi.Buy:                 "buy",
	multi.CollectionBid:       "collection_bid",
	multi.ItemBid:             "item_bid",
	multi.CancelCollectionBid: "cancel_collection_bid",
	multi.CancelItemBid:       "cancel_item_bid",
}

type ActivityMultiChainInfo struct {
	multi.Activity
	ChainName string `gorm:"column:chain_name"`
}

// 计数缓存key前缀
const CacheActivityNumPrefix = "cache:es:activity:count:"

type ActivityCountCache struct {
	Chain             string   `json:"chain"`
	ContractAddresses []string `json:"contract_addresses"`
	TokenId           string   `json:"token_id"`
	UserAddress       string   `json:"user_address"`
	EventTypes        []string `json:"event_types"`
}

// 获取计数缓存key
func getActivityCountCacheKey(activity *ActivityCountCache) (string, error) {
	uid, err := json.Marshal(activity)
	if err != nil {
		return "", errors.Wrap(err, "failed on marshal activity struct")
	}
	return CacheActivityNumPrefix + string(uid), nil
}

// 查询多链上的活动信息
func (dao *Dao) QueryMultiChainActivities(ctx context.Context, chains []string, filter entity.ActivityMultiChainFilterParams) ([]ActivityMultiChainInfo, int64, error) {
	//解析入参
	collectionAddrs := filter.CollectionAddresses
	tokenId := filter.TokenID
	userAddrs := filter.UserAddresses
	eventTypes := filter.EventTypes
	page := filter.Page
	pageSize := filter.PageSize

	//声明返回参数
	var total int64
	var activities []ActivityMultiChainInfo

	//1、将事件类型转换为对应的id
	var events []int
	for _, e := range eventTypes {
		id, ok := eventTypesToID[e]
		if ok {
			events = append(events, id)
		}
	}
	//2、构建查询sql
	//2.1、构建sql头部
	sqlHead := "select * from ("

	//2.2、构建SQL中间部分 - 使用UNION ALL合并多个链的查询
	sqlMid := ""
	for _, chain := range chains {
		//为每个链构建子查询
		if sqlMid != "" {
			sqlMid += " union all "
		}
		sqlMid += fmt.Sprintf("(select '%s' as chain_name,id,collection_address,token_id,currency_address,"+
			"activity_type,maker,taker,price,tx_hash,event_time,marketplace_id from %s ",
			chain, multi.ActivityTableName(chain))
		//添加用户地址过滤条件
		if len(userAddrs) == 1 {
			sqlMid += fmt.Sprintf("where maker = '%s' or taker = '%s'",
				strings.ToLower(userAddrs[0]), strings.ToLower(userAddrs[0]))
		} else if len(userAddrs) > 1 {
			var userAddrParam string
			for i, address := range userAddrs {
				userAddrParam += fmt.Sprintf("`%s`", address)
				if i < len(userAddrs)-1 {
					userAddrParam += ","
				}
			}
			sqlMid += fmt.Sprintf("where maker in (%s) or taker in (%s)", userAddrParam, userAddrParam)
		}
		sqlMid += ") "
	}
	//2.3、构建sql尾部--添加过滤条件
	sqlTail := ") as combind"
	firstFlag := true
	//添加合约地址过滤条件
	if len(collectionAddrs) == 1 {
		sqlTail += fmt.Sprintf("WHERE collection_address = '%s' ", collectionAddrs[0])
		firstFlag = false
	} else if len(collectionAddrs) > 1 {
		var collectionAddrParam string
		for i, address := range collectionAddrs {
			collectionAddrParam += fmt.Sprintf("`%s`", address)
			if i < len(userAddrs)-1 {
				collectionAddrParam += ","
			}
		}
		sqlTail += fmt.Sprintf("WHERE collection_address in (%s) ", collectionAddrParam)
		firstFlag = false
	}
	//添加tokenId过滤条件
	if tokenId != "" {
		if firstFlag {
			sqlTail += fmt.Sprintf("where token_id = '%s' ", tokenId)
			firstFlag = false
		} else {
			sqlTail += fmt.Sprintf("and token_id = '%s' ", tokenId)
		}
	}
	//添加事件类型过滤条件
	if len(events) > 0 {
		if firstFlag {
			sqlTail += fmt.Sprintf("where activity_type in ('%d'", events[0])
			for i := 1; i < len(events); i++ {
				sqlTail += fmt.Sprintf(",`%d`", events[i])
			}
			sqlTail += ") "
			firstFlag = false
		} else {
			sqlTail += fmt.Sprintf("and activity_type in ('%d'", events[0])
			for i := 1; i < len(events); i++ {
				sqlTail += fmt.Sprintf(",`%d`", events[i])
			}
			sqlTail += ") "
		}
	}
	//添加分页
	sqlTail += fmt.Sprintf("order by combined.event_time DESC, "+
		"combined.id DESC limit %d offset %d", pageSize, pageSize*(page-1))
	//2.4、组合完整sql
	sql := sqlHead + sqlMid + sqlTail
	//2.5、执行sql
	err := dao.DB.Raw(sql).Scan(&activities).Error
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed on query activity")
	}

	//3、获取记录数
	//3.1、从redis中获取
	cacheKey, err := getActivityCountCacheKey(&ActivityCountCache{
		Chain:             "MultiChain",
		ContractAddresses: collectionAddrs,
		TokenId:           tokenId,
		UserAddress:       strings.ToLower(strings.Join(userAddrs, ",")),
		EventTypes:        eventTypes,
	})
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed on get activity number cache key")
	}
	strNum, err := dao.KvStore.Get(cacheKey)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed on get activity number from cache")
	}
	if strNum != "" {
		total, _ = strconv.ParseInt(strNum, 10, 64)
	} else {
		//3.2、查数据库
		//构建计数sql
		sqlCnt := "select count(*) from (" + sqlMid + sqlTail
		//执行计数sql
		err := dao.DB.Raw(sqlCnt).Scan(&total).Error
		if err != nil {
			return nil, 0, errors.Wrap(err, "failed on count activity")
		}
		//将total写入redis
		err = dao.KvStore.Setex(cacheKey, strconv.FormatInt(total, 10), 30)
		if err != nil {
			return nil, 0, errors.Wrap(err, "failed on cache activities number")
		}
	}
	return activities, total, nil
}

// 查询多链活动的外部信息
// 包括: 用户地址、NFT信息、合约信息等
func (dao *Dao) QueryMultiChainActivityExternalInfo(ctx context.Context, chainID []int, chains []string, activities []ActivityMultiChainInfo) ([]entity.ActivityInfo, error) {
	//1、收集需要查询的地址和id
	var userAddrs [][]string
	var items [][]string
	var collectionAddrs [][]string
	for _, activity := range activities {
		userAddrs = append(userAddrs, []string{activity.Maker, activity.ChainName}, []string{activity.Taker, activity.ChainName})
		items = append(items, []string{activity.CollectionAddress, activity.TokenId, activity.ChainName})
		collectionAddrs = append(collectionAddrs, []string{activity.CollectionAddress, activity.ChainName})
	}
	//1.1、去重
	userAddrs = utils.RemoveRepeatedElementArr(userAddrs)
	items = utils.RemoveRepeatedElementArr(items)
	collectionAddrs = utils.RemoveRepeatedElementArr(collectionAddrs)

	//2、并发查询外部信息
	var wg sync.WaitGroup
	var queryErr error
	collections := make(map[string]multi.Collection)
	itemsInfos := make(map[string]multi.Item)
	itemExternals := make(map[string]multi.ItemExternal)
	//构建item查询条件
	var itemQuery []clause.Expr
	for _, item := range items {
		itemQuery = append(itemQuery, gorm.Expr("(?,?)", item[0], item[1]))
	}
	//2.1、查询item信息
	wg.Add(1)
	go func() {
		defer wg.Done()
		var newItem multi.Item
		for i := 0; i < len(itemQuery); i++ {
			err := dao.DB.WithContext(ctx).
				Table(multi.ItemTableName(items[i][2])).
				Select("collection_address, token_id, name").
				Where("(collection_address,token_id) = ?", itemQuery[i]).
				Scan(&newItem).Error
			if err != nil {
				queryErr = errors.Wrap(err, "failed on query items info")
				return
			}
			itemsInfos[strings.ToLower(newItem.CollectionAddress+newItem.TokenId)] = newItem
		}
	}()
	//2.2、查询item外部信息（图片等）
	wg.Add(1)
	go func() {
		defer wg.Done()
		var newItemExternal multi.ItemExternal
		for _, item := range items {
			err := dao.DB.WithContext(ctx).
				Table(multi.ItemExternalTableName(item[2])).
				Select("collection_address, token_id, is_uploaded_oss, image_uri, oss_uri").
				Where("collection_address = ? and token_id = ?", item[0], item[1]).
				Scan(&newItemExternal).Error
			if err != nil {
				queryErr = errors.Wrap(err, "failed on query items external info")
				return
			}
			itemExternals[strings.ToLower(newItemExternal.CollectionAddress+newItemExternal.TokenId)] = newItemExternal
		}
	}()
	//2.3、查询collections信息
	wg.Add(1)
	go func() {
		defer wg.Done()
		var newColl multi.Collection
		for _, collection := range collectionAddrs {
			err := dao.DB.WithContext(ctx).
				Table(multi.CollectionTableName(collection[1])).
				Select("id, name, address, image_uri").
				Where("address = ?", collection[0]).
				Scan(&newColl).Error
			if err != nil {
				queryErr = errors.Wrap(err, "failed on query collection info")
				return
			}
			collections[strings.ToLower(newColl.Address)] = newColl
		}
	}()
	//2.4、等待所有查询完成
	wg.Wait()
	if queryErr != nil {
		return nil, errors.Wrap(queryErr, "failed on query activity external info")
	}

	//3.包装返回参数
	//3.1、构建chain name到chain id的映射
	chainNameTochainId := make(map[string]int)
	for i, chain := range chains {
		chainNameTochainId[chain] = chainID[i]
	}
	//3.2、组装最总结果
	var result []entity.ActivityInfo
	for _, act := range activities {
		activityInfo := entity.ActivityInfo{
			EventType:         "unknown",
			EventTime:         act.EventTime,
			CollectionAddress: act.CollectionAddress,
			TokenID:           act.TokenId,
			Currency:          act.CurrencyAddress,
			Price:             act.Price,
			Maker:             act.Maker,
			Taker:             act.Taker,
			TxHash:            act.TxHash,
			MarketplaceID:     act.MarketplaceID,
			ChainID:           chainNameTochainId[act.ChainName],
		}
		// Listing类型活动不需要txHash
		if act.ActivityType == multi.Listing {
			activityInfo.TxHash = ""
		}
		//设置事件类型
		eventType, ok := idToEventTypes[act.ActivityType]
		if ok {
			activityInfo.EventType = eventType
		}
		//设置item名称
		item, ok := itemsInfos[strings.ToLower(act.CollectionAddress+act.TokenId)]
		if ok {
			activityInfo.ItemName = item.Name
		}
		if activityInfo.ItemName == "" {
			activityInfo.ItemName = fmt.Sprintf("#%s", act.TokenId)
		}
		//设置item图片信息
		itemExternal, ok := itemExternals[strings.ToLower(act.CollectionAddress+act.TokenId)]
		if ok {
			if itemExternal.IsUploadedOss {
				activityInfo.ImageURI = itemExternal.OssUri
			} else {
				activityInfo.ImageURI = itemExternal.ImageUri
			}
		}
		//设置collection信息
		collection, ok := collections[strings.ToLower(act.CollectionAddress)]
		if ok {
			activityInfo.CollectionName = collection.Name
			activityInfo.CollectionImageURI = collection.ImageUri
		}

		result = append(result, activityInfo)
	}
	return result, nil
}
