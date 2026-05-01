package web

import (
	"context"
	"errors"
	"net/http"
	"time"
	"upay_pro/db/rdb"
	"upay_pro/db/sdb"

	"github.com/gin-gonic/gin"
)

var errRedisUnavailable = errors.New("redis unavailable")

func registerHealthRoutes(r *gin.Engine) {
	r.GET("/healthz", healthzHandler)
}

func healthzHandler(c *gin.Context) {
	status := http.StatusOK
	dbStatus := "ok"
	redisStatus := "ok"

	if err := checkDatabase(); err != nil {
		status = http.StatusServiceUnavailable
		dbStatus = "error"
	}

	if err := checkRedis(); err != nil {
		status = http.StatusServiceUnavailable
		redisStatus = "error"
	}

	overall := "ok"
	if status != http.StatusOK {
		overall = "degraded"
	}

	c.JSON(status, gin.H{
		"code": 0,
		"data": gin.H{
			"status": overall,
			"db":     dbStatus,
			"redis":  redisStatus,
		},
	})
}

func checkDatabase() error {
	sqlDB, err := sdb.DB.DB()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return sqlDB.PingContext(ctx)
}

func checkRedis() error {
	if rdb.RDB == nil {
		return errRedisUnavailable
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return rdb.RDB.Ping(ctx).Err()
}
