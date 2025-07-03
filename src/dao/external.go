package dao

import (
	"context"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"
)

// QueryCollectionItemImage 查询集合内NFT Item的图片和视频信息
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
