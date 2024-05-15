package component

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"opencsg.com/csghub-server/builder/git"
	"opencsg.com/csghub-server/builder/git/mirrorserver"
	"opencsg.com/csghub-server/builder/store/database"
	"opencsg.com/csghub-server/common/config"
	"opencsg.com/csghub-server/common/types"
)

type MirrorComponent struct {
	mirrorStore  *database.MirrorStore
	repoStore    *database.RepoStore
	repoComp     *RepoComponent
	mirrorServer mirrorserver.MirrorServer
}

func NewMirrorComponent(config *config.Config) (*MirrorComponent, error) {
	var err error
	c := &MirrorComponent{}
	c.mirrorServer, err = git.NewMirrorServer(config)
	if err != nil {
		newError := fmt.Errorf("fail to create git mirror server,error:%w", err)
		slog.Error(newError.Error())
		return nil, newError
	}
	c.repoComp, err = NewRepoComponent(config)
	if err != nil {
		return nil, fmt.Errorf("fail to create repo component,error:%w", err)
	}
	c.repoStore = database.NewRepoStore()
	c.mirrorStore = database.NewMirrorStore()
	return c, nil
}

func (c *MirrorComponent) CreatePushMirrorForFinishedMirrorTask(ctx context.Context) error {
	mirrors, err := c.mirrorStore.NoPushMirror(ctx)
	if err != nil {
		return fmt.Errorf("fail to find all mirrors, %w", err)
	}

	for _, mirror := range mirrors {
		task, err := c.mirrorServer.GetMirrorTaskInfo(ctx, mirror.MirrorTaskID)
		if err != nil {
			slog.Error("fail to get mirror task info", slog.Int64("taskId", mirror.MirrorTaskID), slog.String("error", err.Error()))
			return fmt.Errorf("fail to get mirror task info, %w", err)
		}
		if task.Status == mirrorserver.TaskStatusFinished {
			err = c.mirrorServer.CreatePushMirror(ctx, mirrorserver.CreatePushMirrorReq{
				Name:        mirror.LocalRepoPath,
				PushUrl:     mirror.PushUrl,
				Username:    mirror.PushUsername,
				AccessToken: mirror.PushAccessToken,
				Interval:    "8h",
			})

			if err != nil {
				slog.Error("fail to create push mirror", slog.Int64("mirrorId", mirror.ID), slog.String("error", err.Error()))
				return fmt.Errorf("fail to create mirror, %w", err)
			}
			mirror.PushMirrorCreated = true
			err = c.mirrorStore.Update(ctx, &mirror)
			if err != nil {
				slog.Error("fail to update mirror", slog.Int64("mirrorId", mirror.ID), slog.String("error", err.Error()))
				return fmt.Errorf("fail to update mirror, %w", err)
			}
		}
	}
	return nil
}

// CreateMirrorRepo often called by the crawler server to create new repo which will then be mirrored from other sources
func (c *MirrorComponent) CreateMirrorRepo(ctx context.Context, req types.CreateMirrorRepoReq) (*database.Mirror, error) {
	namespace := c.mapNamespaceAndName(req.SourceNamespace)
	name := req.SourceName
	repo, err := c.repoStore.FindByPath(ctx, req.RepoType, namespace, name)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to check repo existance, error: %w", err)
	}
	if repo != nil {
		name = fmt.Sprintf("%s_%s", req.SourceNamespace, req.SourceName)
		repo, err = c.repoStore.FindByPath(ctx, req.RepoType, namespace, name)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("failed to check repo existance, error: %w", err)
		}
		if repo != nil {
			return nil, fmt.Errorf("repo already exists,repo type:%s, source namespace: %s, source name: %s, target namespace: %s, target name: %s",
				req.RepoType, req.SourceNamespace, req.SourceName, namespace, name)
		}
	}
	//create repo, create mirror repo
	gitRepo, _, err := c.repoComp.CreateRepo(ctx, types.CreateRepoReq{
		Username:  namespace,
		Namespace: namespace,
		Name:      name,
		Nickname:  name,
		//TODO: tranlate description automatically
		Description: req.Description,
		//only mirror public repository
		Private:       false,
		License:       req.License,
		DefaultBranch: req.DefaultBranch,
		RepoType:      req.RepoType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenCSG repo, error: %w", err)
	}
	var mirror database.Mirror
	// mirror.Interval = req.Interval
	mirror.SourceUrl = req.SourceGitCloneUrl
	mirror.MirrorSourceID = req.MirrorSourceID
	mirror.PushUrl = gitRepo.HttpCloneURL
	mirror.Username = req.SourceNamespace
	mirror.PushUsername = "root"
	//TODO: get user git access token from db git access token
	mirror.PushAccessToken = ""
	mirror.RepositoryID = repo.ID

	reqMirror, err := c.mirrorStore.Create(ctx, &mirror)
	if err != nil {
		return nil, fmt.Errorf("failed to create mirror")
	}

	c.mirrorServer.CreateMirrorRepo(ctx, mirrorserver.CreateMirrorRepoReq{
		Namespace: "root",
		Name:      fmt.Sprintf("%d_%s_%s_%s", req.MirrorSourceID, req.RepoType, req.SourceNamespace, req.SourceName),
		CloneUrl:  mirror.SourceUrl,
		// Username:    req.SourceNamespace,
		// AccessToken: mirror.AccessToken,
		Private: false,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create push mirror in mirror server: %v", err)
	}

	return reqMirror, nil

}
func (m *MirrorComponent) mapNamespaceAndName(sourceNamespace string) string {
	namespace := sourceNamespace
	if ns, found := mirrorOrganizationMap[sourceNamespace]; found {
		namespace = ns
	} else {
		//map all organization to AIWizards if not found
		namespace = "AIWizards"
	}

	return namespace
}

var mirrorOrganizationMap = map[string]string{
	"THUDM":          "THUDM",
	"baichuan-inc":   "BaiChuanAI",
	"IDEA-CCNL":      "FengShenBang",
	"internlm":       "ShangHaiAILab",
	"pleisto":        "Pleisto",
	"01-ai":          "01AI",
	"codefuse-ai":    "codefuse-ai",
	"WisdomShell":    "WisdomShell",
	"microsoft":      "microsoft",
	"Skywork":        "Skywork",
	"BAAI":           "BAAI",
	"deepseek-ai":    "deepseek-ai",
	"WizardLMTeam":   "WizardLM",
	"IEITYuan":       "IEITYuan",
	"Qwen":           "Qwen",
	"TencentARC":     "TencentARC",
	"OrionStarAI":    "OrionStarAI",
	"openbmb":        "OpenBMB",
	"netease-youdao": "Netease-youdao",
	"ByteDance":      "ByteDance",
}