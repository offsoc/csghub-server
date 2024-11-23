// Code generated by mockery v2.49.1. DO NOT EDIT.

package accounting

import (
	mock "github.com/stretchr/testify/mock"
	types "opencsg.com/csghub-server/common/types"
)

// MockAccountingClient is an autogenerated mock type for the AccountingClient type
type MockAccountingClient struct {
	mock.Mock
}

type MockAccountingClient_Expecter struct {
	mock *mock.Mock
}

func (_m *MockAccountingClient) EXPECT() *MockAccountingClient_Expecter {
	return &MockAccountingClient_Expecter{mock: &_m.Mock}
}

// ListMeteringsByUserIDAndTime provides a mock function with given fields: req
func (_m *MockAccountingClient) ListMeteringsByUserIDAndTime(req types.ACCT_STATEMENTS_REQ) (any, error) {
	ret := _m.Called(req)

	if len(ret) == 0 {
		panic("no return value specified for ListMeteringsByUserIDAndTime")
	}

	var r0 any
	var r1 error
	if rf, ok := ret.Get(0).(func(types.ACCT_STATEMENTS_REQ) (any, error)); ok {
		return rf(req)
	}
	if rf, ok := ret.Get(0).(func(types.ACCT_STATEMENTS_REQ) any); ok {
		r0 = rf(req)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(any)
		}
	}

	if rf, ok := ret.Get(1).(func(types.ACCT_STATEMENTS_REQ) error); ok {
		r1 = rf(req)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockAccountingClient_ListMeteringsByUserIDAndTime_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ListMeteringsByUserIDAndTime'
type MockAccountingClient_ListMeteringsByUserIDAndTime_Call struct {
	*mock.Call
}

// ListMeteringsByUserIDAndTime is a helper method to define mock.On call
//   - req types.ACCT_STATEMENTS_REQ
func (_e *MockAccountingClient_Expecter) ListMeteringsByUserIDAndTime(req interface{}) *MockAccountingClient_ListMeteringsByUserIDAndTime_Call {
	return &MockAccountingClient_ListMeteringsByUserIDAndTime_Call{Call: _e.mock.On("ListMeteringsByUserIDAndTime", req)}
}

func (_c *MockAccountingClient_ListMeteringsByUserIDAndTime_Call) Run(run func(req types.ACCT_STATEMENTS_REQ)) *MockAccountingClient_ListMeteringsByUserIDAndTime_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(types.ACCT_STATEMENTS_REQ))
	})
	return _c
}

func (_c *MockAccountingClient_ListMeteringsByUserIDAndTime_Call) Return(_a0 any, _a1 error) *MockAccountingClient_ListMeteringsByUserIDAndTime_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockAccountingClient_ListMeteringsByUserIDAndTime_Call) RunAndReturn(run func(types.ACCT_STATEMENTS_REQ) (any, error)) *MockAccountingClient_ListMeteringsByUserIDAndTime_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockAccountingClient creates a new instance of MockAccountingClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockAccountingClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockAccountingClient {
	mock := &MockAccountingClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}