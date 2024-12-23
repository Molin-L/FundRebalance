package log

import (
	"bytes"
	"time"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

const (
	BatchSize = 100 * 1024
	Frequency = 1 * time.Second
)

var (
	_       zapcore.Core = (*asyncCore)(nil)
	stopped bool
)

// 复制zapcore 重写write方法 从同步写日志变成写chan
type asyncCore struct {
	zapcore.LevelEnabler
	enc   zapcore.Encoder
	out   zapcore.WriteSyncer
	logCh chan *buffer.Buffer
	done  chan struct{}
}

// NewCore creates a Core that writes logs to a WriteSyncer.
func NewCore(enc zapcore.Encoder, ws zapcore.WriteSyncer, enab zapcore.LevelEnabler) *asyncCore {
	core := &asyncCore{
		LevelEnabler: enab,
		enc:          enc,
		out:          ws,
		logCh:        make(chan *buffer.Buffer, 10240),
		done:         make(chan struct{}, 1),
	}

	go func() {
		var logBuffer bytes.Buffer
		ticker := time.NewTicker(Frequency)
		flushLog := false

		for {
			select {
			case buf := <-core.logCh:
				{
					if buf == nil {
						//sync log and return
						core.out.Write(logBuffer.Bytes())
						ticker.Stop()
						core.done <- struct{}{}
						return
					}

					logBuffer.Write(buf.Bytes())
					if logBuffer.Len() >= BatchSize {
						flushLog = true
					}

					buf.Free()
				}
			case <-ticker.C:
				flushLog = true
			}

			if flushLog {
				//日志写盘慢可能ticker超时,触发连续刷盘
				core.out.Write(logBuffer.Bytes())
				logBuffer.Reset()
				flushLog = false
			}
		}
	}()

	return core
}

func (c *asyncCore) With(fields []zapcore.Field) zapcore.Core {
	clone := c.clone()

	for i := range fields {
		fields[i].AddTo(clone.enc)
	}

	return clone
}

func (c *asyncCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *asyncCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	buf, err := c.enc.EncodeEntry(ent, fields)
	if err != nil {
		return err
	}

	if ent.Level > zapcore.ErrorLevel {
		c.logCh <- buf
		c.Close()
		return nil
	}

	if stopped {
		c.out.Write(buf.Bytes())
		buf.Free()
	} else {
		//fixme 先压测看数据，再决定是否需要select
		select {
		case c.logCh <- buf:
		default:
			c.out.Write([]byte("log chan blocked\n"))
			c.logCh <- buf
		}
	}

	return nil
}

func (c *asyncCore) Sync() error {
	return c.out.Sync()
}

func (c *asyncCore) clone() *asyncCore {
	return &asyncCore{
		LevelEnabler: c.LevelEnabler,
		enc:          c.enc.Clone(),
		out:          c.out,
	}
}

func (c *asyncCore) Close() {
	stopped = true
	c.logCh <- nil
	closeTimer := time.NewTimer(time.Second * 2)
	select {
	case <-closeTimer.C:
		c.out.Write([]byte("log flush timeout, log may lost.\n"))
	case <-c.done:
		c.out.Write([]byte("log flush successfully.\n"))
	}
}
