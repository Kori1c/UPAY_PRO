package rdb

import (
	"context"
	"fmt"
	"sync"
	"time"
	"upay_pro/db/sdb"
	"upay_pro/mylog"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var RDB *redis.Client
var mu sync.Mutex

func init() {
	if err := Reload(); err != nil {
		mylog.Logger.Warn("redis 初始化失败，系统将以受限模式启动", zap.Error(err))
	}
}

func newClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		// 基本连接配置
		Addr:     fmt.Sprintf("%s:%d", sdb.GetSetting().Redishost, sdb.GetSetting().Redisport), // Redis 地址
		Password: sdb.GetSetting().Redispasswd,                                                 // Redis 密码
		DB:       sdb.GetSetting().Redisdb,                                                     // 数据库编号

		// 连接超时设置
		DialTimeout:  10 * time.Second, // 建立连接超时时间
		ReadTimeout:  30 * time.Second, // 读取超时时间
		WriteTimeout: 30 * time.Second, // 写入超时时间

		// 连接池设置
		PoolSize:        10,               // 连接池最大连接数
		MinIdleConns:    5,                // 最小空闲连接数
		PoolTimeout:     4 * time.Second,  // 从连接池获取连接的超时时间
		ConnMaxLifetime: 30 * time.Minute, // 连接的最大存活时间（替代 MaxConnAge）
		ConnMaxIdleTime: 5 * time.Minute,  // 空闲连接超时时间（替代 IdleTimeout）

		// 其他设置
		OnConnect: func(ctx context.Context, cn *redis.Conn) error {
			// 连接建立时的回调函数
			return nil
		},
	})
}

func Reload() error {
	mu.Lock()
	defer mu.Unlock()

	rdb := newClient()
	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		_ = rdb.Close()
		return err
	}

	old := RDB
	RDB = rdb
	if old != nil {
		_ = old.Close()
	}

	mylog.Logger.Info("redis 连接成功")
	return nil
}

// Close 优雅关闭 Redis 连接
