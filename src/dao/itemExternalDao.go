package dao

import (
	"EasySwapBackend-test/src/entity"
	"context"
	"fmt"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"
	"strings"
)

// 查询集合内NFT Item的图片和视频信息
func (dao *Dao) QueryCollectionItemImage(ctx context.Context, chain, collectionAddr string, tokenIds []string) ([]multi.ItemExternal, error) {
	var itemExternal []multi.ItemExternal
	err := dao.DB.WithContext(ctx).
		Table(multi.ItemExternalTableName(chain)).
		Select("collection_address, token_id, is_uploaded_oss, image_uri, oss_uri, "+
			"video_type, is_video_uploaded, video_uri, video_oss_uri").
		Where("collection_address = ? and token_id in (?)", collectionAddr, tokenIds).
		Scan(&itemExternal).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on query items external info")
	}
	return itemExternal, nil
}

/*
*
查询多条链上NFT Item的图片信息
// 主要功能:
// 1. 按链名称对输入的Item信息进行分组
// 2. 构建多条链的联合查询SQL
// 3. 返回所有链上Item的图片信息
*/
func (dao *Dao) QueryMultiChainCollectionsItemsImage(ctx context.Context, itemInfos []entity.MultiChainItemInfo) ([]multi.ItemExternal, error) {
	var itemExternal []multi.ItemExternal
	//1、按链名称对Item信息分组
	chainItemMap := make(map[string][]entity.ItemInfo)
	for _, item := range itemInfos {
		infos, ok := chainItemMap[strings.ToLower(item.ChainName)]
		if ok {
			infos = append(infos, item.ItemInfo)
			chainItemMap[strings.ToLower(item.ChainName)] = infos
		} else {
			chainItemMap[strings.ToLower(item.ChainName)] = []entity.ItemInfo{item.ItemInfo}
		}
	}
	//2 构建查询sql
	//2.1 构建sql头
	sqlHead := "select * from ("
	//2.2 构建sql中部
	sqlMid := ""
	for chain, infos := range chainItemMap {
		if sqlMid != "" {
			sqlMid += " union all "
		}

		sqlMid += "(select collection_address, token_id, is_uploaded_oss, image_uri, oss_uri "
		sqlMid += fmt.Sprintf("from %s ", multi.ItemExternalTableName(chain))
		sqlMid += "where (collection_address,token_id) in "
		sqlMid += fmt.Sprintf("(('%s','%s')", infos[0].CollectionAddress, infos[0].TokenID)
		for i := 1; i < len(infos); i++ {
			sqlMid += fmt.Sprintf(",('%s','%s')", infos[i].CollectionAddress, infos[i].TokenID)
		}
		sqlMid += "))"
	}
	//2.3 构建sql尾部
	sqlTail := ") as combined"
	//2.4组合sql
	sql := sqlHead + sqlMid + sqlTail

	//3 执行sql
	err := dao.DB.WithContext(ctx).Raw(sql).Scan(&itemExternal).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed on query items external info")
	}
	return itemExternal, nil
}
