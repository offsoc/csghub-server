package component

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/stretchr/testify/require"
	"opencsg.com/csghub-server/builder/deploy"
	"opencsg.com/csghub-server/builder/deploy/scheduler"
	"opencsg.com/csghub-server/builder/git/gitserver"
	"opencsg.com/csghub-server/builder/git/membership"
	"opencsg.com/csghub-server/builder/store/database"
	"opencsg.com/csghub-server/common/types"
)

// func TestSpaceComponent_Create(t *testing.T) {
// 	ctx := context.TODO()
// 	sc := initializeTestSpaceComponent(ctx, t)

// 	sc.mocks.stores.SpaceResourceMock().EXPECT().FindByID(ctx, int64(1)).Return(&database.SpaceResource{
// 		ID:        1,
// 		Name:      "sp",
// 		Resources: `{"memory": "foo"}`,
// 	}, nil)

// 	sc.mocks.deployer.EXPECT().CheckResourceAvailable(ctx, int64(0), &types.HardWare{
// 		Memory: "foo",
// 	}).Return(true, nil)

// 	sc.mocks.components.repo.EXPECT().CreateRepo(ctx, types.CreateRepoReq{
// 		DefaultBranch: "main",
// 		Readme:        generateReadmeData("MIT"),
// 		License:       "MIT",
// 		Namespace:     "ns",
// 		Name:          "n",
// 		Nickname:      "n",
// 		RepoType:      types.SpaceRepo,
// 		Username:      "user",
// 	}).Return(nil, &database.Repository{
// 		ID: 321,
// 		User: database.User{
// 			Username: "user",
// 			Email:    "foo@bar.com",
// 		},
// 	}, nil)

// 	sc.mocks.stores.SpaceMock().EXPECT().Create(ctx, database.Space{
// 		RepositoryID: 321,
// 		Sdk:          scheduler.STREAMLIT.Name,
// 		SdkVersion:   "v1",
// 		Env:          "env",
// 		Hardware:     `{"memory": "foo"}`,
// 		Secrets:      "sss",
// 		SKU:          "1",
// 	}).Return(&database.Space{}, nil)
// 	sc.mocks.gitServer.EXPECT().CreateRepoFile(buildCreateFileReq(&types.CreateFileParams{
// 		Username:  "user",
// 		Email:     "foo@bar.com",
// 		Message:   initCommitMessage,
// 		Branch:    "main",
// 		Content:   generateReadmeData("MIT"),
// 		NewBranch: "main",
// 		Namespace: "ns",
// 		Name:      "n",
// 		FilePath:  readmeFileName,
// 	}, types.SpaceRepo)).Return(nil)
// 	sc.mocks.gitServer.EXPECT().CreateRepoFile(buildCreateFileReq(&types.CreateFileParams{
// 		Username:  "user",
// 		Email:     "foo@bar.com",
// 		Message:   initCommitMessage,
// 		Branch:    "main",
// 		Content:   spaceGitattributesContent,
// 		NewBranch: "main",
// 		Namespace: "ns",
// 		Name:      "n",
// 		FilePath:  gitattributesFileName,
// 	}, types.SpaceRepo)).Return(nil)
// 	sc.mocks.gitServer.EXPECT().CreateRepoFile(buildCreateFileReq(&types.CreateFileParams{
// 		Username:  "user",
// 		Email:     "foo@bar.com",
// 		Message:   initCommitMessage,
// 		Branch:    "main",
// 		Content:   streamlitConfigContent,
// 		NewBranch: "main",
// 		Namespace: "ns",
// 		Name:      "n",
// 		FilePath:  streamlitConfig,
// 	}, types.SpaceRepo)).Return(nil)

// 	space, err := sc.Create(ctx, types.CreateSpaceReq{
// 		Sdk:        scheduler.STREAMLIT.Name,
// 		SdkVersion: "v1",
// 		Env:        "env",
// 		Secrets:    "sss",
// 		ResourceID: 1,
// 		ClusterID:  "cluster",
// 		CreateRepoReq: types.CreateRepoReq{
// 			DefaultBranch: "main",
// 			Readme:        "readme",
// 			Namespace:     "ns",
// 			Name:          "n",
// 			License:       "MIT",
// 			Username:      "user",
// 		},
// 	})
// 	require.Nil(t, err)

// 	require.Equal(t, &types.Space{
// 		License:    "MIT",
// 		Name:       "n",
// 		Sdk:        "streamlit",
// 		SdkVersion: "v1",
// 		Env:        "env",
// 		Secrets:    "sss",
// 		Hardware:   `{"memory": "foo"}`,
// 		Creator:    "user",
// 	}, space)

// }

func TestSpaceComponent_Show(t *testing.T) {
	ctx := context.TODO()
	sc := initializeTestSpaceComponent(ctx, t)

	sc.mocks.stores.SpaceMock().EXPECT().FindByPath(ctx, "ns", "n").Return(&database.Space{
		ID:         1,
		Repository: &database.Repository{ID: 123, Name: "n", Path: "foo/bar"},
		HasAppFile: true,
	}, nil)
	sc.mocks.components.repo.EXPECT().GetUserRepoPermission(ctx, "user", &database.Repository{
		ID:   123,
		Name: "n",
		Path: "foo/bar",
	}).Return(
		&types.UserRepoPermission{CanRead: true, CanAdmin: true}, nil,
	)
	sc.mocks.components.repo.EXPECT().GetNameSpaceInfo(ctx, "ns").Return(&types.Namespace{Path: "ns"}, nil)

	sc.mocks.stores.DeployTaskMock().EXPECT().GetLatestDeployBySpaceID(ctx, int64(1)).Return(
		&database.Deploy{}, nil,
	)
	sc.mocks.deployer.EXPECT().Status(ctx, types.DeployRepo{
		Namespace: "foo",
		Name:      "bar",
	}, true).Return("svc", 1, nil, nil)

	sc.mocks.stores.UserLikesMock().EXPECT().IsExist(ctx, "user", int64(123)).Return(true, nil)

	space, err := sc.Show(ctx, "ns", "n", "user")
	require.Nil(t, err)
	require.Equal(t, &types.Space{
		ID:           1,
		Name:         "n",
		Namespace:    &types.Namespace{Path: "ns"},
		UserLikes:    true,
		RepositoryID: 123,
		Status:       "Stopped",
		CanManage:    true,
		User:         &types.User{},
		Path:         "foo/bar",
		Repository: &types.Repository{
			HTTPCloneURL: "/s/foo/bar.git",
			SSHCloneURL:  ":s/foo/bar.git",
		},
		Endpoint: "endpoint/svc",
		SvcName:  "svc",
	}, space)
}

func TestSpaceComponent_Update(t *testing.T) {
	ctx := context.TODO()
	sc := initializeTestSpaceComponent(ctx, t)

	sc.mocks.stores.SpaceResourceMock().EXPECT().FindByID(ctx, int64(12)).Return(&database.SpaceResource{
		ID:        12,
		Name:      "sp",
		Resources: `{"memory": "foo"}`,
	}, nil)

	sc.mocks.components.repo.EXPECT().UpdateRepo(ctx, types.UpdateRepoReq{
		Username:  "user",
		Namespace: "ns",
		Name:      "n",
		RepoType:  types.SpaceRepo,
	}).Return(
		&database.Repository{
			ID:   123,
			Name: "repo",
		}, nil,
	)
	sc.mocks.stores.SpaceMock().EXPECT().ByRepoID(ctx, int64(123)).Return(&database.Space{
		ID: 321,
	}, nil)
	sc.mocks.stores.SpaceMock().EXPECT().Update(ctx, database.Space{
		ID:       321,
		Hardware: `{"memory": "foo"}`,
		SKU:      "12",
	}).Return(nil)

	space, err := sc.Update(ctx, &types.UpdateSpaceReq{
		ResourceID: tea.Int64(12),
		UpdateRepoReq: types.UpdateRepoReq{
			Username:  "user",
			Namespace: "ns",
			Name:      "n",
		},
	})
	require.Nil(t, err)

	require.Equal(t, &types.Space{
		ID:       321,
		Name:     "repo",
		Hardware: `{"memory": "foo"}`,
		SKU:      "12",
	}, space)

}

func TestSpaceComponent_Index(t *testing.T) {
	ctx := context.TODO()
	sc := initializeTestSpaceComponent(ctx, t)

	sc.mocks.components.repo.EXPECT().PublicToUser(
		ctx, types.SpaceRepo, "user", &types.RepoFilter{Sort: "z", Username: "user"}, 10, 1,
	).Return([]*database.Repository{
		{ID: 123, Name: "r1", Tags: []database.Tag{{Name: "t1"}}},
		{ID: 124, Name: "r2", Tags: []database.Tag{{Name: "t2"}}},
	}, 100, nil)

	sc.mocks.stores.SpaceMock().EXPECT().ByRepoIDs(ctx, []int64{123, 124}).Return(
		[]database.Space{
			{ID: 11, RepositoryID: 123, Repository: &database.Repository{
				ID:   123,
				Name: "r1",
			}},
			{ID: 12, RepositoryID: 124, Repository: &database.Repository{
				ID:   124,
				Name: "r2",
			}},
		}, nil,
	)

	data, total, err := sc.Index(ctx, &types.RepoFilter{Sort: "z", Username: "user"}, 10, 1)
	require.Nil(t, err)
	require.Equal(t, 100, total)
	require.Equal(t, []types.Space{
		{
			RepositoryID: 123, Name: "r1", Tags: []types.RepoTag{{Name: "t1"}},
			Status: "NoAppFile",
		},
		{
			RepositoryID: 124, Name: "r2", Tags: []types.RepoTag{{Name: "t2"}},
			Status: "NoAppFile",
		},
	}, data)

}

func TestSpaceComponent_OrgSpaces(t *testing.T) {
	ctx := context.TODO()
	sc := initializeTestSpaceComponent(ctx, t)

	sc.mocks.userSvcClient.EXPECT().GetMemberRole(ctx, "ns", "user").Return(membership.RoleAdmin, nil)
	sc.mocks.stores.SpaceMock().EXPECT().ByOrgPath(ctx, "ns", 10, 1, false).Return(
		[]database.Space{
			{ID: 1, Repository: &database.Repository{ID: 11, Name: "r1"}},
			{ID: 2, Repository: &database.Repository{ID: 12, Name: "r2"}},
		}, 100, nil,
	)

	data, total, err := sc.OrgSpaces(ctx, &types.OrgDatasetsReq{
		Namespace:   "ns",
		CurrentUser: "user",
		PageOpts: types.PageOpts{
			Page:     1,
			PageSize: 10,
		},
	})
	require.Nil(t, err)
	require.Equal(t, 100, total)
	require.Equal(t, []types.Space{
		{ID: 1, Name: "r1", RepositoryID: 11, Status: "NoAppFile"},
		{ID: 2, Name: "r2", RepositoryID: 12, Status: "NoAppFile"},
	}, data)

}

func TestSpaceComponent_UserSpaces(t *testing.T) {
	ctx := context.TODO()
	sc := initializeTestSpaceComponent(ctx, t)

	sc.mocks.stores.SpaceMock().EXPECT().ByUsername(ctx, "owner", 10, 1, true).Return(
		[]database.Space{
			{ID: 1, RepositoryID: 11, Repository: &database.Repository{ID: 11, Name: "r1"}},
			{ID: 2, RepositoryID: 12, Repository: &database.Repository{ID: 12, Name: "r2"}},
		}, 100, nil,
	)

	data, total, err := sc.UserSpaces(ctx, &types.UserDatasetsReq{
		Owner:       "owner",
		CurrentUser: "user",
		PageOpts: types.PageOpts{
			Page:     1,
			PageSize: 10,
		},
	})
	require.Nil(t, err)
	require.Equal(t, 100, total)
	require.Equal(t, []types.Space{
		{ID: 1, Name: "r1", RepositoryID: 11, Status: "NoAppFile"},
		{ID: 2, Name: "r2", RepositoryID: 12, Status: "NoAppFile"},
	}, data)

}

func TestSpaceComponent_UserLikeSpaces(t *testing.T) {
	ctx := context.TODO()
	sc := initializeTestSpaceComponent(ctx, t)

	sc.mocks.stores.SpaceMock().EXPECT().ByUserLikes(ctx, int64(111), 10, 1).Return(
		[]database.Space{
			{ID: 1, RepositoryID: 11, Repository: &database.Repository{ID: 11, Name: "r1"}},
			{ID: 2, RepositoryID: 12, Repository: &database.Repository{ID: 12, Name: "r2"}},
		}, 100, nil,
	)

	data, total, err := sc.UserLikesSpaces(ctx, &types.UserDatasetsReq{
		Owner:       "owner",
		CurrentUser: "user",
		PageOpts: types.PageOpts{
			Page:     1,
			PageSize: 10,
		},
	}, 111)
	require.Nil(t, err)
	require.Equal(t, 100, total)
	require.Equal(t, []types.Space{
		{ID: 1, Name: "r1", Status: "NoAppFile"},
		{ID: 2, Name: "r2", Status: "NoAppFile"},
	}, data)

}

func TestSpaceComponent_ListByPath(t *testing.T) {
	ctx := context.TODO()
	sc := initializeTestSpaceComponent(ctx, t)

	sc.mocks.stores.SpaceMock().EXPECT().ListByPath(ctx, []string{"foo"}).Return(
		[]database.Space{
			{ID: 1, RepositoryID: 11, Repository: &database.Repository{ID: 11, Name: "r1"}},
			{ID: 2, RepositoryID: 12, Repository: &database.Repository{ID: 12, Name: "r2"}},
		}, nil,
	)

	data, err := sc.ListByPath(ctx, []string{"foo"})
	require.Nil(t, err)
	require.Equal(t, []*types.Space{
		{Name: "r1", Status: "NoAppFile", RepositoryID: 11},
		{Name: "r2", Status: "NoAppFile", RepositoryID: 12},
	}, data)

}

func TestSpaceComponent_AllowCallApi(t *testing.T) {
	ctx := context.TODO()
	sc := initializeTestSpaceComponent(ctx, t)

	sc.mocks.stores.SpaceMock().EXPECT().ByID(ctx, int64(123)).Return(&database.Space{
		Repository: &database.Repository{Path: "foo/bar", RepositoryType: types.ModelRepo},
	}, nil)

	sc.mocks.components.repo.EXPECT().AllowReadAccess(ctx, types.ModelRepo, "foo", "bar", "user").Return(true, nil)
	allow, err := sc.AllowCallApi(ctx, 123, "user")
	require.Nil(t, err)
	require.True(t, allow)

}

func TestSpaceComponent_Delete(t *testing.T) {
	ctx := context.TODO()
	sc := initializeTestSpaceComponent(ctx, t)

	sc.mocks.stores.SpaceMock().EXPECT().FindByPath(ctx, "ns", "n").Return(&database.Space{ID: 1}, nil)
	sc.mocks.components.repo.EXPECT().DeleteRepo(ctx, types.DeleteRepoReq{
		Username:  "user",
		Namespace: "ns",
		Name:      "n",
		RepoType:  types.SpaceRepo,
	}).Return(nil, nil)
	sc.mocks.stores.SpaceMock().EXPECT().Delete(ctx, database.Space{ID: 1}).Return(nil)
	sc.mocks.stores.DeployTaskMock().EXPECT().GetLatestDeployBySpaceID(ctx, int64(1)).Return(
		&database.Deploy{
			RepoID: 2,
			UserID: 3,
			ID:     4,
		}, nil,
	)
	sc.mocks.deployer.EXPECT().Stop(ctx, types.DeployRepo{
		SpaceID:   1,
		Namespace: "ns",
		Name:      "n",
	}).Return(nil)
	sc.mocks.stores.DeployTaskMock().EXPECT().StopDeploy(
		ctx, types.SpaceRepo, int64(2), int64(3), int64(4),
	).Return(nil)

	err := sc.Delete(ctx, "ns", "n", "user")
	time.Sleep(1 * time.Second)
	require.Nil(t, err)

}

func TestSpaceComponent_Deploy(t *testing.T) {
	ctx := context.TODO()
	sc := initializeTestSpaceComponent(ctx, t)

	sc.mocks.stores.SpaceMock().EXPECT().FindByPath(ctx, "ns", "n").Return(&database.Space{
		ID:         1,
		Repository: &database.Repository{Path: "foo/bar"},
	}, nil)
	sc.mocks.stores.UserMock().EXPECT().FindByUsername(ctx, "user").Return(database.User{}, nil)
	sc.mocks.deployer.EXPECT().Deploy(ctx, types.DeployRepo{
		SpaceID:    1,
		Path:       "foo/bar",
		Annotation: "{\"hub-res-name\":\"ns/n\",\"hub-res-type\":\"space\"}",
	}).Return(123, nil)

	id, err := sc.Deploy(ctx, "ns", "n", "user")
	require.Nil(t, err)
	require.Equal(t, int64(123), id)

}

func TestSpaceComponent_Wakeup(t *testing.T) {
	ctx := context.TODO()
	sc := initializeTestSpaceComponent(ctx, t)
	sc.mocks.stores.SpaceMock().EXPECT().FindByPath(ctx, "ns", "n").Return(&database.Space{
		ID: 1,
	}, nil)

	sc.mocks.stores.DeployTaskMock().EXPECT().GetLatestDeployBySpaceID(ctx, int64(1)).Return(
		&database.Deploy{SvcName: "svc"}, nil,
	)

	sc.mocks.deployer.EXPECT().Wakeup(ctx, types.DeployRepo{
		SpaceID:   1,
		Namespace: "ns",
		Name:      "n",
		SvcName:   "svc",
	}).Return(nil)

	err := sc.Wakeup(ctx, "ns", "n")
	require.Nil(t, err)

}

func TestSpaceComponent_FixHasEntryFile(t *testing.T) {

	cases := []struct {
		nginx bool
		name  string
		tp    string
		exist bool
	}{
		{false, "app.py", "file", true},
		{false, "app.py", "foo", false},
		{false, "z.py", "file", false},
		{true, "app.py", "file", false},
		{true, "nginx.conf", "file", true},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%+v", c), func(t *testing.T) {

			ctx := context.TODO()
			sc := initializeTestSpaceComponent(ctx, t)

			sc.mocks.gitServer.EXPECT().GetRepoFileTree(ctx, gitserver.GetRepoInfoByPathReq{
				Namespace: "foo",
				Name:      "bar",
				RepoType:  types.SpaceRepo,
			}).Return([]*types.File{
				{Type: c.tp, Path: c.name},
			}, nil)
			sdk := ""
			if c.nginx {
				sdk = scheduler.NGINX.Name
			}
			exist := sc.HasEntryFile(ctx, &database.Space{
				Repository: &database.Repository{Path: "foo/bar"},
				Sdk:        sdk,
			})
			require.Equal(t, c.exist, exist)
		})
	}
}

func TestSpaceComponent_Logs(t *testing.T) {
	ctx := context.TODO()
	sc := initializeTestSpaceComponent(ctx, t)
	sc.mocks.stores.SpaceMock().EXPECT().FindByPath(ctx, "ns", "n").Return(&database.Space{
		ID: 1,
	}, nil)

	sc.mocks.deployer.EXPECT().Logs(ctx, types.DeployRepo{
		SpaceID:   1,
		Namespace: "ns",
		Name:      "n",
	}).Return(&deploy.MultiLogReader{}, nil)

	r, err := sc.Logs(ctx, "ns", "n")

	require.Nil(t, err)
	require.Equal(t, &deploy.MultiLogReader{}, r)

}