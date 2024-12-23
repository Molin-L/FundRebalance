package log

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var (
	logger, _ = zap.Config{
		Level:    zap.NewAtomicLevelAt(zapcore.DebugLevel),
		Encoding: "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "T",
			LevelKey:       "L",
			NameKey:        "N",
			CallerKey:      "C",
			MessageKey:     "M",
			StacktraceKey:  "S",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     microTimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths: []string{"stderr"},
	}.Build(zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zap.ErrorLevel))
	//logger, _              = zap.NewProduction(zap.AddCallerSkip(1))
	cfg                    = zap.NewProductionConfig()
	defaultPbJsonMarshaler = protojson.MarshalOptions{EmitUnpopulated: true}
	asynccore              *asyncCore
	asyncLogEnabled        bool
)

const (
	SpanKey = "span"
)

type Config struct {
	FileName   string
	MaxSize    int
	MaxAge     int
	MaxBackups int
	Compress   bool

	// 采样配置
	Tick       time.Duration
	First      int
	Thereafter int
	Level      zapcore.Level
}

type Span struct {
	TraceID      string `json:"trace_id"`
	SpanID       string `json:"span_id"`
	ParentSpanID string `json:"parent_span_id"`
}

func Start(logPath string) {
	cfg := &Config{
		FileName: logPath,
		MaxSize:  100,
	}
	StartWithConfig(cfg)
}

func newCore(config *Config) zapcore.Core {
	if config.MaxSize == 0 {
		config.MaxSize = 100
	}

	if config.FileName != "" {
		// lumberjack.Logger默认创建的文件夹权限是0744, 其他用户没有进入目录的权限
		// 这里预先创建文件夹，设置权限为0755
		_, err := os.Stat(config.FileName)
		if os.IsNotExist(err) {
			err := os.MkdirAll(filepath.Dir(config.FileName), 0755)
			if err != nil {
				fmt.Printf("can't make directories for new logfile: fileName:%s, err:%s", config.FileName, err)
			}
		}
	}

	l := &lumberjack.Logger{
		Filename:   config.FileName,
		MaxAge:     config.MaxAge,
		MaxBackups: config.MaxBackups,
		MaxSize:    config.MaxSize, // megabytes
		Compress:   config.Compress,
	}
	w := zapcore.AddSync(l)
	cfg.EncoderConfig = zapcore.EncoderConfig{
		TimeKey:        "T",
		LevelKey:       "L",
		NameKey:        "N",
		CallerKey:      "C",
		MessageKey:     "M",
		StacktraceKey:  "S",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     microTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	if asyncLogEnabled {
		asynccore = NewCore(
			zapcore.NewJSONEncoder(cfg.EncoderConfig),
			w,
			config.Level,
		)
		return asynccore
	} else {
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(cfg.EncoderConfig),
			w,
			config.Level,
		)
		return core
	}
}

func microTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02T15:04:05.000000Z"))
}

func StartWithConfig(config *Config) {
	core := newCore(config)
	if config.Tick > 0 {
		core = zapcore.NewSampler(core, config.Tick, config.First, config.Thereafter)
	}
	logger = zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zap.ErrorLevel),
	)
	zap.RedirectStdLog(logger)
}

// NewSpan 创建新的span信息
func NewSpan() *Span {
	return &Span{
		TraceID:      uuid.New().String(),
		SpanID:       uuid.New().String(),
		ParentSpanID: "",
	}
}

// WithSpan 检查ctx中是否有span信息，如没有则创建span信息
func WithSpan(ctx context.Context) context.Context {
	span := SpanFromContext(ctx)
	if span == nil {
		return context.WithValue(ctx, SpanKey, NewSpan())
	}
	return ctx
}

// SpanFromContext 从ctx中读取span信息
func SpanFromContext(ctx context.Context) *Span {
	if ctx == nil {
		return nil
	}
	span, ok := ctx.Value(SpanKey).(*Span)
	if ok {
		return span
	}
	return nil
}

func NewChildSpan(ctx context.Context) context.Context {
	span := SpanFromContext(ctx)
	if span == nil {
		return context.WithValue(ctx, SpanKey, NewSpan())
	}
	childSpan := NewSpan()
	childSpan.ParentSpanID = span.SpanID
	return context.WithValue(ctx, SpanKey, childSpan)
}

func Debug(ctx context.Context, msg string, fields ...zap.Field) {
	fields = append(fields, traceInfo(ctx)...)
	logger.Debug(msg, fields...)
}

func Info(ctx context.Context, msg string, fields ...zap.Field) {
	fields = append(fields, traceInfo(ctx)...)
	logger.Info(msg, fields...)
}

func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	fields = append(fields, traceInfo(ctx)...)
	logger.Warn(msg, fields...)
}

func Error(ctx context.Context, msg string, fields ...zap.Field) {
	fields = append(fields, traceInfo(ctx)...)
	logger.Error(msg, fields...)
}

func Panic(ctx context.Context, msg string, fields ...zap.Field) {
	fields = append(fields, traceInfo(ctx)...)
	logger.Panic(msg, fields...)
}

func WithField(f ...zapcore.Field) {
	logger = logger.With(f...)
}

func traceInfo(ctx context.Context) []zap.Field {
	span := SpanFromContext(ctx)
	if span == nil {
		return nil
	}
	var fields = []zap.Field{
		zap.String("TID", span.TraceID),
	}
	if span.SpanID != "" {
		fields = append(fields, zap.String("SID", span.SpanID))
	}
	if span.ParentSpanID != "" {
		fields = append(fields, zap.String("PSID", span.ParentSpanID))
	}
	return fields
}

// GetLocalIP 获取本机ip地址
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		return ""
	}

	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func Pb(key string, pbMsg proto.Message) zap.Field {
	jsonBytes, err := defaultPbJsonMarshaler.Marshal(pbMsg)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "log.Pb marshal to json err: %v, pbMsg: %v", err, pbMsg)
	}
	return zap.Any(key, json.RawMessage(jsonBytes))
}

func Close() {
	if asynccore != nil {
		asynccore.Close()
	}
}

/*
 Refactored log utils
*/

/*
 logger without context
*/

func Debugf(format string, args ...interface{}) {
	logger.Sugar().Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	logger.Sugar().Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	logger.Sugar().Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	logger.Sugar().Errorf(format, args...)
}

func Panicf(format string, args ...interface{}) {
	logger.Sugar().Panicf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	logger.Sugar().Fatalf(format, args...)
}

/*
 logger with context
*/

type (
	contextKey struct{}
)

func CtxAddKv(ctx context.Context, v ...interface{}) context.Context {
	return addLogger(ctx, getLogger(ctx).With(v...))
}

func CtxDebugf(ctx context.Context, format string, args ...interface{}) {
	formatArgs, fields := extractFields(ctx, args)
	getLogger(ctx).With(fields...).Debugf(format, formatArgs...)
}

func CtxInfof(ctx context.Context, format string, args ...interface{}) {
	formatArgs, fields := extractFields(ctx, args)
	getLogger(ctx).With(fields...).Infof(format, formatArgs...)
}

func CtxWarnf(ctx context.Context, format string, args ...interface{}) {
	formatArgs, fields := extractFields(ctx, args)
	getLogger(ctx).With(fields...).Warnf(format, formatArgs...)
}

func CtxErrorf(ctx context.Context, format string, args ...interface{}) {
	formatArgs, fields := extractFields(ctx, args)
	getLogger(ctx).With(fields...).Errorf(format, formatArgs...)
}

func CtxPanicf(ctx context.Context, format string, args ...interface{}) {
	formatArgs, fields := extractFields(ctx, args)
	getLogger(ctx).With(fields...).Panicf(format, formatArgs...)
}

func CtxFatalf(ctx context.Context, format string, args ...interface{}) {
	formatArgs, fields := extractFields(ctx, args)
	getLogger(ctx).With(fields...).Fatalf(format, formatArgs...)
}

func extractFields(ctx context.Context, args []interface{}) (formatArgs []interface{}, fields []interface{}) {
	for _, field := range traceInfo(ctx) {
		fields = append(fields, field)
	}

	for _, arg := range args {
		if _, ok := arg.(zap.Field); ok {
			fields = append(fields, arg)
			continue
		}
		formatArgs = append(formatArgs, arg)
	}
	return
}

func getLogger(ctx context.Context) *zap.SugaredLogger {
	if ctxLogger, ok := ctx.Value(contextKey{}).(*zap.SugaredLogger); ok {
		return ctxLogger
	}
	return logger.Sugar()
}

func addLogger(ctx context.Context, l *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}
