/*
Copyright 2021 The Dapr Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package logger

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fakeLoggerName = "fakeLogger"

func getTestLogger(buf io.Writer) *daprLogger {
	l := newDaprLogger(fakeLoggerName)
	l.SetOutput(buf)
	l.logger.Logger.ExitFunc = func(i int) {} // don't quit the test

	return l
}

func TestEnableJSON(t *testing.T) {
	var buf bytes.Buffer
	testLogger := getTestLogger(&buf)

	expectedHost, _ := os.Hostname()
	testLogger.EnableJSONOutput(true)
	_, okJSON := testLogger.logger.Logger.Formatter.(*logrus.JSONFormatter)
	assert.True(t, okJSON)
	assert.Equal(t, "fakeLogger", testLogger.logger.Data[logFieldScope])
	assert.Equal(t, LogTypeLog, testLogger.logger.Data[logFieldType])
	assert.Equal(t, expectedHost, testLogger.logger.Data[logFieldInstance])

	testLogger.EnableJSONOutput(false)
	_, okText := testLogger.logger.Logger.Formatter.(*logrus.TextFormatter)
	assert.True(t, okText)
	assert.Equal(t, "fakeLogger", testLogger.logger.Data[logFieldScope])
	assert.Equal(t, LogTypeLog, testLogger.logger.Data[logFieldType])
	assert.Equal(t, expectedHost, testLogger.logger.Data[logFieldInstance])
}

func TestJSONLoggerFields(t *testing.T) {
	tests := []struct {
		name        string
		outputLevel LogLevel
		level       string
		appID       string
		message     string
		instance    string
		fn          func(*daprLogger, string)
	}{
		{
			"info()",
			InfoLevel,
			"info",
			"dapr_app",
			"King Dapr",
			"dapr-pod",
			func(l *daprLogger, msg string) {
				l.Info(msg)
			},
		},
		{
			"infof()",
			InfoLevel,
			"info",
			"dapr_app",
			"King Dapr",
			"dapr-pod",
			func(l *daprLogger, msg string) {
				l.Infof("%s", msg)
			},
		},
		{
			"debug()",
			DebugLevel,
			"debug",
			"dapr_app",
			"King Dapr",
			"dapr-pod",
			func(l *daprLogger, msg string) {
				l.Debug(msg)
			},
		},
		{
			"debugf()",
			DebugLevel,
			"debug",
			"dapr_app",
			"King Dapr",
			"dapr-pod",
			func(l *daprLogger, msg string) {
				l.Debugf("%s", msg)
			},
		},
		{
			"error()",
			InfoLevel,
			"error",
			"dapr_app",
			"King Dapr",
			"dapr-pod",
			func(l *daprLogger, msg string) {
				l.Error(msg)
			},
		},
		{
			"errorf()",
			InfoLevel,
			"error",
			"dapr_app",
			"King Dapr",
			"dapr-pod",
			func(l *daprLogger, msg string) {
				l.Errorf("%s", msg)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			testLogger := getTestLogger(&buf)
			testLogger.EnableJSONOutput(true)
			testLogger.SetAppID(tt.appID)
			DaprVersion = tt.appID
			testLogger.SetOutputLevel(tt.outputLevel)
			testLogger.logger.Data[logFieldInstance] = tt.instance

			tt.fn(testLogger, tt.message)

			b, _ := buf.ReadBytes('\n')
			var o map[string]interface{}
			require.NoError(t, json.Unmarshal(b, &o))

			// assert
			assert.Equal(t, tt.appID, o[logFieldAppID])
			assert.Equal(t, tt.instance, o[logFieldInstance])
			assert.Equal(t, tt.level, o[logFieldLevel])
			assert.Equal(t, LogTypeLog, o[logFieldType])
			assert.Equal(t, fakeLoggerName, o[logFieldScope])
			assert.Equal(t, tt.message, o[logFieldMessage])
			_, err := time.Parse(time.RFC3339, o[logFieldTimeStamp].(string))
			require.NoError(t, err)
		})
	}
}

func TestOutputLevel(t *testing.T) {
	tests := []struct {
		outputLevel          LogLevel
		expectedOutputLevels map[LogLevel]bool
	}{
		{
			outputLevel: DebugLevel,
			expectedOutputLevels: map[LogLevel]bool{
				DebugLevel: true,
				InfoLevel:  true,
				WarnLevel:  true,
				ErrorLevel: true,
				FatalLevel: true,
			},
		},
		{
			outputLevel: InfoLevel,
			expectedOutputLevels: map[LogLevel]bool{
				DebugLevel: false,
				InfoLevel:  true,
				WarnLevel:  true,
				ErrorLevel: true,
				FatalLevel: true,
			},
		},
		{
			outputLevel: WarnLevel,
			expectedOutputLevels: map[LogLevel]bool{
				DebugLevel: false,
				InfoLevel:  false,
				WarnLevel:  true,
				ErrorLevel: true,
				FatalLevel: true,
			},
		},
		{
			outputLevel: ErrorLevel,
			expectedOutputLevels: map[LogLevel]bool{
				DebugLevel: false,
				InfoLevel:  false,
				WarnLevel:  false,
				ErrorLevel: true,
				FatalLevel: true,
			},
		},
		{
			outputLevel: FatalLevel,
			expectedOutputLevels: map[LogLevel]bool{
				DebugLevel: false,
				InfoLevel:  false,
				WarnLevel:  false,
				ErrorLevel: false,
				FatalLevel: true,
			},
		},
		{
			outputLevel: UndefinedLevel,
			expectedOutputLevels: map[LogLevel]bool{
				DebugLevel: false,
				InfoLevel:  false,
				WarnLevel:  false,
				ErrorLevel: false,
				FatalLevel: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.outputLevel), func(t *testing.T) {
			for l, want := range tt.expectedOutputLevels {
				var buf bytes.Buffer
				testLogger := getTestLogger(&buf)
				testLogger.SetOutputLevel(tt.outputLevel)

				assert.Equal(t, want, testLogger.IsOutputLevelEnabled(l))

				switch l {
				case DebugLevel:
					testLogger.Debug("")
				case InfoLevel:
					testLogger.Info("")
				case WarnLevel:
					testLogger.Warn("")
				case ErrorLevel:
					testLogger.Error("")
				case FatalLevel:
					testLogger.Fatal("")
				}

				if want {
					assert.NotEmptyf(t, buf.Bytes(), "expected to log %v", l)
				} else {
					assert.Emptyf(t, buf.Bytes(), "expected to not log %v", l)
				}
			}
		})
	}
}

func TestWithTypeFields(t *testing.T) {
	var buf bytes.Buffer
	testLogger := getTestLogger(&buf)
	testLogger.EnableJSONOutput(true)
	testLogger.SetAppID("dapr_app")
	testLogger.SetOutputLevel(InfoLevel)

	// WithLogType will return new Logger with request log type
	// Meanwhile, testLogger uses the default logtype
	loggerWithRequestType := testLogger.WithLogType(LogTypeRequest)
	loggerWithRequestType.Info("call user app")

	b, _ := buf.ReadBytes('\n')
	var o map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &o))

	assert.Equalf(t, LogTypeRequest, o[logFieldType], "new logger must be %s type", LogTypeRequest)

	// Log our via testLogger to ensure that testLogger still uses the default logtype
	testLogger.Info("testLogger with log LogType")

	b, _ = buf.ReadBytes('\n')
	clear(o)
	require.NoError(t, json.Unmarshal(b, &o))

	assert.Equalf(t, LogTypeLog, o[logFieldType], "testLogger must be %s type", LogTypeLog)
}

func TestWithFields(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		var buf bytes.Buffer
		testLogger := getTestLogger(&buf)
		testLogger.EnableJSONOutput(true)
		testLogger.SetAppID("dapr_app")
		testLogger.SetOutputLevel(InfoLevel)

		var o map[string]interface{}

		// Test adding fields
		testLogger.WithFields(map[string]any{
			"answer": 42,
			"hello":  "world",
		}).Info("🙃")

		b, _ := buf.ReadBytes('\n')
		clear(o)
		require.NoError(t, json.Unmarshal(b, &o))

		assert.Equal(t, "🙃", o["msg"])
		assert.Equal(t, "world", o["hello"])
		assert.InDelta(t, float64(42), o["answer"], 000.1)

		// Test with other fields
		testLogger.WithFields(map[string]any{
			"🤌": []string{"👍", "🚀"},
		}).Info("🐶")

		b, _ = buf.ReadBytes('\n')
		clear(o)
		require.NoError(t, json.Unmarshal(b, &o))

		assert.Equal(t, "🐶", o["msg"])
		assert.Len(t, o["🤌"], 2)
		assert.Equal(t, "👍", (o["🤌"].([]any))[0])
		assert.Equal(t, "🚀", (o["🤌"].([]any))[1])
		assert.Empty(t, o["hello"])
		assert.Empty(t, o["answer"])

		// Log our via testLogger to ensure that testLogger still uses the default fields
		testLogger.Info("🤔")

		b, _ = buf.ReadBytes('\n')
		clear(o)
		require.NoError(t, json.Unmarshal(b, &o))

		assert.Equal(t, "🤔", o["msg"])
		assert.Empty(t, o["hello"])
		assert.Empty(t, o["answer"])
	})

	t.Run("text", func(t *testing.T) {
		var buf bytes.Buffer
		testLogger := getTestLogger(&buf)
		testLogger.EnableJSONOutput(false)
		testLogger.SetAppID("dapr_app")
		testLogger.SetOutputLevel(InfoLevel)

		// Test adding fields
		testLogger.WithFields(map[string]any{
			"answer": 42,
			"hello":  "world",
		}).Info("🙃")

		b, _ := buf.ReadBytes('\n')

		assert.True(t, regexp.MustCompile(`(^| )msg="🙃"($| )`).Match(b))
		assert.True(t, regexp.MustCompile(`(^| )answer=42($| )`).Match(b))
		assert.True(t, regexp.MustCompile(`(^| )hello=world($| )`).Match(b))

		// Test with other fields
		testLogger.WithFields(map[string]any{
			"🤌": []string{"👍", "🚀"},
		}).Info("🐶")

		b, _ = buf.ReadBytes('\n')

		assert.True(t, regexp.MustCompile(`(^| )msg="🐶"($| )`).Match(b))
		assert.True(t, regexp.MustCompile(`(^| )🤌=`).Match(b))
		assert.False(t, regexp.MustCompile(`(^| )answer=`).Match(b))
		assert.False(t, regexp.MustCompile(`(^| )hello=`).Match(b))

		// Log our via testLogger to ensure that testLogger still uses the default fields
		testLogger.Info("🤔")

		b, _ = buf.ReadBytes('\n')

		assert.True(t, regexp.MustCompile(`(^| )msg="🤔"($| )`).Match(b))
		assert.False(t, regexp.MustCompile(`(^| )answer=`).Match(b))
		assert.False(t, regexp.MustCompile(`(^| )hello=`).Match(b))
	})
}

func TestToLogrusLevel(t *testing.T) {
	t.Run("Dapr DebugLevel to Logrus.DebugLevel", func(t *testing.T) {
		assert.Equal(t, logrus.DebugLevel, toLogrusLevel(DebugLevel))
	})

	t.Run("Dapr InfoLevel to Logrus.InfoLevel", func(t *testing.T) {
		assert.Equal(t, logrus.InfoLevel, toLogrusLevel(InfoLevel))
	})

	t.Run("Dapr WarnLevel to Logrus.WarnLevel", func(t *testing.T) {
		assert.Equal(t, logrus.WarnLevel, toLogrusLevel(WarnLevel))
	})

	t.Run("Dapr ErrorLevel to Logrus.ErrorLevel", func(t *testing.T) {
		assert.Equal(t, logrus.ErrorLevel, toLogrusLevel(ErrorLevel))
	})

	t.Run("Dapr FatalLevel to Logrus.FatalLevel", func(t *testing.T) {
		assert.Equal(t, logrus.FatalLevel, toLogrusLevel(FatalLevel))
	})
}
