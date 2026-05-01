package mylog

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func init() {
	// 创建一个 Console 编码器，输出更易读的文本格式
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	// 添加以下配置来显示调用者信息
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder // 显示调用者信息
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder   // 时间格式
	encoder := zapcore.NewConsoleEncoder(encoderConfig)

	// 创建一个日志核心，输出到文件
	fileCore := zapcore.NewCore(
		encoder,
		zapcore.AddSync(&lumberjack.Logger{
			Filename:   "logs/upay.log",
			MaxSize:    30, // MB
			MaxBackups: 3,
			MaxAge:     7, // days
		}),
		zap.InfoLevel,
	)

	// 创建另一个日志核心，输出到标准输出
	consoleCore := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		zap.InfoLevel,
	)

	// 使用 zapcore.NewTee 将两个核心组合起来
	log_zap := zap.New(zapcore.NewTee(fileCore, consoleCore),
		zap.AddCaller(),      // 添加调用者信息
		zap.AddCallerSkip(0), // 调整调用栈跳过的帧数
	)

	// 将 logger 设置为全局变量
	Logger = log_zap
	// 确保 logger 在退出时进行同步

}

func MaskSensitive(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return "********"
	}
	return "********" + value[len(value)-4:]
}

func RedactURL(raw string) string {
	replacements := []string{"apikey=", "apiKey=", "key=", "token=", "signature=", "sign="}
	redacted := raw
	for _, marker := range replacements {
		lowerRedacted := strings.ToLower(redacted)
		lowerMarker := strings.ToLower(marker)
		idx := strings.Index(lowerRedacted, lowerMarker)
		for idx >= 0 {
			start := idx + len(marker)
			end := start
			for end < len(redacted) && redacted[end] != '&' {
				end++
			}
			redacted = redacted[:start] + "********" + redacted[end:]
			lowerRedacted = strings.ToLower(redacted)
			idx = strings.Index(lowerRedacted[start:], lowerMarker)
			if idx >= 0 {
				idx += start
			}
		}
	}
	return redacted
}

func RedactJSONBody(raw []byte, keys ...string) string {
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "[unparseable-json]"
	}

	sensitiveKeys := map[string]bool{}
	for _, key := range keys {
		sensitiveKeys[strings.ToLower(key)] = true
	}
	for key, value := range payload {
		if sensitiveKeys[strings.ToLower(key)] {
			if str, ok := value.(string); ok {
				payload[key] = MaskSensitive(str)
			} else {
				payload[key] = "********"
			}
		}
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return "[redaction-failed]"
	}
	return string(encoded)
}
