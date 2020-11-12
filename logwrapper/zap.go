package logwrapper

import (
	"fmt"

	"go.uber.org/zap"
)

// ZapWrapper wraps zap logger to implement ClientLogger
type ZapWrapper struct {
	logger *zap.Logger
}

// Info assumes first arg is a string checks for zap filds
// panics if args[0] is not a string and args[x] are not zap.Fields
func (w *ZapWrapper) Info(args ...interface{}) {
	msg, fields := getMsgAndFields(args...)
	w.logger.Info(msg, fields...)
}

// Fatal assumes first arg is a string checks for zap filds
// panics if args[0] is not a string and args[x] are not zap.Fields
func (w *ZapWrapper) Fatal(args ...interface{}) {
	msg, fields := getMsgAndFields(args)
	w.logger.Fatal(msg, fields...)
}

// NewZapWrapper returns new pointer to a ZapWrapper
func NewZapWrapper(logger *zap.Logger) (*ZapWrapper, error) {
	if logger == nil {
		return nil, fmt.Errorf("nil pointer sent as zap logger")
	}
	return &ZapWrapper{logger: logger}, nil
}

func getMsgAndFields(args ...interface{}) (string, []zap.Field) {
	if len(args) == 0 {
		panic("missing mandatory zap log msg")
	}

	msg, ok := args[0].(string)
	if !ok {
		panic(fmt.Sprintf("first argument not string %#v", args))
	}

	var fields []zap.Field
	for i := range args {
		if i == 0 {
			continue
		}

		field, ok := args[i].(zap.Field)
		if !ok {
			panic("arg not zap field")
		}
		fields = append(fields, field)
	}
	return msg, fields
}
