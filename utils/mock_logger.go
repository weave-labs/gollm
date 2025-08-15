package utils

import "github.com/stretchr/testify/mock"

type MockLogger struct {
	mock.Mock
	ErrorCallCount   int
	LastErrorMessage string
}

func (m *MockLogger) Debug(msg string, keysAndValues ...any) {
	m.Called(msg, keysAndValues)
}

func (m *MockLogger) Info(msg string, keysAndValues ...any) {
	m.Called(msg, keysAndValues)
}

func (m *MockLogger) Warn(msg string, keysAndValues ...any) {
	m.Called(msg, keysAndValues)
}

func (m *MockLogger) Error(msg string, keysAndValues ...any) {
	m.ErrorCallCount++
	m.LastErrorMessage = msg
	m.Called(msg, keysAndValues)
}

func (m *MockLogger) SetLevel(level LogLevel) {
	m.Called(level)
}
