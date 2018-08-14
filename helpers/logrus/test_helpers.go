package logrus

import (
	"bytes"

	"github.com/sirupsen/logrus"
)

func RunOnHijackedLogrusLevel(level logrus.Level, handler func()) {
	oldLevel := logrus.GetLevel()
	defer func() {
		logrus.SetLevel(oldLevel)
	}()

	logrus.SetLevel(level)

	handler()
}

func RunOnHijackedLogrusOutput(handler func(output *bytes.Buffer)) {
	oldOutput := logrus.StandardLogger().Out
	defer func() {
		logrus.StandardLogger().Out = oldOutput

	}()

	buf := bytes.NewBuffer([]byte{})
	logrus.StandardLogger().Out = buf

	handler(buf)
}
