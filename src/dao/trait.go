package dao

import (
	"EasySwapBackend-test/src/entity"
	"context"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"
)

// 查询单个NFT Item的 Trait信息
func (dao *Dao) QueryItemTraits(ctx context.Context, chain, collectionAddr, tokenId string) ([]multi.ItemTrait, error) {
	var itemTraits []multi.ItemTrait
	err := dao.DB.WithContext(ctx).
		Table(multi.ItemTraitTableName(chain)).
		Select("collection_address, token_id, trait, trait_value").
		Where("collection_address = ? and token_id = ?", collectionAddr, tokenId).
		Scan(&itemTraits).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on query items trait info")
	}
	return itemTraits, nil
}

// 查询多个NFT Item的 Trait信息
func (dao *Dao) QueryItemsTraits(ctx context.Context, chain, collectionAddr string, tokenIds []string) ([]multi.ItemTrait, error) {
	var itemTraits []multi.ItemTrait
	err := dao.DB.WithContext(ctx).
		Table(multi.ItemTraitTableName(chain)).
		Select("collection_address, token_id, trait, trait_value").
		Where("collection_address = ? and token_id in (?)", collectionAddr, tokenIds).
		Scan(&itemTraits).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on query items trait info")
	}
	return itemTraits, nil
}

// 查询集合维度的 Trait信息统计
func (dao *Dao) QueryCollectionTraitCount(ctx context.Context, chain, collectionAddr string) ([]entity.TraitCount, error) {
	var traitCount []entity.TraitCount
	err := dao.DB.WithContext(ctx).
		Table(multi.ItemTraitTableName(chain)).
		Select("`trait`,`trait_value`,count(*) as count").
		Where("collection_address=?", collectionAddr).
		Group("`trait`,`trait_value`").
		Scan(&traitCount).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on query collection trait amount")
	}
	return traitCount, nil
}
