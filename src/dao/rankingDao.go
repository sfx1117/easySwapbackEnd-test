package dao

import (
	"context"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"time"
)

type CollectionTrade struct {
	ContractAddress string          `json:"contract_address"`
	ItemCount       int64           `json:"item_count"`
	Volume          decimal.Decimal `json:"volume"`
	VolumeChange    int             `json:"volume_change"`
	PreFlooPrice    decimal.Decimal `json:"pre_fool_price"`
	FlooChange      int             `json:"fool_change"`
}

type periodEpochMap map[string]int

var periodToEpoch = periodEpochMap{
	"15m": 3,
	"1h":  12,
	"6h":  72,
	"24h": 288,
	"1d":  288,
	"7d":  2016,
	"30d": 8640,
}

// 获取指定时间段内集合的交易统计信息
func (dao *Dao) GetTradeInfoByCollection(chain, collectionAddr, period string) (*CollectionTrade, error) {
	//查询当前时间段的交易信息
	var tradeCount int64
	var totalVolume decimal.Decimal
	var flooPrice decimal.Decimal

	//获取时间段对应的epoch值
	epoch, ok := periodToEpoch[period]
	if !ok {
		return nil, errors.Errorf("invalid period: %s", period)
	}
	//计算查询的时间范围
	startTime := time.Now().Add(-time.Duration(epoch) * time.Minute)
	endTime := time.Now()

	//统计当前时间段内的交易数量和交易总额
	err := dao.DB.WithContext(dao.ctx).Table(multi.ActivityTableName(chain)).
		Select("COUNT(*) as trade_count,COALESCE(SUM(price),0) as total_volume").
		Where("collection_address = ? and activity_type = ? and event_time >= ? and event_time <= ?",
			collectionAddr, multi.Sale, startTime, endTime).
		Row().Scan(&tradeCount, &totalVolume)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get trade count and volume")
	}
	//获取当前时间段内的地板价（交易最低价）
	err = dao.DB.WithContext(dao.ctx).Table(multi.ActivityTableName(chain)).
		Select("COALESCE(MIN(price),0)").
		Where("collection_address = ? and activity_type = ? and event_time >= ? and event_time <= ?",
			collectionAddr, multi.Sale, startTime, endTime).
		Row().Scan(&flooPrice)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get floor price")
	}

	//计算上一个时间段的时间范围
	prevStartTime := startTime.Add(-time.Duration(epoch) * time.Minute)
	prevEndTime := startTime
	//上一个时间段的交易总额
	var prevVolume decimal.Decimal
	//上一个时间段的地板价
	var prevFlooPrice decimal.Decimal
	//获取上一个时间段内的交易总额
	err = dao.DB.WithContext(dao.ctx).Table(multi.ActivityTableName(chain)).
		Select("COALESCE(SUM(price),0)").
		Where("collection_address = ? and activity_type = ? and event_time >= ? and event_time <= ?",
			collectionAddr, multi.Sale, prevStartTime, prevEndTime).
		Row().Scan(&prevVolume)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get previous volume")
	}
	//获取上一个时间段内的地板价
	err = dao.DB.WithContext(dao.ctx).Table(multi.ActivityTableName(chain)).
		Select("COALESCE(MIN(price),0)").
		Where("collection_address = ? and activity_type = ? and event_time >= ? and event_time <= ?",
			collectionAddr, multi.Sale, prevStartTime, prevEndTime).
		Row().Scan(&prevFlooPrice)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get previous floor price")
	}

	//计算交易总额和地板价的变化百分比
	volumeChange := 0
	flooPriceChange := 0
	//如果上一个时间段交易总额不为0
	if !prevVolume.IsZero() {
		volumeChangeDecimal := totalVolume.Sub(prevVolume).Div(prevVolume).Mul(decimal.NewFromInt(100))
		volumeChange = int(volumeChangeDecimal.IntPart())
	}
	//如果上一个时间段地板价不为0
	if !prevVolume.IsZero() {
		flooChangeDecimal := flooPrice.Sub(prevFlooPrice).Div(prevFlooPrice).Mul(decimal.NewFromInt(100))
		flooPriceChange = int(flooChangeDecimal.IntPart())
	}

	// 返回集合交易统计信息
	return &CollectionTrade{
		ContractAddress: collectionAddr,
		ItemCount:       tradeCount,
		Volume:          totalVolume,
		VolumeChange:    volumeChange,
		PreFlooPrice:    prevFlooPrice,
		FlooChange:      flooPriceChange,
	}, nil
}

// 获取指定集合的总交易量
func (dao *Dao) QueryCollectionVolume(ctx context.Context, chain, collectionAddr string) (decimal.Decimal, error) {
	var volume decimal.Decimal
	err := dao.DB.WithContext(ctx).Table(multi.ActivityTableName(chain)).
		Select("COALESCE(SUM(price), 0)").
		Where("collection_address = ? AND activity_type = ?", collectionAddr, multi.Sale).
		Row().Scan(&volume)
	if err != nil {
		return decimal.Zero, errors.Wrap(err, "failed to get collection volume")
	}
	return volume, nil
}
