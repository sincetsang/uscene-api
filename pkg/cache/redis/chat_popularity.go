package redis

import (
	"TMA/pkg/common"
	"TMA/pkg/nucl"
	"TMA/pkg/sdredis"
	"context"
	"github.com/go-redis/redis/v8"
	"strconv"
	"time"
)

const maxMessages = 1000 // 最大消息数量

// ChatRoom 结构体包含 Redis 客户端和消息缓存
type ChatRoom struct {
	Ctx         context.Context
	redisClient *sdredis.Client
	key         string
}

// NewChatRoom 创建一个新的 ChatRoom 实例
func NewChatRoom(ctx context.Context, nu *nucl.Nucleus, chatRoomID int64) *ChatRoom {
	chatRoomIDStr := strconv.FormatInt(chatRoomID, 10)
	key := common.Key(common.PrefixChatPopularity, chatRoomIDStr)
	return &ChatRoom{
		Ctx:         ctx,
		redisClient: nu.RedisClient,
		key:         key,
	}
}

func (cr *ChatRoom) AddMessage() {
	currentTime := time.Now().UnixNano()

	cr.redisClient.ZAdd(cr.Ctx, cr.key, &redis.Z{
		Score:  float64(currentTime), // 使用时间戳作为分数
		Member: currentTime,          // 成员值也是时间戳
	})

	// 保留最新的 1000 条消息
	cr.redisClient.ZRemRangeByRank(cr.Ctx, cr.key, 0, -maxMessages-1)
}

func (cr *ChatRoom) GetMessageFrequency(interval time.Duration) int {
	currentTime := time.Now().UnixNano()
	windowStart := currentTime - int64(interval.Nanoseconds()) // 窗口起点为当前时间减去间隔时间
	count, _ := cr.redisClient.ZCount(cr.Ctx, cr.key, strconv.FormatInt(windowStart, 10), strconv.FormatInt(currentTime, 10)).Result()
	return int(count)
}
