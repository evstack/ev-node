// Code generated by mockery; DO NOT EDIT.
// github.com/vektra/mockery
// template: testify

package mocks

import (
	"context"

	"github.com/evstack/ev-node/core/da"
	mock "github.com/stretchr/testify/mock"
)

// NewMockDA creates a new instance of MockDA. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockDA(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockDA {
	mock := &MockDA{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// MockDA is an autogenerated mock type for the DA type
type MockDA struct {
	mock.Mock
}

type MockDA_Expecter struct {
	mock *mock.Mock
}

func (_m *MockDA) EXPECT() *MockDA_Expecter {
	return &MockDA_Expecter{mock: &_m.Mock}
}

// Commit provides a mock function for the type MockDA
func (_mock *MockDA) Commit(ctx context.Context, blobs []da.Blob, namespace []byte) ([]da.Commitment, error) {
	ret := _mock.Called(ctx, blobs, namespace)

	if len(ret) == 0 {
		panic("no return value specified for Commit")
	}

	var r0 []da.Commitment
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, []da.Blob, []byte) ([]da.Commitment, error)); ok {
		return returnFunc(ctx, blobs, namespace)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, []da.Blob, []byte) []da.Commitment); ok {
		r0 = returnFunc(ctx, blobs, namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]da.Commitment)
		}
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, []da.Blob, []byte) error); ok {
		r1 = returnFunc(ctx, blobs, namespace)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockDA_Commit_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Commit'
type MockDA_Commit_Call struct {
	*mock.Call
}

// Commit is a helper method to define mock.On call
//   - ctx context.Context
//   - blobs []da.Blob
//   - namespace []byte
func (_e *MockDA_Expecter) Commit(ctx interface{}, blobs interface{}, namespace interface{}) *MockDA_Commit_Call {
	return &MockDA_Commit_Call{Call: _e.mock.On("Commit", ctx, blobs, namespace)}
}

func (_c *MockDA_Commit_Call) Run(run func(ctx context.Context, blobs []da.Blob, namespace []byte)) *MockDA_Commit_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 context.Context
		if args[0] != nil {
			arg0 = args[0].(context.Context)
		}
		var arg1 []da.Blob
		if args[1] != nil {
			arg1 = args[1].([]da.Blob)
		}
		var arg2 []byte
		if args[2] != nil {
			arg2 = args[2].([]byte)
		}
		run(
			arg0,
			arg1,
			arg2,
		)
	})
	return _c
}

func (_c *MockDA_Commit_Call) Return(vs []da.Commitment, err error) *MockDA_Commit_Call {
	_c.Call.Return(vs, err)
	return _c
}

func (_c *MockDA_Commit_Call) RunAndReturn(run func(ctx context.Context, blobs []da.Blob, namespace []byte) ([]da.Commitment, error)) *MockDA_Commit_Call {
	_c.Call.Return(run)
	return _c
}

// GasMultiplier provides a mock function for the type MockDA
func (_mock *MockDA) GasMultiplier(ctx context.Context) (float64, error) {
	ret := _mock.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for GasMultiplier")
	}

	var r0 float64
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context) (float64, error)); ok {
		return returnFunc(ctx)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context) float64); ok {
		r0 = returnFunc(ctx)
	} else {
		r0 = ret.Get(0).(float64)
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = returnFunc(ctx)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockDA_GasMultiplier_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GasMultiplier'
type MockDA_GasMultiplier_Call struct {
	*mock.Call
}

// GasMultiplier is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockDA_Expecter) GasMultiplier(ctx interface{}) *MockDA_GasMultiplier_Call {
	return &MockDA_GasMultiplier_Call{Call: _e.mock.On("GasMultiplier", ctx)}
}

func (_c *MockDA_GasMultiplier_Call) Run(run func(ctx context.Context)) *MockDA_GasMultiplier_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 context.Context
		if args[0] != nil {
			arg0 = args[0].(context.Context)
		}
		run(
			arg0,
		)
	})
	return _c
}

func (_c *MockDA_GasMultiplier_Call) Return(f float64, err error) *MockDA_GasMultiplier_Call {
	_c.Call.Return(f, err)
	return _c
}

func (_c *MockDA_GasMultiplier_Call) RunAndReturn(run func(ctx context.Context) (float64, error)) *MockDA_GasMultiplier_Call {
	_c.Call.Return(run)
	return _c
}

// GasPrice provides a mock function for the type MockDA
func (_mock *MockDA) GasPrice(ctx context.Context) (float64, error) {
	ret := _mock.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for GasPrice")
	}

	var r0 float64
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context) (float64, error)); ok {
		return returnFunc(ctx)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context) float64); ok {
		r0 = returnFunc(ctx)
	} else {
		r0 = ret.Get(0).(float64)
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = returnFunc(ctx)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockDA_GasPrice_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GasPrice'
type MockDA_GasPrice_Call struct {
	*mock.Call
}

// GasPrice is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockDA_Expecter) GasPrice(ctx interface{}) *MockDA_GasPrice_Call {
	return &MockDA_GasPrice_Call{Call: _e.mock.On("GasPrice", ctx)}
}

func (_c *MockDA_GasPrice_Call) Run(run func(ctx context.Context)) *MockDA_GasPrice_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 context.Context
		if args[0] != nil {
			arg0 = args[0].(context.Context)
		}
		run(
			arg0,
		)
	})
	return _c
}

func (_c *MockDA_GasPrice_Call) Return(f float64, err error) *MockDA_GasPrice_Call {
	_c.Call.Return(f, err)
	return _c
}

func (_c *MockDA_GasPrice_Call) RunAndReturn(run func(ctx context.Context) (float64, error)) *MockDA_GasPrice_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function for the type MockDA
func (_mock *MockDA) Get(ctx context.Context, ids []da.ID, namespace []byte) ([]da.Blob, error) {
	ret := _mock.Called(ctx, ids, namespace)

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 []da.Blob
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, []da.ID, []byte) ([]da.Blob, error)); ok {
		return returnFunc(ctx, ids, namespace)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, []da.ID, []byte) []da.Blob); ok {
		r0 = returnFunc(ctx, ids, namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]da.Blob)
		}
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, []da.ID, []byte) error); ok {
		r1 = returnFunc(ctx, ids, namespace)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockDA_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type MockDA_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - ctx context.Context
//   - ids []da.ID
//   - namespace []byte
func (_e *MockDA_Expecter) Get(ctx interface{}, ids interface{}, namespace interface{}) *MockDA_Get_Call {
	return &MockDA_Get_Call{Call: _e.mock.On("Get", ctx, ids, namespace)}
}

func (_c *MockDA_Get_Call) Run(run func(ctx context.Context, ids []da.ID, namespace []byte)) *MockDA_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 context.Context
		if args[0] != nil {
			arg0 = args[0].(context.Context)
		}
		var arg1 []da.ID
		if args[1] != nil {
			arg1 = args[1].([]da.ID)
		}
		var arg2 []byte
		if args[2] != nil {
			arg2 = args[2].([]byte)
		}
		run(
			arg0,
			arg1,
			arg2,
		)
	})
	return _c
}

func (_c *MockDA_Get_Call) Return(vs []da.Blob, err error) *MockDA_Get_Call {
	_c.Call.Return(vs, err)
	return _c
}

func (_c *MockDA_Get_Call) RunAndReturn(run func(ctx context.Context, ids []da.ID, namespace []byte) ([]da.Blob, error)) *MockDA_Get_Call {
	_c.Call.Return(run)
	return _c
}

// GetIDs provides a mock function for the type MockDA
func (_mock *MockDA) GetIDs(ctx context.Context, height uint64, namespace []byte) (*da.GetIDsResult, error) {
	ret := _mock.Called(ctx, height, namespace)

	if len(ret) == 0 {
		panic("no return value specified for GetIDs")
	}

	var r0 *da.GetIDsResult
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, uint64, []byte) (*da.GetIDsResult, error)); ok {
		return returnFunc(ctx, height, namespace)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, uint64, []byte) *da.GetIDsResult); ok {
		r0 = returnFunc(ctx, height, namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*da.GetIDsResult)
		}
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, uint64, []byte) error); ok {
		r1 = returnFunc(ctx, height, namespace)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockDA_GetIDs_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetIDs'
type MockDA_GetIDs_Call struct {
	*mock.Call
}

// GetIDs is a helper method to define mock.On call
//   - ctx context.Context
//   - height uint64
//   - namespace []byte
func (_e *MockDA_Expecter) GetIDs(ctx interface{}, height interface{}, namespace interface{}) *MockDA_GetIDs_Call {
	return &MockDA_GetIDs_Call{Call: _e.mock.On("GetIDs", ctx, height, namespace)}
}

func (_c *MockDA_GetIDs_Call) Run(run func(ctx context.Context, height uint64, namespace []byte)) *MockDA_GetIDs_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 context.Context
		if args[0] != nil {
			arg0 = args[0].(context.Context)
		}
		var arg1 uint64
		if args[1] != nil {
			arg1 = args[1].(uint64)
		}
		var arg2 []byte
		if args[2] != nil {
			arg2 = args[2].([]byte)
		}
		run(
			arg0,
			arg1,
			arg2,
		)
	})
	return _c
}

func (_c *MockDA_GetIDs_Call) Return(getIDsResult *da.GetIDsResult, err error) *MockDA_GetIDs_Call {
	_c.Call.Return(getIDsResult, err)
	return _c
}

func (_c *MockDA_GetIDs_Call) RunAndReturn(run func(ctx context.Context, height uint64, namespace []byte) (*da.GetIDsResult, error)) *MockDA_GetIDs_Call {
	_c.Call.Return(run)
	return _c
}

// GetProofs provides a mock function for the type MockDA
func (_mock *MockDA) GetProofs(ctx context.Context, ids []da.ID, namespace []byte) ([]da.Proof, error) {
	ret := _mock.Called(ctx, ids, namespace)

	if len(ret) == 0 {
		panic("no return value specified for GetProofs")
	}

	var r0 []da.Proof
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, []da.ID, []byte) ([]da.Proof, error)); ok {
		return returnFunc(ctx, ids, namespace)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, []da.ID, []byte) []da.Proof); ok {
		r0 = returnFunc(ctx, ids, namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]da.Proof)
		}
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, []da.ID, []byte) error); ok {
		r1 = returnFunc(ctx, ids, namespace)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockDA_GetProofs_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetProofs'
type MockDA_GetProofs_Call struct {
	*mock.Call
}

// GetProofs is a helper method to define mock.On call
//   - ctx context.Context
//   - ids []da.ID
//   - namespace []byte
func (_e *MockDA_Expecter) GetProofs(ctx interface{}, ids interface{}, namespace interface{}) *MockDA_GetProofs_Call {
	return &MockDA_GetProofs_Call{Call: _e.mock.On("GetProofs", ctx, ids, namespace)}
}

func (_c *MockDA_GetProofs_Call) Run(run func(ctx context.Context, ids []da.ID, namespace []byte)) *MockDA_GetProofs_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 context.Context
		if args[0] != nil {
			arg0 = args[0].(context.Context)
		}
		var arg1 []da.ID
		if args[1] != nil {
			arg1 = args[1].([]da.ID)
		}
		var arg2 []byte
		if args[2] != nil {
			arg2 = args[2].([]byte)
		}
		run(
			arg0,
			arg1,
			arg2,
		)
	})
	return _c
}

func (_c *MockDA_GetProofs_Call) Return(vs []da.Proof, err error) *MockDA_GetProofs_Call {
	_c.Call.Return(vs, err)
	return _c
}

func (_c *MockDA_GetProofs_Call) RunAndReturn(run func(ctx context.Context, ids []da.ID, namespace []byte) ([]da.Proof, error)) *MockDA_GetProofs_Call {
	_c.Call.Return(run)
	return _c
}

// Submit provides a mock function for the type MockDA
func (_mock *MockDA) Submit(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte) ([]da.ID, error) {
	ret := _mock.Called(ctx, blobs, gasPrice, namespace)

	if len(ret) == 0 {
		panic("no return value specified for Submit")
	}

	var r0 []da.ID
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, []da.Blob, float64, []byte) ([]da.ID, error)); ok {
		return returnFunc(ctx, blobs, gasPrice, namespace)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, []da.Blob, float64, []byte) []da.ID); ok {
		r0 = returnFunc(ctx, blobs, gasPrice, namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]da.ID)
		}
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, []da.Blob, float64, []byte) error); ok {
		r1 = returnFunc(ctx, blobs, gasPrice, namespace)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockDA_Submit_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Submit'
type MockDA_Submit_Call struct {
	*mock.Call
}

// Submit is a helper method to define mock.On call
//   - ctx context.Context
//   - blobs []da.Blob
//   - gasPrice float64
//   - namespace []byte
func (_e *MockDA_Expecter) Submit(ctx interface{}, blobs interface{}, gasPrice interface{}, namespace interface{}) *MockDA_Submit_Call {
	return &MockDA_Submit_Call{Call: _e.mock.On("Submit", ctx, blobs, gasPrice, namespace)}
}

func (_c *MockDA_Submit_Call) Run(run func(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte)) *MockDA_Submit_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 context.Context
		if args[0] != nil {
			arg0 = args[0].(context.Context)
		}
		var arg1 []da.Blob
		if args[1] != nil {
			arg1 = args[1].([]da.Blob)
		}
		var arg2 float64
		if args[2] != nil {
			arg2 = args[2].(float64)
		}
		var arg3 []byte
		if args[3] != nil {
			arg3 = args[3].([]byte)
		}
		run(
			arg0,
			arg1,
			arg2,
			arg3,
		)
	})
	return _c
}

func (_c *MockDA_Submit_Call) Return(vs []da.ID, err error) *MockDA_Submit_Call {
	_c.Call.Return(vs, err)
	return _c
}

func (_c *MockDA_Submit_Call) RunAndReturn(run func(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte) ([]da.ID, error)) *MockDA_Submit_Call {
	_c.Call.Return(run)
	return _c
}

// SubmitWithOptions provides a mock function for the type MockDA
func (_mock *MockDA) SubmitWithOptions(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte, options []byte) ([]da.ID, error) {
	ret := _mock.Called(ctx, blobs, gasPrice, namespace, options)

	if len(ret) == 0 {
		panic("no return value specified for SubmitWithOptions")
	}

	var r0 []da.ID
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, []da.Blob, float64, []byte, []byte) ([]da.ID, error)); ok {
		return returnFunc(ctx, blobs, gasPrice, namespace, options)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, []da.Blob, float64, []byte, []byte) []da.ID); ok {
		r0 = returnFunc(ctx, blobs, gasPrice, namespace, options)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]da.ID)
		}
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, []da.Blob, float64, []byte, []byte) error); ok {
		r1 = returnFunc(ctx, blobs, gasPrice, namespace, options)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockDA_SubmitWithOptions_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SubmitWithOptions'
type MockDA_SubmitWithOptions_Call struct {
	*mock.Call
}

// SubmitWithOptions is a helper method to define mock.On call
//   - ctx context.Context
//   - blobs []da.Blob
//   - gasPrice float64
//   - namespace []byte
//   - options []byte
func (_e *MockDA_Expecter) SubmitWithOptions(ctx interface{}, blobs interface{}, gasPrice interface{}, namespace interface{}, options interface{}) *MockDA_SubmitWithOptions_Call {
	return &MockDA_SubmitWithOptions_Call{Call: _e.mock.On("SubmitWithOptions", ctx, blobs, gasPrice, namespace, options)}
}

func (_c *MockDA_SubmitWithOptions_Call) Run(run func(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte, options []byte)) *MockDA_SubmitWithOptions_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 context.Context
		if args[0] != nil {
			arg0 = args[0].(context.Context)
		}
		var arg1 []da.Blob
		if args[1] != nil {
			arg1 = args[1].([]da.Blob)
		}
		var arg2 float64
		if args[2] != nil {
			arg2 = args[2].(float64)
		}
		var arg3 []byte
		if args[3] != nil {
			arg3 = args[3].([]byte)
		}
		var arg4 []byte
		if args[4] != nil {
			arg4 = args[4].([]byte)
		}
		run(
			arg0,
			arg1,
			arg2,
			arg3,
			arg4,
		)
	})
	return _c
}

func (_c *MockDA_SubmitWithOptions_Call) Return(vs []da.ID, err error) *MockDA_SubmitWithOptions_Call {
	_c.Call.Return(vs, err)
	return _c
}

func (_c *MockDA_SubmitWithOptions_Call) RunAndReturn(run func(ctx context.Context, blobs []da.Blob, gasPrice float64, namespace []byte, options []byte) ([]da.ID, error)) *MockDA_SubmitWithOptions_Call {
	_c.Call.Return(run)
	return _c
}

// Validate provides a mock function for the type MockDA
func (_mock *MockDA) Validate(ctx context.Context, ids []da.ID, proofs []da.Proof, namespace []byte) ([]bool, error) {
	ret := _mock.Called(ctx, ids, proofs, namespace)

	if len(ret) == 0 {
		panic("no return value specified for Validate")
	}

	var r0 []bool
	var r1 error
	if returnFunc, ok := ret.Get(0).(func(context.Context, []da.ID, []da.Proof, []byte) ([]bool, error)); ok {
		return returnFunc(ctx, ids, proofs, namespace)
	}
	if returnFunc, ok := ret.Get(0).(func(context.Context, []da.ID, []da.Proof, []byte) []bool); ok {
		r0 = returnFunc(ctx, ids, proofs, namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]bool)
		}
	}
	if returnFunc, ok := ret.Get(1).(func(context.Context, []da.ID, []da.Proof, []byte) error); ok {
		r1 = returnFunc(ctx, ids, proofs, namespace)
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// MockDA_Validate_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Validate'
type MockDA_Validate_Call struct {
	*mock.Call
}

// Validate is a helper method to define mock.On call
//   - ctx context.Context
//   - ids []da.ID
//   - proofs []da.Proof
//   - namespace []byte
func (_e *MockDA_Expecter) Validate(ctx interface{}, ids interface{}, proofs interface{}, namespace interface{}) *MockDA_Validate_Call {
	return &MockDA_Validate_Call{Call: _e.mock.On("Validate", ctx, ids, proofs, namespace)}
}

func (_c *MockDA_Validate_Call) Run(run func(ctx context.Context, ids []da.ID, proofs []da.Proof, namespace []byte)) *MockDA_Validate_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 context.Context
		if args[0] != nil {
			arg0 = args[0].(context.Context)
		}
		var arg1 []da.ID
		if args[1] != nil {
			arg1 = args[1].([]da.ID)
		}
		var arg2 []da.Proof
		if args[2] != nil {
			arg2 = args[2].([]da.Proof)
		}
		var arg3 []byte
		if args[3] != nil {
			arg3 = args[3].([]byte)
		}
		run(
			arg0,
			arg1,
			arg2,
			arg3,
		)
	})
	return _c
}

func (_c *MockDA_Validate_Call) Return(bools []bool, err error) *MockDA_Validate_Call {
	_c.Call.Return(bools, err)
	return _c
}

func (_c *MockDA_Validate_Call) RunAndReturn(run func(ctx context.Context, ids []da.ID, proofs []da.Proof, namespace []byte) ([]bool, error)) *MockDA_Validate_Call {
	_c.Call.Return(run)
	return _c
}
