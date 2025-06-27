package service

import (
	"EasySwapBackend-test/src/entity"
	"EasySwapBackend-test/src/middleware"
	"EasySwapBackend-test/src/svc"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/ProjectsTask/EasySwapBase/errcode"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/base"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"io"
	"strings"
)

// LoginMsg缓存key
func getUserLoginMsgCacheKey(address string) string {
	return middleware.CR_LOGIN_MSG_KEY + ":" + strings.ToLower(address)
}

// LoginToken缓存key
func getUserLoginTokenCacheKey(address string) string {
	return middleware.CR_LOGIN_TOKEN_KEY + ":" + strings.ToLower(address)
}

/*
*
生成login签名信息，并将uuid写如reids缓存
*/
func GetLoginMessage(ctx context.Context, serverCtx *svc.ServerCtx, address string) (*entity.UserLoginMessageRes, error) {
	//生成uuid
	uuid := uuid.NewString()
	//包装uuid作为message返回
	loginMsg := getLoginTemplate(uuid)
	//将uuid写入redis
	err := serverCtx.KvStore.Setex(getUserLoginMsgCacheKey(address), uuid, 72*60*60)
	if err != nil {
		return nil, errors.Wrap(err, "failed on generate login msg")
	}
	return &entity.UserLoginMessageRes{Address: address, Message: loginMsg}, nil
}

// 包装uuid
func getLoginTemplate(uuid string) string {
	return fmt.Sprintf("Welcome to EasySwap!\nNonce:%s", uuid)
}

/*
*
登录核心方法
*/
func UserLogin(ctx context.Context, serverCtx *svc.ServerCtx, req entity.LoginReq) (*entity.UserLoginInfo, error) {
	//返回结果
	res := entity.UserLoginInfo{}

	//todo: add verify signature
	//ok := verifySignature(req.Message, req.Signature, req.PublicKey)
	//if !ok {
	//	return nil, errors.New("invalid signature")
	//}

	//从缓存中获取登录消息uuid
	cachedUUID, err := serverCtx.KvStore.Get(getUserLoginMsgCacheKey(req.Address))
	if cachedUUID == "" || err != nil {
		return nil, errcode.ErrTokenExpire
	}

	//从入参中获取uuid
	split := strings.Split(req.Message, "Nonce:")
	if len(split) != 2 {
		return nil, errcode.ErrTokenExpire
	}
	//获取登录UUID并和缓存中的uuid做校验
	loginUUID := strings.Trim(split[1], "\n")
	if loginUUID != cachedUUID {
		return nil, errcode.ErrTokenExpire
	}

	//从数据库查询用户信息
	var user base.User
	db := serverCtx.DB.WithContext(ctx).Table(base.UserTableName()).
		Select("id,address,is_allowed").
		Where("address = ?", req.Address).
		Find(&user)
	if db.Error != nil {
		return nil, errors.Wrap(db.Error, "failed on get user info")
	}

	//如果用户不存在，则创建新用户
	if user.Id == 0 {
		err := serverCtx.Dao.AddUser(ctx, req.Address)
		if err != nil {
			return nil, errors.Wrap(err, "failed on create new user")
		}
	}

	//生成token
	tokenKey := getUserLoginTokenCacheKey(req.Address)
	userToken, err := AesEncryptOFB([]byte(tokenKey), []byte(middleware.CR_LOGIN_SALT))
	if err != nil {
		return nil, errors.Wrap(err, "failed on get user token")
	}

	//缓存token
	err = CacheUserToken(serverCtx, tokenKey, uuid.NewString())
	if err != nil {
		return nil, err
	}
	//设置返回结果
	res.Token = hex.EncodeToString(userToken)
	res.IsAllowed = user.IsAllowed
	return &res, nil
}

// 将token写入缓存redis
func CacheUserToken(serverCtx *svc.ServerCtx, tokenKey string, token string) error {
	err := serverCtx.KvStore.Setex(tokenKey, token, 30*24*60*60)
	if err != nil {
		return err
	}
	return nil
}

// 加密
func AesEncryptOFB(data []byte, key []byte) ([]byte, error) {
	data = PKCS7Padding(data, aes.BlockSize)
	block, _ := aes.NewCipher([]byte(key))
	out := make([]byte, aes.BlockSize+len(data))
	iv := out[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewOFB(block, iv)
	stream.XORKeyStream(out[aes.BlockSize:], data)
	return out, nil
}

// 补码
// AES加密数据块分组长度必须为128bit(byte[16])，密钥长度可以是128bit(byte[16])、192bit(byte[24])、256bit(byte[32])中的任意一个。
func PKCS7Padding(ciphertext []byte, blocksize int) []byte {
	padding := blocksize - len(ciphertext)%blocksize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

/*
*
获取用户登录状态
*/
func GetUserSignStatus(ctx context.Context, serverCtx *svc.ServerCtx, address string) (*entity.UserSignStatusRes, error) {
	isSigned, err := serverCtx.Dao.GetUserSignStatus(ctx, address)
	if err != nil {
		return nil, errors.Wrap(err, "failed on get user sign status")
	}
	return &entity.UserSignStatusRes{IsSigned: isSigned}, nil
}
