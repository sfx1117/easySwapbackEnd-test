package dao

import (
	"context"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/base"
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
