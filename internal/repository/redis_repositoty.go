package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/CoolBitX-Technology/subscan/model"
	"github.com/CoolBitX-Technology/subscan/util"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/common/log"
)

type redisRepository struct {
	Redis *redis.Client
}

var (
	RedisMetadataKey           = redisKeyPrefix() + "metadata"
	RedisFillAlreadyBlockNum   = redisKeyPrefix() + "FillAlreadyBlockNum"
	RedisFillFinalizedBlockNum = redisKeyPrefix() + "FillFinalizedBlockNum"
	RedisMissingBlocksSet      = redisKeyPrefix() + "missing_blocks"
)

func NewRedisRepository(redisClient *redis.Client) model.RedisRepository {
	return &redisRepository{
		Redis: redisClient,
	}
}

func redisKeyPrefix() string {
	return util.NetworkNode + ":"
}

func (r *redisRepository) DaemonHealth(c context.Context) map[string]bool {
	status := map[string]bool{}
	for _, dt := range model.DaemonAction {
		cacheKey := fmt.Sprintf("%s:heartBeat:%s", util.NetworkNode, dt)
		g := r.Redis.Get(c, cacheKey)
		t, err := g.Int64()
		if err != nil || time.Now().Unix()-t > 60 {
			status[dt] = false
		} else {
			status[dt] = true
		}
	}
	return status
}

func (r *redisRepository) Close() {
	if r.Redis != nil {
		_ = r.Redis.Close()
	}
}

func (r *redisRepository) Ping(ctx context.Context) (err error) {
	if err = r.pingRedis(ctx); err != nil {
		return
	}
	return
}

func (r *redisRepository) SetHeartBeatNow(c context.Context, action string) error {
	return r.setCache(c, action, time.Now().Unix(), 300)
}

func (r *redisRepository) SaveFillAlreadyBlockNum(c context.Context, blockNum int) (err error) {
	g := r.Redis.Get(c, RedisFillAlreadyBlockNum)
	if num, _ := g.Int(); blockNum > num {
		err = r.Redis.Set(c, RedisFillAlreadyBlockNum, blockNum, 0).Err()
	}
	return err
}

func (r *redisRepository) SaveFillAlreadyFinalizedBlockNum(c context.Context, blockNum int) (err error) {
	g := r.Redis.Get(c, RedisFillFinalizedBlockNum)
	if num, _ := g.Int(); blockNum > num {
		err = r.Redis.Set(c, RedisFillFinalizedBlockNum, blockNum, 0).Err()
	}
	return
}

func (r *redisRepository) GetFillBestBlockNum(c context.Context) (num int, err error) {
	g := r.Redis.Get(c, RedisFillAlreadyBlockNum)
	num, err = g.Int()
	return
}

func (r *redisRepository) GetFillFinalizedBlockNum(c context.Context) (num int, err error) {
	g := r.Redis.Get(c, RedisFillFinalizedBlockNum)
	num, err = g.Int()
	return
}

func (r *redisRepository) SetMetadata(c context.Context, metadata map[string]interface{}) (err error) {
	err = r.Redis.HSet(c, RedisMetadataKey, metadata).Err()
	return
}

func (r *redisRepository) IncrMetadata(c context.Context, filed string, incrNum int) (err error) {
	if incrNum == 0 {
		return
	}
	err = r.Redis.HIncrBy(c, RedisMetadataKey, filed, int64(incrNum)).Err()
	return
}

func (r *redisRepository) GetMetadata(c context.Context) (ms map[string]string, err error) {
	g := r.Redis.HGetAll(c, RedisMetadataKey)
	ms, err = g.Result()
	return
}

func (r *redisRepository) GetBestBlockNum(c context.Context) (num uint64, err error) {
	g := r.Redis.HGet(c, RedisMetadataKey, "blockNum")
	num, err = g.Uint64()
	return
}

func (r *redisRepository) GetBestBlockNumForPlugin(c context.Context) (num uint64, err error) {
	g := r.Redis.HGet(c, RedisMetadataKey, "plugins:blockNum")
	num, err = g.Uint64()
	return
}

func (r *redisRepository) GetFinalizedBlockNum(c context.Context) (num uint64, err error) {
	g := r.Redis.HGet(c, RedisMetadataKey, "finalized_blockNum")
	num, err = g.Uint64()
	return
}

func (r *redisRepository) GetFinalizedBlockNumForPlugin(c context.Context) (num uint64, err error) {
	g := r.Redis.HGet(c, RedisMetadataKey, "plugins:finalized_blockNum")
	num, err = g.Uint64()
	return
}

func (r *redisRepository) setCache(c context.Context, key string, value interface{}, ttl int) (err error) {
	var val string
	switch v := value.(type) {
	case string:
		val = v
	case int64:
		val = strconv.FormatInt(v, 10)
	case int:
		val = strconv.Itoa(v)
	default:
		b, _ := json.Marshal(v)
		if val = string(b); val == "null" {
			return
		}
	}
	if ttl <= 0 {
		if err = r.Redis.Set(c, key, val, 0).Err(); err != nil {
			return
		}
	}
	err = r.Redis.SetEX(c, key, val, time.Duration(ttl)*time.Second).Err()
	return
}

func (r *redisRepository) getCacheBytes(c context.Context, key string) []byte {
	g := r.Redis.Get(c, key)
	if cache, err := g.Bytes(); err == nil {
		return cache
	}
	return nil
}

func (r *redisRepository) getCacheString(c context.Context, key string) string {
	g := r.Redis.Get(c, key)
	return g.String()
}

func (r *redisRepository) getCacheInt64(c context.Context, key string) int64 {
	g := r.Redis.Get(c, key)
	if cache, err := g.Int64(); err == nil {
		return cache
	}
	return 0
}

func (r *redisRepository) pingRedis(c context.Context) (err error) {

	if err := r.Redis.Set(c, "ping", "pong", 0).Err(); err != nil {
		log.Error("conn.Set(PING) error(", err, ")")
	}
	return
}

func (r *redisRepository) delCache(c context.Context, key ...string) error {
	if len(key) == 0 {
		return nil
	}
	err := r.Redis.Del(c, key...).Err()
	return err
}

func (r *redisRepository) AddMissingBlocks(c context.Context, num int) (err error) {
	if err = r.Redis.SAdd(c, RedisMissingBlocksSet, num).Err(); err != nil {
		return err
	}
	return nil
}

func (r *redisRepository) AddRepairedBlock(c context.Context, num int) (err error) {
	if err = r.Redis.SRem(c, RedisMissingBlocksSet, num).Err(); err != nil {
		return err
	}
	return nil
}

func (r *redisRepository) AddMissingBlocksInBulk(c context.Context, blockNum int, page, row int) error {
	if _, err := r.Redis.Pipelined(c, func(rdb redis.Pipeliner) error {
		for i := 0; i < (page+1)*row; i++ {
			rdb.SAdd(c, RedisMissingBlocksSet, blockNum-i)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (r *redisRepository) AddRepairedBlocksInBulk(c context.Context, blks []model.ChainBlock) (err error) {
	for _, block := range blks {
		if _, err := r.Redis.Pipelined(c, func(rdb redis.Pipeliner) error {
			if block.EventCount == 0 || block.ExtrinsicsCount == 0 {
				rdb.SRem(c, RedisMissingBlocksSet, block.BlockNum)
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	}
	return nil
}

func (r *redisRepository) GetMissingBlockSet(c context.Context) ([]string, error) {
	sm := r.Redis.SMembers(c, RedisMissingBlocksSet)
	if err := sm.Err(); err != nil {
		return nil, err
	}
	return sm.Val(), nil
}
