// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package callback

import (
	"context"
	"github.com/stretchr/testify/mock"
	"opencsg.com/csghub-server/_mocks/opencsg.com/csghub-server/builder/git/gitserver"
	"opencsg.com/csghub-server/_mocks/opencsg.com/csghub-server/builder/rpc"
	component2 "opencsg.com/csghub-server/_mocks/opencsg.com/csghub-server/component"
	"opencsg.com/csghub-server/common/tests"
	"opencsg.com/csghub-server/component"
)

// Injectors from wire.go:

func initializeTestGitCallbackComponent(ctx context.Context, t interface {
	Cleanup(func())
	mock.TestingT
}) *testGitCallbackWithMocks {
	config := component.ProvideTestConfig()
	mockStores := tests.NewMockStores(t)
	mockGitServer := gitserver.NewMockGitServer(t)
	mockTagComponent := component2.NewMockTagComponent(t)
	mockModerationSvcClient := rpc.NewMockModerationSvcClient(t)
	mockRuntimeArchitectureComponent := component2.NewMockRuntimeArchitectureComponent(t)
	mockSpaceComponent := component2.NewMockSpaceComponent(t)
	callbackGitCallbackComponentImpl := NewTestGitCallbackComponent(config, mockStores, mockGitServer, mockTagComponent, mockModerationSvcClient, mockRuntimeArchitectureComponent, mockSpaceComponent)
	mocks := &Mocks{
		stores:               mockStores,
		tagComponent:         mockTagComponent,
		spaceComponent:       mockSpaceComponent,
		gitServer:            mockGitServer,
		runtimeArchComponent: mockRuntimeArchitectureComponent,
	}
	callbackTestGitCallbackWithMocks := &testGitCallbackWithMocks{
		gitCallbackComponentImpl: callbackGitCallbackComponentImpl,
		mocks:                    mocks,
	}
	return callbackTestGitCallbackWithMocks
}

// wire.go:

type testGitCallbackWithMocks struct {
	*gitCallbackComponentImpl
	mocks *Mocks
}