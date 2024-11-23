// Code generated by mockery v2.49.1. DO NOT EDIT.

package database

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	database "opencsg.com/csghub-server/builder/store/database"
)

// MockTelemetryStore is an autogenerated mock type for the TelemetryStore type
type MockTelemetryStore struct {
	mock.Mock
}

type MockTelemetryStore_Expecter struct {
	mock *mock.Mock
}

func (_m *MockTelemetryStore) EXPECT() *MockTelemetryStore_Expecter {
	return &MockTelemetryStore_Expecter{mock: &_m.Mock}
}

// Save provides a mock function with given fields: ctx, telemetry
func (_m *MockTelemetryStore) Save(ctx context.Context, telemetry *database.Telemetry) error {
	ret := _m.Called(ctx, telemetry)

	if len(ret) == 0 {
		panic("no return value specified for Save")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *database.Telemetry) error); ok {
		r0 = rf(ctx, telemetry)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockTelemetryStore_Save_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Save'
type MockTelemetryStore_Save_Call struct {
	*mock.Call
}

// Save is a helper method to define mock.On call
//   - ctx context.Context
//   - telemetry *database.Telemetry
func (_e *MockTelemetryStore_Expecter) Save(ctx interface{}, telemetry interface{}) *MockTelemetryStore_Save_Call {
	return &MockTelemetryStore_Save_Call{Call: _e.mock.On("Save", ctx, telemetry)}
}

func (_c *MockTelemetryStore_Save_Call) Run(run func(ctx context.Context, telemetry *database.Telemetry)) *MockTelemetryStore_Save_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*database.Telemetry))
	})
	return _c
}

func (_c *MockTelemetryStore_Save_Call) Return(_a0 error) *MockTelemetryStore_Save_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockTelemetryStore_Save_Call) RunAndReturn(run func(context.Context, *database.Telemetry) error) *MockTelemetryStore_Save_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockTelemetryStore creates a new instance of MockTelemetryStore. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockTelemetryStore(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockTelemetryStore {
	mock := &MockTelemetryStore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}