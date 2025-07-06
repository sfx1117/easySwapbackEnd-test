package dao

import (
	"context"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/base"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"
	"time"
)

func (dao *Dao) AddUser(ctx context.Context, address string) error {
	now := time.Now().UnixMilli()
	user := &base.User{
		Address:    address,
		IsAllowed:  false,
		IsSigned:   true,
		CreateTime: now,
		UpdateTime: now,
	}
	err := dao.DB.WithContext(ctx).Table(base.UserTableName()).Create(user).Error
	if err != nil {
		return errors.Wrap(err, "failed on create new user")
	}
	return nil
}

// 获取user签名状态
func (dao *Dao) GetUserSignStatus(ctx context.Context, address string) (bool, error) {
	var user base.User
	err := dao.DB.WithContext(ctx).Table(base.UserTableName()).
		Where("address = ?", address).
		Find(&user).
		Error
	if err != nil {
		return false, errors.Wrap(err, "failed on get user info")
	}
	return user.IsSigned, nil
}

// 查询用户的出价订单信息
func (dao *Dao) QueryUserBids(ctx context.Context, chain string, userAddr []string, collectionAddrs []string) ([]multi.Order, error) {
	var userBids []multi.Order
	// SQL解释:
	// 1. 从订单表中查询订单详细信息
	// 2. 选择字段包括:集合地址、代币ID、订单ID、订单类型、剩余数量等
	// 3. WHERE条件:
	//    - maker在给定用户地址列表中
	//    - 订单类型为Item出价或集合出价
	//    - 订单状态为活跃
	//    - 剩余数量大于0
	err := dao.DB.WithContext(ctx).
		Table(multi.OrderTableName(chain)).
		Select("collection_address, token_id, order_id, token_id,order_type,"+
			"quantity_remaining, size, event_time, price, salt, expire_time").
		Where("maker in (?) and order_type in (?,?) and order_status = ? and quantity_remaining > 0",
			userAddr, multi.ItemBidOrder, multi.CollectionBidOrder, multi.OrderStatusActive).
		Scan(&userBids).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on get user bids info")
	}
	return userBids, nil
}
