// Code generated by mockery v2.48.0. DO NOT EDIT.

package database

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	database "opencsg.com/csghub-server/builder/store/database"
)

// MockPromptPrefixStore is an autogenerated mock type for the PromptPrefixStore type
type MockPromptPrefixStore struct {
	mock.Mock
}

type MockPromptPrefixStore_Expecter struct {
	mock *mock.Mock
}

func (_m *MockPromptPrefixStore) EXPECT() *MockPromptPrefixStore_Expecter {
	return &MockPromptPrefixStore_Expecter{mock: &_m.Mock}
}

// Get provides a mock function with given fields: ctx
func (_m *MockPromptPrefixStore) Get(ctx context.Context) (*database.PromptPrefix, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 *database.PromptPrefix
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (*database.PromptPrefix, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) *database.PromptPrefix); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*database.PromptPrefix)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockPromptPrefixStore_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type MockPromptPrefixStore_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockPromptPrefixStore_Expecter) Get(ctx interface{}) *MockPromptPrefixStore_Get_Call {
	return &MockPromptPrefixStore_Get_Call{Call: _e.mock.On("Get", ctx)}
}

func (_c *MockPromptPrefixStore_Get_Call) Run(run func(ctx context.Context)) *MockPromptPrefixStore_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockPromptPrefixStore_Get_Call) Return(_a0 *database.PromptPrefix, _a1 error) *MockPromptPrefixStore_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockPromptPrefixStore_Get_Call) RunAndReturn(run func(context.Context) (*database.PromptPrefix, error)) *MockPromptPrefixStore_Get_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockPromptPrefixStore creates a new instance of MockPromptPrefixStore. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockPromptPrefixStore(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockPromptPrefixStore {
	mock := &MockPromptPrefixStore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}