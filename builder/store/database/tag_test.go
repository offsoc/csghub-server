package database_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"opencsg.com/csghub-server/builder/store/database"
	"opencsg.com/csghub-server/common/tests"
	"opencsg.com/csghub-server/common/types"
)

func TestTagStore_FindOrCreate(t *testing.T) {
	db := tests.InitTestDB()
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ts := database.NewTagStoreWithDB(db)
	uuid := uuid.New().String()
	tag := database.Tag{
		Name:     "tag_" + uuid,
		Category: "task",
		Group:    "",
		Scope:    "model",
		BuiltIn:  true,
		ShowName: "Tag One",
	}
	newTag, err := ts.FindOrCreate(ctx, tag)
	require.Empty(t, err)
	require.NotEmpty(t, newTag.ID)
	require.Equal(t, tag.Name, newTag.Name)

	existingTag, err := ts.FindOrCreate(ctx, tag)
	require.Empty(t, err)
	require.Equal(t, newTag.ID, existingTag.ID)
}

func TestTagStore_AllTags(t *testing.T) {
	db := tests.InitTestDB()
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var err error
	//clear all existing data
	_, err = db.Core.NewDelete().Model(&database.Tag{}).Where("1=1").Exec(ctx)
	require.Empty(t, err)

	ts := database.NewTagStoreWithDB(db)
	tags, err := ts.AllTags(ctx)
	require.Empty(t, err)
	require.Equal(t, 0, len(tags))

	modelTag, err := ts.CreateTag(ctx, "task", "tag_"+uuid.New().String(), "Group One", database.ModelTagScope)
	require.Empty(t, err)
	datasetTag, err := ts.CreateTag(ctx, "library", "tag_"+uuid.New().String(), "Group One", database.DatasetTagScope)
	require.Empty(t, err)
	codeTag, err := ts.CreateTag(ctx, "task", "tag_"+uuid.New().String(), "Group One", database.CodeTagScope)
	require.Empty(t, err)
	promptTag, err := ts.CreateTag(ctx, "library", "tag_"+uuid.New().String(), "Group One", database.PromptTagScope)
	require.Empty(t, err)
	spaceTag, err := ts.CreateTag(ctx, "library", "tag_"+uuid.New().String(), "Group One", database.SpaceTagScope)
	require.Empty(t, err)

	tags, err = ts.AllTags(ctx)
	require.Empty(t, err)
	require.Equal(t, 5, len(tags))

	modelTags, err := ts.AllModelTags(ctx)
	require.Empty(t, err)
	require.Equal(t, 1, len(modelTags))
	require.Equal(t, modelTag.Name, modelTags[0].Name)

	datasetTags, err := ts.AllDatasetTags(ctx)
	require.Empty(t, err)
	require.Equal(t, 1, len(datasetTags))
	require.Equal(t, datasetTag.Name, datasetTags[0].Name)

	codeTags, err := ts.AllCodeTags(ctx)
	require.Empty(t, err)
	require.Equal(t, 1, len(codeTags))
	require.Equal(t, codeTag.Name, codeTags[0].Name)

	promptTags, err := ts.AllPromptTags(ctx)
	require.Empty(t, err)
	require.Equal(t, 1, len(promptTags))
	require.Equal(t, promptTag.Name, promptTags[0].Name)

	spaceTags, err := ts.AllSpaceTags(ctx)
	require.Empty(t, err)
	require.Equal(t, 1, len(spaceTags))
	require.Equal(t, spaceTag.Name, spaceTags[0].Name)
}

// TestAllTagsByScope tests the AllTagsByScope method
func TestTagStore_AllTagsByScope(t *testing.T) {
	db := tests.InitTestDB()
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ts := database.NewTagStoreWithDB(db)
	_, err := ts.CreateTag(ctx, "task", "tag_"+uuid.New().String(), "Group One", "test_scope")
	require.Empty(t, err)
	_, err = ts.CreateTag(ctx, "task", "tag_"+uuid.New().String(), "Group One", "test_scope")
	require.Empty(t, err)

	tags, err := ts.AllTagsByScope(ctx, "test_scope")
	require.Empty(t, err)
	require.Len(t, tags, 2)
	require.Equal(t, string(tags[0].Scope), "test_scope")
	require.Equal(t, string(tags[1].Scope), "test_scope")
}

// TestAllTagsByScopeAndCategory tests the AllTagsByScopeAndCategory method
func TestTagStore_AllTagsByScopeAndCategory(t *testing.T) {
	db := tests.InitTestDB()
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ts := database.NewTagStoreWithDB(db)
	_, err := ts.CreateTag(ctx, "task", "tag_"+uuid.New().String(), "Group One", "test_scope")
	require.Empty(t, err)
	_, err = ts.CreateTag(ctx, "library", "tag_"+uuid.New().String(), "Group One", "test_scope")
	require.Empty(t, err)

	tags, err := ts.AllTagsByScopeAndCategory(ctx, "test_scope", "task")
	require.Empty(t, err)
	require.Len(t, tags, 1)
	require.Equal(t, string(tags[0].Scope), "test_scope")
}

// TestGetTagsByScopeAndCategories tests the GetTagsByScopeAndCategories method
func TestTagStore_GetTagsByScopeAndCategories(t *testing.T) {
	db := tests.InitTestDB()
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ts := database.NewTagStoreWithDB(db)
	_, err := ts.CreateTag(ctx, "task", "tag_"+uuid.New().String(), "Group One", "test_scope")
	require.Empty(t, err)
	_, err = ts.CreateTag(ctx, "library", "tag_"+uuid.New().String(), "Group One", "test_scope")
	require.Empty(t, err)
	_, err = ts.CreateTag(ctx, "test_category", "tag_"+uuid.New().String(), "Group One", "test_scope")
	require.Empty(t, err)

	tags, err := ts.GetTagsByScopeAndCategories(ctx, "test_scope", []string{"task", "library"})
	require.Empty(t, err)
	require.Len(t, tags, 2)
}

// TestAllModelCategories tests the AllModelCategories method
func TestTagStore_AllModelCategories(t *testing.T) {
	db := tests.InitTestDB()
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var err error
	//clear all existing data
	_, err = db.Core.NewDelete().Model(&database.TagCategory{}).Where("1=1").Exec(ctx)
	require.Empty(t, err)

	ts := database.NewTagStoreWithDB(db)
	tags, err := ts.AllModelCategories(ctx)
	require.Empty(t, err)
	require.Len(t, tags, 0)

	tc1 := &database.TagCategory{
		Name:  "tc_1",
		Scope: database.ModelTagScope,
	}
	tc2 := &database.TagCategory{
		Name:  "tc_2",
		Scope: database.DatasetTagScope,
	}
	_, err = db.Core.NewInsert().Model(tc1).Exec(ctx)
	require.Empty(t, err)
	_, err = db.Core.NewInsert().Model(tc2).Exec(ctx)
	require.Empty(t, err)

	tags, err = ts.AllModelCategories(ctx)
	require.Empty(t, err)
	require.Len(t, tags, 1)
}

// TestAllPromptCategories tests the AllPromptCategories method
func TestTagStore_AllPromptCategories(t *testing.T) {
	db := tests.InitTestDB()
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var err error
	//clear all existing data
	_, err = db.Core.NewDelete().Model(&database.TagCategory{}).Where("1=1").Exec(ctx)
	require.Empty(t, err)

	ts := database.NewTagStoreWithDB(db)
	tags, err := ts.AllPromptCategories(ctx)
	require.Empty(t, err)
	require.Len(t, tags, 0)

	tc1 := &database.TagCategory{
		Name:  "tc_1",
		Scope: database.ModelTagScope,
	}
	tc2 := &database.TagCategory{
		Name:  "tc_2",
		Scope: database.PromptTagScope,
	}
	_, err = db.Core.NewInsert().Model(tc1).Exec(ctx)
	require.Empty(t, err)
	_, err = db.Core.NewInsert().Model(tc2).Exec(ctx)
	require.Empty(t, err)

	tags, err = ts.AllPromptCategories(ctx)
	require.Empty(t, err)
	require.Len(t, tags, 1)
}

// TestAllDatasetCategories tests the AllDatasetCategories method
func TestTagStore_AllDatasetCategories(t *testing.T) {
	db := tests.InitTestDB()
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var err error
	//clear all existing data
	_, err = db.Core.NewDelete().Model(&database.TagCategory{}).Where("1=1").Exec(ctx)
	require.Empty(t, err)

	ts := database.NewTagStoreWithDB(db)
	tags, err := ts.AllDatasetCategories(ctx)
	require.Empty(t, err)
	require.Len(t, tags, 0)

	tc1 := &database.TagCategory{
		Name:  "tc_1",
		Scope: database.ModelTagScope,
	}
	tc2 := &database.TagCategory{
		Name:  "tc_2",
		Scope: database.DatasetTagScope,
	}
	_, err = db.Core.NewInsert().Model(tc1).Exec(ctx)
	require.Empty(t, err)
	_, err = db.Core.NewInsert().Model(tc2).Exec(ctx)
	require.Empty(t, err)

	tags, err = ts.AllDatasetCategories(ctx)
	require.Empty(t, err)
	require.Len(t, tags, 1)
}

// TestAllCodeCategories tests the AllCodeCategories method
func TestTagStore_AllCodeCategories(t *testing.T) {
	db := tests.InitTestDB()
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var err error
	//clear all existing data
	_, err = db.Core.NewDelete().Model(&database.TagCategory{}).Where("1=1").Exec(ctx)
	require.Empty(t, err)

	ts := database.NewTagStoreWithDB(db)
	tags, err := ts.AllCodeCategories(ctx)
	require.Empty(t, err)
	require.Len(t, tags, 0)

	tc1 := &database.TagCategory{
		Name:  "tc_1",
		Scope: database.ModelTagScope,
	}
	tc2 := &database.TagCategory{
		Name:  "tc_2",
		Scope: database.CodeTagScope,
	}
	_, err = db.Core.NewInsert().Model(tc1).Exec(ctx)
	require.Empty(t, err)
	_, err = db.Core.NewInsert().Model(tc2).Exec(ctx)
	require.Empty(t, err)

	tags, err = ts.AllCodeCategories(ctx)
	require.Empty(t, err)
	require.Len(t, tags, 1)
}

// TestAllSpaceCategories tests the AllSpaceCategories method
func TestTagStore_AllSpaceCategories(t *testing.T) {
	db := tests.InitTestDB()
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var err error
	//clear all existing data
	_, err = db.Core.NewDelete().Model(&database.TagCategory{}).Where("1=1").Exec(ctx)
	require.Empty(t, err)

	ts := database.NewTagStoreWithDB(db)
	tags, err := ts.AllSpaceCategories(ctx)
	require.Empty(t, err)
	require.Len(t, tags, 0)

	tc1 := &database.TagCategory{
		Name:  "tc_1",
		Scope: database.ModelTagScope,
	}
	tc2 := &database.TagCategory{
		Name:  "tc_2",
		Scope: database.SpaceTagScope,
	}
	_, err = db.Core.NewInsert().Model(tc1).Exec(ctx)
	require.Empty(t, err)
	_, err = db.Core.NewInsert().Model(tc2).Exec(ctx)
	require.Empty(t, err)

	tags, err = ts.AllSpaceCategories(ctx)
	require.Empty(t, err)
	require.Len(t, tags, 1)
}

// TestCreateTag tests the CreateTag method
func TestTagStore_CreateTag(t *testing.T) {
	db := tests.InitTestDB()
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var err error
	ts := database.NewTagStoreWithDB(db)
	t1, err := ts.CreateTag(ctx, "task", "tag_"+uuid.NewString(), "", database.ModelTagScope)
	require.Empty(t, err)
	require.NotEmpty(t, t1.ID)
}

// TestSetMetaTags tests the SetMetaTags method
func TestTagStore_SetMetaTags(t *testing.T) {
	db := tests.InitTestDB()
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// insert a new repo
	rs := database.NewRepoStoreWithDB(db)
	userName := "user_name_" + uuid.NewString()
	repoName := "repo_name_" + uuid.NewString()
	repo, err := rs.CreateRepo(ctx, database.Repository{
		UserID:         1,
		Path:           fmt.Sprintf("%s/%s", userName, repoName),
		GitPath:        fmt.Sprintf("models_%s/%s", userName, repoName),
		Name:           repoName,
		Nickname:       "",
		Description:    "",
		Private:        false,
		RepositoryType: types.ModelRepo,
	})
	require.Empty(t, err)
	require.NotNil(t, repo)

	ts := database.NewTagStoreWithDB(db)
	var tags []*database.Tag
	tags = append(tags, &database.Tag{
		Name:     "tag_" + uuid.NewString(),
		Category: "task",
		Group:    "",
		Scope:    database.ModelTagScope,
		BuiltIn:  true,
		ShowName: "",
	})
	tags = append(tags, &database.Tag{
		Name:     "tag_" + uuid.NewString(),
		Category: "framework",
		Group:    "",
		Scope:    database.ModelTagScope,
		BuiltIn:  true,
		ShowName: "",
	})
	repoTags, err := ts.SetMetaTags(ctx, types.ModelRepo, userName, repoName, tags)
	// should report err as framework tag is not allowed
	require.NotEmpty(t, err)

	tags = tags[:1]
	repoTags, err = ts.SetMetaTags(ctx, types.ModelRepo, userName, repoName, tags)
	require.Empty(t, err)
	require.Len(t, repoTags, 1)
}

// TestSetLibraryTag tests the SetLibraryTag method
func TestTagStore_SetLibraryTag(t *testing.T) {
	db := tests.InitTestDB()
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// insert a new repo
	rs := database.NewRepoStoreWithDB(db)
	userName := "user_name_" + uuid.NewString()
	repoName := "repo_name_" + uuid.NewString()
	repo, err := rs.CreateRepo(ctx, database.Repository{
		UserID:         1,
		Path:           fmt.Sprintf("%s/%s", userName, repoName),
		GitPath:        fmt.Sprintf("models_%s/%s", userName, repoName),
		Name:           repoName,
		Nickname:       "",
		Description:    "",
		Private:        false,
		RepositoryType: types.ModelRepo,
	})
	require.Empty(t, err)
	require.NotNil(t, repo)

	ts := database.NewTagStoreWithDB(db)
	oldTag, err := ts.CreateTag(ctx, "task", "tag_"+uuid.NewString(), "", database.ModelTagScope)
	require.Empty(t, err)
	newTag, err := ts.CreateTag(ctx, "task", "tag_"+uuid.NewString(), "", database.ModelTagScope)
	require.Empty(t, err)

	err = ts.SetLibraryTag(ctx, types.ModelRepo, userName, repoName, &newTag, &oldTag)
	require.Empty(t, err)

	repoTagStore := database.NewRepoStoreWithDB(db)
	tags, err := repoTagStore.Tags(ctx, repo.ID)
	require.Empty(t, err)
	require.Len(t, tags, 1)
}

// TestUpsertRepoTags tests the UpsertRepoTags method
func TestTagStore_UpsertRepoTags(t *testing.T) {
	db := tests.InitTestDB()
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ts := database.NewTagStoreWithDB(db)
	oldTag, err := ts.CreateTag(ctx, "task", "tag_"+uuid.NewString(), "", database.ModelTagScope)
	require.Empty(t, err)

	// insert a new repo with tags
	rs := database.NewRepoStoreWithDB(db)
	userName := "user_name_" + uuid.NewString()
	repoName := "repo_name_" + uuid.NewString()
	repo, err := rs.CreateRepo(ctx, database.Repository{
		UserID:         1,
		Path:           fmt.Sprintf("%s/%s", userName, repoName),
		GitPath:        fmt.Sprintf("models_%s/%s", userName, repoName),
		Name:           repoName,
		Nickname:       "",
		Description:    "",
		Private:        false,
		RepositoryType: types.ModelRepo,
		Tags:           []database.Tag{oldTag},
	})
	require.Empty(t, err)
	require.NotNil(t, repo)
	require.Len(t, repo.Tags, 1)

	oldTagIds := make([]int64, 0, len(repo.Tags))
	oldTagIds = append(oldTagIds, repo.Tags[0].ID)

	newTag, err := ts.CreateTag(ctx, "task", "tag_"+uuid.NewString(), "", database.ModelTagScope)
	require.Empty(t, err)
	newTagIds := make([]int64, 0, 2)
	newTagIds = append(newTagIds, newTag.ID)
	newTagIds = append(newTagIds, oldTag.ID)

	err = ts.UpsertRepoTags(ctx, repo.ID, oldTagIds, newTagIds)
	require.Empty(t, err)
	repoTags, err := rs.Tags(ctx, repo.ID)
	require.Empty(t, err)
	require.Len(t, repoTags, 2)
}

// TestRemoveRepoTags tests the RemoveRepoTags method
func TestTagStore_RemoveRepoTags(t *testing.T) {
	db := tests.InitTestDB()
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ts := database.NewTagStoreWithDB(db)
	tag1, err := ts.CreateTag(ctx, "task", "tag_"+uuid.NewString(), "", database.ModelTagScope)
	require.Empty(t, err)
	require.NotEmpty(t, tag1.ID)
	tag2, err := ts.CreateTag(ctx, "task", "tag_"+uuid.NewString(), "", database.ModelTagScope)
	require.Empty(t, err)
	require.NotEmpty(t, tag2.ID)

	// insert a new repo with tags
	rs := database.NewRepoStoreWithDB(db)
	userName := "user_name_" + uuid.NewString()
	repoName := "repo_name_" + uuid.NewString()
	repo, err := rs.CreateRepo(ctx, database.Repository{
		UserID:         1,
		Path:           fmt.Sprintf("%s/%s", userName, repoName),
		GitPath:        fmt.Sprintf("models_%s/%s", userName, repoName),
		Name:           repoName,
		Nickname:       "",
		Description:    "",
		Private:        false,
		RepositoryType: types.ModelRepo,
	})
	require.Empty(t, err)
	require.NotNil(t, repo)
	// set repo tags
	ts.UpsertRepoTags(ctx, repo.ID, []int64{}, []int64{tag1.ID, tag2.ID})

	removeTagIds := make([]int64, 0, 1)
	removeTagIds = append(removeTagIds, tag2.ID)

	err = ts.RemoveRepoTags(ctx, repo.ID, removeTagIds)
	require.Empty(t, err)
	repoTags, err := rs.Tags(ctx, repo.ID)
	require.Empty(t, err)
	require.Len(t, repoTags, 1)
	require.EqualValues(t, repoTags[0], tag1)

}