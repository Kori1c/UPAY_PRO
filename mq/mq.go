package mq

import (
	"context"
	"fmt"
	"sync"
	"time"
	"upay_pro/db/sdb"
	"upay_pro/mylog"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// 客户端
var Client *asynq.Client

// 服务端
var Mux *asynq.ServeMux

// 任务管理器
var Inspector *asynq.Inspector
var Server *asynq.Server
var mu sync.Mutex

func init() {
	if err := Reload(); err != nil {
		mylog.Logger.Warn("mq 初始化失败，系统将跳过异步队列能力", zap.Error(err))
	}
}

func redisOpt() asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Addr:     fmt.Sprintf("%s:%d", sdb.GetSetting().Redishost, sdb.GetSetting().Redisport),
		Password: sdb.GetSetting().Redispasswd,
		DB:       sdb.GetSetting().Redisdb,
	}
}

func Reload() error {
	mu.Lock()
	defer mu.Unlock()

	opt := redisOpt()
	client := asynq.NewClient(opt)
	inspector := asynq.NewInspector(opt)
	server := asynq.NewServer(opt, asynq.Config{Concurrency: 10})

	if err := server.Ping(); err != nil {
		_ = client.Close()
		_ = inspector.Close()
		return err
	}

	oldClient := Client
	oldInspector := Inspector
	oldServer := Server

	Client = client
	Inspector = inspector
	Server = server

	if oldServer != nil {
		oldServer.Shutdown()
	}
	if oldClient != nil {
		_ = oldClient.Close()
	}
	if oldInspector != nil {
		_ = oldInspector.Close()
	}

	go asyncServerRun(server)
	mylog.Logger.Info("mq redis 连接已重载")
	return nil
}

// QueueOrderExpiration 订单过期任务的队列名称
const QueueOrderExpiration = "order:expiration"

// TaskOrderExpiration 创建任务和任务加入对列
func TaskOrderExpiration(payload string, expirationDuration time.Duration) {
	if Client == nil {
		mylog.Logger.Warn("mq client 未初始化，跳过订单过期任务", zap.String("trade_id", payload))
		return
	}

	task := asynq.NewTask(QueueOrderExpiration, []byte(payload)) // 转换为字节切片
	// 将任务加入队列
	info, err := Client.Enqueue(task, asynq.ProcessIn(expirationDuration))
	if err != nil {
		mylog.Logger.Info("任务加入失败:" + err.Error())
	}
	mylog.Logger.Info("任务已加入队列:", zap.Any("info", info))

	// 把订单号和任务ID存在数据库中，方便使用
	var tradeIdTaskID sdb.TradeIdTaskID
	tradeIdTaskID.TradeId = payload
	tradeIdTaskID.TaskID = info.ID
	// 不存在就创建，存在就更新现有的记录
	sdb.DB.Create(&tradeIdTaskID)

}

// 队列服务端
func asyncServerRun(server *asynq.Server) {
	Mux = asynq.NewServeMux()
	// 注册处理函数，根据任务名称，调用不同的处理函数
	Mux.HandleFunc(QueueOrderExpiration, handleCheckStatusCodeTask)
	if err := server.Run(Mux); err != nil {
		mylog.Logger.Info("Error starting server:", zap.Any("err", err))
	}
}

// 处理过期任务
func handleCheckStatusCodeTask(ctx context.Context, t *asynq.Task) error {

	// 提取任务载荷传入的交易ID，根据ID去查一下订单记录里面的支付状态是否是待支付，如果是待支付，改为已过期
	// 订单过期后，需要解锁钱包地址和金额【从Redis里删除】
	payload := string(t.Payload())

	var order sdb.Orders

	err := sdb.DB.First(&order, "trade_id = ?", payload).Error
	if err != nil {
		mylog.Logger.Info("订单查询失败")
		return err
	}

	if order.Status == sdb.StatusWaitPay {
		order.Status = sdb.StatusExpired
		sdb.DB.Save(&order)
		mylog.Logger.Info(fmt.Sprintf("订单%v已设置为过期", order.TradeId))
	}

	// 根据订单号查到记录，删除记录
	var task sdb.TradeIdTaskID

	re := sdb.DB.Where("TradeId = ?", payload).Delete(&task)
	if re.Error != nil {
		mylog.Logger.Info("删除数据库TradeIdTaskID中的任务记录失败", zap.Error(re.Error))
		return re.Error
	}

	return nil
}

// 终止任务
func StopTask(taskID string) error {
	if Inspector == nil {
		return fmt.Errorf("mq inspector 未初始化")
	}
	// 从队列中删除任务
	err := Inspector.DeleteTask("default", taskID)
	if err != nil {
		mylog.Logger.Info("删除任务失败")
		return err
	}
	return nil
}
