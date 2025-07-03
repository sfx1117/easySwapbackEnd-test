package cached

import (
	"github.com/ProjectsTask/EasySwapBase/ordermanager"
	"github.com/pkg/errors"
)

// 缓存集合的上架数量
func (cached *Cached) CacheCollectionsListed(chain, collectionAddr string, listedCount int) error {
	err := cached.KvStore.SetInt(ordermanager.GenCollectionListedKey(chain, collectionAddr), listedCount)
	if err != nil {
		return errors.Wrap(err, "failed on set collection listed count")
	}
	return nil
}

// 获取缓存 集合的上架数量
func (cached *Cached) GetCollectionsListed(chain, collectionAddr string) (int, error) {
	listedCount, err := cached.KvStore.GetInt(ordermanager.GenCollectionListedKey(chain, collectionAddr))
	if err != nil {
		return 0, errors.Wrap(err, "failed on get collection listed count")
	}
	return listedCount, nil
}
