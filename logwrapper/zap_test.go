package logwrapper

import (
	"testing"

	"github.com/blendle/zapdriver"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapSuite struct{}

func TestZapSuite(t *testing.T) {
	s := &zapSuite{}

	t.Run("testNonStructured", s.testNonStructured)
	t.Run("testStructured", s.testStructured)
}

func (s *zapSuite) testNonStructured(t *testing.T) {
	logger, err := newLogger(false, "")
	require.NoError(t, err)

	wlogger, err := NewZapWrapper(logger)
	require.NoError(t, err)
	require.NotNil(t, wlogger)
}

func (s *zapSuite) testStructured(t *testing.T) {
	logger, err := newLogger(true, "zappy")
	require.NoError(t, err)

	wlogger, err := NewZapWrapper(logger)
	require.NoError(t, err)
	require.NotNil(t, wlogger)

	wlogger.Info("testing")
	wlogger.Info("testing", zap.String("key", "val"))

	t.Run("testArgument", func(t *testing.T) {
		defer func() {
			r := recover()
			require.NotNil(t, r)
		}()
		wlogger.Info(1)
	})
}

func newLogger(structuredLogs bool, servicename string) (*zap.Logger, error) {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	logger, err := config.Build()
	if err != nil {
		return logger, err
	}

	if structuredLogs {
		logger, err = zapdriver.NewProductionWithCore(zapdriver.WrapCore(
			zapdriver.ReportAllErrors(true),
			zapdriver.ServiceName(servicename),
		))
		if err != nil {
			return logger, err
		}
	}
	return logger, nil
}
