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
