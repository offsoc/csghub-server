package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "knative.dev/serving/pkg/apis/serving/v1"
	"opencsg.com/csghub-server/builder/deploy/cluster"
	"opencsg.com/csghub-server/builder/deploy/common"
	"opencsg.com/csghub-server/builder/store/database"
	"opencsg.com/csghub-server/common/config"
	"opencsg.com/csghub-server/common/types"
	"opencsg.com/csghub-server/runner/component"
)

type K8sHandler struct {
	clusterPool        *cluster.ClusterPool
	k8sNameSpace       string
	modelDockerRegBase string
	env                *config.Config
	s                  *component.ServiceComponent
}

func NewK8sHandler(config *config.Config, clusterPool *cluster.ClusterPool) (*K8sHandler, error) {
	domainParts := strings.SplitN(config.Space.InternalRootDomain, ".", 2)
	serviceComponent := component.NewServiceComponent(config, domainParts[0])
	return &K8sHandler{
		k8sNameSpace:       domainParts[0],
		clusterPool:        clusterPool,
		env:                config,
		s:                  serviceComponent,
		modelDockerRegBase: config.Model.DockerRegBase,
	}, nil
}

func (s *K8sHandler) RunService(c *gin.Context) {
	request := &types.SVCRequest{}
	err := c.BindJSON(&request)
	if err != nil {
		slog.Error("runService get bad request", slog.Any("error", err), slog.Any("req", request))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	slog.Debug("Recv request", slog.Any("body", request))

	cluster, err := s.clusterPool.GetClusterByID(c, request.ClusterID)
	if err != nil {
		slog.Error("fail to get cluster ", slog.Any("error", err), slog.Any("req", request))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	srvName := s.getServiceNameFromRequest(c)
	// check if the ksvc exists
	_, err = cluster.KnativeClient.ServingV1().Services(s.k8sNameSpace).Get(c.Request.Context(), srvName, metav1.GetOptions{})
	if err == nil {
		err = s.removeServiceForcely(c, cluster, srvName)
		if err != nil {
			slog.Error("fail to remove service", slog.Any("error", err), slog.Any("req", request))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		slog.Info("service already exists,delete it first", slog.String("srv_name", srvName), slog.Any("image_id", request.ImageID))
	}
	service, err := s.s.GenerateService(c, *cluster, *request, srvName)
	if err != nil {
		slog.Error("fail to generate service ", slog.Any("error", err), slog.Any("req", request))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}
	if request.DeployType != types.SpaceType {
		// dshm volume for multi-gpu share memory
		volumes = append(volumes, corev1.Volume{
			Name: "dshm",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumMemory,
				},
			},
		})

		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "dshm",
			MountPath: "/dev/shm",
		})
	}
	pvcName := srvName
	if request.DeployType == types.InferenceType {
		pvcName = request.UserID
	}
	// add pvc if possible
	// space image was built from user's code, model cache dir is hard to control
	// so no PV cache for space case so far
	if cluster.StorageClass != "" && request.DeployType != types.SpaceType {
		err = s.s.NewPersistentVolumeClaim(pvcName, c, *cluster, request.Hardware)
		if err != nil {
			slog.Error("Failed to create persist volume", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create persist volume"})
			return
		}
		volumes = append(volumes, corev1.Volume{
			Name: "nas-pvc",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
				},
			},
		})

		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "nas-pvc",
			MountPath: "/workspace",
		})
	}
	service.Spec.Template.Spec.Volumes = volumes
	service.Spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts

	slog.Debug("ksvc", slog.Any("knative service", service))

	// create ksvc
	_, err = cluster.KnativeClient.ServingV1().Services(s.k8sNameSpace).Create(c, service, metav1.CreateOptions{})
	if err != nil {
		slog.Error("Failed to create service", "error", err, slog.Int64("deploy_id", request.DeployID),
			slog.String("image_id", request.ImageID),
			slog.String("srv_name", srvName))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create service"})
		return
	}

	slog.Info("service created successfully", slog.String("srv_name", srvName), slog.Int64("deploy_id", request.DeployID))
	c.JSON(http.StatusOK, gin.H{"message": "Service created successfully"})
}

func (s *K8sHandler) StopService(c *gin.Context) {
	var resp types.StopResponse
	var request = &types.StopRequest{}
	err := c.BindJSON(request)

	if err != nil {
		slog.Error("stopService get bad request", slog.Any("error", err), slog.Any("req", request))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cluster, err := s.clusterPool.GetClusterByID(c, request.ClusterID)
	if err != nil {
		slog.Error("fail to get cluster ", slog.Any("error", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	srvName := s.getServiceNameFromRequest(c)
	srv, err := cluster.KnativeClient.ServingV1().Services(s.k8sNameSpace).
		Get(c.Request.Context(), srvName, metav1.GetOptions{})
	if err != nil {
		k8serr := new(k8serrors.StatusError)
		if errors.As(err, &k8serr) {
			if k8serr.Status().Code == http.StatusNotFound {
				slog.Info("stop image skip,service not exist", slog.String("srv_name", srvName), slog.Any("k8s_err", k8serr))
				resp.Code = 0
				resp.Message = "skip,service not exist"
				c.JSON(http.StatusOK, nil)
				return
			}
		}
		slog.Error("stop image failed, cannot get service info", slog.String("srv_name", srvName), slog.Any("error", err),
			slog.String("srv_name", srvName))
		resp.Code = -1
		resp.Message = "failed to get service status"
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	if srv == nil {
		resp.Code = 0
		resp.Message = "service not exist"
		c.JSON(http.StatusOK, resp)
		return
	}
	err = s.removeServiceForcely(c, cluster, srvName)
	if err != nil {
		slog.Error("stop image failed, cannot delete service ", slog.String("srv_name", srvName), slog.Any("error", err),
			slog.String("srv_name", srvName))
		resp.Code = -1
		resp.Message = "failed to get service status"
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	slog.Info("service deleted", slog.String("srv_name", srvName))
	resp.Code = 0
	resp.Message = "service deleted"
	c.JSON(http.StatusOK, resp)
}

func (s *K8sHandler) removeServiceForcely(c *gin.Context, cluster *cluster.Cluster, svcName string) error {
	err := cluster.KnativeClient.ServingV1().Services(s.k8sNameSpace).Delete(context.Background(), svcName, *metav1.NewDeleteOptions(0))
	if err != nil {
		return err
	}
	podNames, _ := s.GetServicePods(c.Request.Context(), *cluster, svcName, s.k8sNameSpace, -1)
	if podNames == nil {
		return nil
	}
	//before k8s 1.31, kill pod does not kill the process immediately, instead we still need wait for the process to exit. more details see: https://github.com/kubernetes/kubernetes/issues/120449
	gracePeriodSeconds := int64(10)
	deletePolicy := metav1.DeletePropagationForeground
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriodSeconds,
		PropagationPolicy:  &deletePolicy,
	}

	for _, podName := range podNames {
		errForce := cluster.Client.CoreV1().Pods(s.k8sNameSpace).Delete(c.Request.Context(), podName, deleteOptions)
		if errForce != nil {
			slog.Error("removeServiceForcely failed to delete pod", slog.String("pod_name", podName), slog.Any("error", errForce))
		}
	}
	return nil
}

func (s *K8sHandler) UpdateService(c *gin.Context) {
	var resp types.ModelUpdateResponse
	var request = &types.ModelUpdateRequest{}
	err := c.BindJSON(request)

	if err != nil {
		slog.Error("updateService get bad request", slog.Any("error", err), slog.Any("req", request))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cluster, err := s.clusterPool.GetClusterByID(c, request.ClusterID)
	if err != nil {
		slog.Error("fail to get cluster ", slog.Any("error", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	srvName := s.getServiceNameFromRequest(c)
	srv, err := cluster.KnativeClient.ServingV1().Services(s.k8sNameSpace).
		Get(c.Request.Context(), srvName, metav1.GetOptions{})
	if err != nil {
		k8serr := new(k8serrors.StatusError)
		if errors.As(err, &k8serr) {
			if k8serr.Status().Code == http.StatusNotFound {
				slog.Info("update service skip,service not exist", slog.String("srv_name", srvName), slog.Any("k8s_err", k8serr))
				resp.Code = 0
				resp.Message = "skip,service not exist"
				c.JSON(http.StatusOK, nil)
				return
			}
		}
		slog.Error("update service failed, cannot get service info", slog.String("srv_name", srvName), slog.Any("error", err),
			slog.String("srv_name", srvName))
		resp.Code = -1
		resp.Message = "failed to get service status"
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	if srv == nil {
		resp.Code = 0
		resp.Message = "service not exist"
		c.JSON(http.StatusOK, resp)
		return
	}
	// Update Image
	containerImg := path.Join(s.modelDockerRegBase, request.ImageID)
	srv.Spec.Template.Spec.Containers[0].Image = containerImg
	// Update env
	environments := []corev1.EnvVar{}
	if request.Env != nil {
		// generate env
		for key, value := range request.Env {
			environments = append(environments, corev1.EnvVar{Name: key, Value: value})
		}
		srv.Spec.Template.Spec.Containers[0].Env = environments
	}
	// Update CPU and Memory requests and limits
	hardware := request.Hardware
	resReq, _ := component.GenerateResources(hardware)
	resources := corev1.ResourceRequirements{
		Limits:   resReq,
		Requests: resReq,
	}
	srv.Spec.Template.Spec.Containers[0].Resources = resources
	// Update replica
	srv.Spec.Template.Annotations["autoscaling.knative.dev/min-scale"] = strconv.Itoa(request.MinReplica)
	srv.Spec.Template.Annotations["autoscaling.knative.dev/max-scale"] = strconv.Itoa(request.MaxReplica)

	_, err = cluster.KnativeClient.ServingV1().Services(s.k8sNameSpace).Update(c, srv, metav1.UpdateOptions{})
	if err != nil {
		slog.Error("failed to update service ", slog.String("srv_name", srvName), slog.Any("error", err),
			slog.String("srv_name", srvName))
		resp.Code = -1
		resp.Message = "failed to update service"
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	slog.Info("service updated", slog.String("srv_name", srvName))
	resp.Code = 0
	resp.Message = "service updated"
	c.JSON(http.StatusOK, resp)
}

func (s *K8sHandler) ServiceStatus(c *gin.Context) {
	var resp types.StatusResponse

	var request = &types.StatusRequest{}
	err := c.BindJSON(request)

	if err != nil {
		slog.Error("serviceStatus get bad request", slog.Any("error", err), slog.Any("req", request))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cluster, err := s.clusterPool.GetClusterByID(c, request.ClusterID)

	if err != nil {
		slog.Error("fail to get cluster ", slog.Any("error", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	srvName := s.getServiceNameFromRequest(c)
	srv, err := cluster.KnativeClient.ServingV1().Services(s.k8sNameSpace).
		Get(c.Request.Context(), srvName, metav1.GetOptions{})
	if err != nil {
		slog.Error("get image status failed, cannot get service info", slog.String("srv_name", srvName), slog.Any("error", err),
			slog.String("srv_name", srvName))
		resp.Code = common.Stopped
		resp.Message = "failed to get service status"
		c.JSON(http.StatusOK, resp)
		return
	}
	deployIDStr := srv.Annotations["deploy_id"]
	deployID, _ := strconv.ParseInt(deployIDStr, 10, 64)
	resp.DeployID = deployID
	resp.UserID = srv.Annotations["user_id"]

	// retrieve pod list and status
	if request.NeedDetails {
		instList, err := s.s.GetServicePodsWithStatus(c.Request.Context(), *cluster, srvName, s.k8sNameSpace)
		if err != nil {
			slog.Error("fail to get service pod name list", slog.Any("error", err))
			c.JSON(http.StatusNotFound, gin.H{"error": "fail to get service pod name list"})
			return
		}
		resp.Instances = instList
	}

	if srv.IsFailed() {
		resp.Code = common.DeployFailed
		// read message of Ready
		resp.Message = srv.Status.GetCondition(v1.ServiceConditionReady).Message
		// append message of ConfigurationsReady
		srvConfigReady := srv.Status.GetCondition(v1.ServiceConditionConfigurationsReady)
		if srvConfigReady != nil {
			resp.Message += srvConfigReady.Message
		}
		// for inference case: model loading case one pod is not ready
		for _, instance := range resp.Instances {
			if instance.Status == string(corev1.PodRunning) || instance.Status == string(corev1.PodPending) {
				resp.Code = common.Deploying
				break
			}
		}
		slog.Info("service status is failed", slog.String("srv_name", srvName), slog.Any("resp", resp))
		c.JSON(http.StatusOK, resp)
		return
	}

	if srv.IsReady() {
		podNames, err := s.GetServicePods(c.Request.Context(), *cluster, srvName, s.k8sNameSpace, 1)
		if err != nil {
			slog.Error("get image status failed, can not get pods info", slog.String("srv_name", srvName), slog.Any("error", err))
			c.JSON(http.StatusInternalServerError, gin.H{"code": 0, "message": "unknown service status, failed to get pods"})
			return
		}
		if len(podNames) == 0 {
			resp.Code = common.Sleeping
			resp.Message = "service sleeping, no running pods"
			slog.Debug("get image status success", slog.String("srv_name", srvName), slog.Any("resp", resp))
			c.JSON(http.StatusOK, resp)
			return
		}

		resp.Code = common.Running
		resp.Message = "service running"
		if srv.Status.URL != nil {
			slog.Debug("knative endpoint", slog.Any("svc name", srvName), slog.Any("url", srv.Status.URL.URL().String()))
			resp.Endpoint = srv.Status.URL.URL().String()
		}

		slog.Debug("service status is ready", slog.String("srv_name", srvName), slog.Any("resp", resp))
		c.JSON(http.StatusOK, resp)
		return
	}

	// default to deploying status
	resp.Code = common.Deploying
	resp.Message = "service is not ready or failed"
	slog.Info("get service status success, service is not ready or failed", slog.String("srv_name", srvName), slog.Any("resp", resp))
	c.JSON(http.StatusOK, resp)
}

func (s *K8sHandler) ServiceLogs(c *gin.Context) {
	var request = &types.LogsRequest{}
	err := c.BindJSON(request)

	if err != nil {
		slog.Error("serviceLogs get bad request", slog.Any("error", err), slog.Any("req", request))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cluster, err := s.clusterPool.GetClusterByID(c, request.ClusterID)
	if err != nil {
		slog.Error("fail to get cluster ", slog.Any("error", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	srvName := s.getServiceNameFromRequest(c)
	podNames, err := s.GetServicePods(c.Request.Context(), *cluster, srvName, s.k8sNameSpace, 1)
	if err != nil {
		slog.Error("failed to read image logs, cannot get pods info", slog.Any("error", err), slog.String("srv_name", srvName))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get pods info"})
		return
	}
	if len(podNames) == 0 {
		slog.Error("failed to read image logs, no running pods", slog.String("srv_name", srvName))
		c.JSON(http.StatusNotFound, gin.H{"error": "no running pods, service maybe sleeping"})
		return
	}
	s.GetLogsByPod(c, *cluster, podNames[0], srvName)
}

func (s *K8sHandler) ServiceLogsByPod(c *gin.Context) {
	var request = &types.ServiceRequest{}
	err := c.BindJSON(request)

	if err != nil {
		slog.Error("serviceLogs get bad request", slog.Any("error", err), slog.Any("req", request))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cluster, err := s.clusterPool.GetClusterByID(c, request.ClusterID)
	if err != nil {
		slog.Error("fail to get cluster ", slog.Any("error", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	srvName := s.getServiceNameFromRequest(c)
	podName := s.getPodNameFromRequest(c)
	s.GetLogsByPod(c, *cluster, podName, srvName)
}

func (s *K8sHandler) GetLogsByPod(c *gin.Context, cluster cluster.Cluster, podName string, srvName string) {

	logs := cluster.Client.CoreV1().Pods(s.k8sNameSpace).GetLogs(podName, &corev1.PodLogOptions{
		Container: "user-container",
		Follow:    true,
	})
	stream, err := logs.Stream(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open stream"})
		return
	}
	defer stream.Close()

	// c.Header("Content-Type", "text/event-stream")
	c.Header("Content-Type", "text/plain")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Writer.WriteHeader(http.StatusOK)
	buf := make([]byte, 32*1024)

	pod, err := cluster.Client.CoreV1().Pods(s.k8sNameSpace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		slog.Error("fail to get pod ", slog.Any("error", err), slog.String("pod name", podName))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if pod.Status.Phase == "Pending" {
		for _, condition := range pod.Status.Conditions {
			if condition.Type == "PodScheduled" && condition.Status == "False" {
				message := fmt.Sprintf("Pod is pending due to reason: %s, message: %s", condition.Reason, condition.Message)
				_, err = c.Writer.Write([]byte(message))
				if err != nil {
					slog.Error("write data failed", "error", err)
				}
				c.Writer.Flush()
				c.JSON(http.StatusBadRequest, gin.H{"error": message})
				return
			}
		}
	}

	for {
		select {
		case <-c.Request.Context().Done():
			slog.Info("logs request context done", slog.Any("error", c.Request.Context().Err()))
			return
		default:
			n, err := stream.Read(buf)
			if err != nil {
				slog.Error("read pod logs failed", slog.Any("error", err), slog.String("srv_name", srvName))
				break
			}
			if n == 0 {
				time.Sleep(5 * time.Second)
			}

			if n > 0 {
				_, err = c.Writer.Write(buf[:n])
				if err != nil {
					slog.Error("write data failed", "error", err)
				}
				c.Writer.Flush()
				slog.Info("send pod logs", slog.String("srv_name", srvName), slog.String("srv_name", srvName), slog.Int("len", n))
			}
		}

	}
}

func (s *K8sHandler) ServiceStatusAll(c *gin.Context) {
	allStatus := make(map[string]*types.StatusResponse)
	for index := range s.clusterPool.Clusters {
		cluster := s.clusterPool.Clusters[index]
		services, err := cluster.KnativeClient.ServingV1().Services(s.k8sNameSpace).
			List(c.Request.Context(), metav1.ListOptions{})
		if err != nil {
			slog.Error("get image status all failed, cannot get service infos", slog.Any("error", err))
			//continue to next in multi cluster
			continue
		}

		for _, srv := range services.Items {
			deployIDStr := srv.Annotations[component.KeyDeployID]
			deployID, _ := strconv.ParseInt(deployIDStr, 10, 64)
			deployTypeStr := srv.Annotations[component.KeyDeployType]
			deployType, err := strconv.ParseInt(deployTypeStr, 10, 64)
			if err != nil {
				deployType = 0
			}
			userID := srv.Annotations[component.KeyUserID]
			deploySku := srv.Annotations[component.KeyDeploySKU]
			status := &types.StatusResponse{
				DeployID:    deployID,
				UserID:      userID,
				DeployType:  int(deployType),
				ServiceName: srv.Name,
				DeploySku:   deploySku,
			}
			allStatus[srv.Name] = status
			if srv.IsFailed() {
				status.Code = common.DeployFailed
				continue
			}

			if srv.IsReady() {
				podNames, err := s.GetServicePods(c.Request.Context(), cluster, srv.Name, s.k8sNameSpace, 1)
				if err != nil {
					slog.Error("get image status failed, cannot get pods info", slog.Any("error", err))
					status.Code = common.Running
					continue
				}
				status.Replica = len(podNames)
				if len(podNames) == 0 {
					status.Code = common.Sleeping
					continue
				}

				status.Code = common.Running
				continue
			}

			// default to deploying
			status.Code = common.Deploying
		}
	}

	c.JSON(http.StatusOK, allStatus)
}

func (s *K8sHandler) GetServicePods(ctx context.Context, cluster cluster.Cluster, srvName string, namespace string, limit int64) ([]string, error) {
	labelSelector := fmt.Sprintf("serving.knative.dev/service=%s", srvName)
	// Get the list of Pods based on the label selector
	opts := metav1.ListOptions{
		LabelSelector: labelSelector,
	}
	if limit > 0 {
		opts = metav1.ListOptions{
			LabelSelector: labelSelector,
			Limit:         limit,
		}
	}
	pods, err := cluster.Client.CoreV1().Pods(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Extract the Pod names
	var podNames []string
	for _, pod := range pods.Items {
		podNames = append(podNames, pod.Name)
	}

	return podNames, nil
}

func (s *K8sHandler) GetClusterInfo(c *gin.Context) {
	clusterRes := []types.CluserResponse{}
	for index := range s.clusterPool.Clusters {
		cls := s.clusterPool.Clusters[index]
		cInfo, err := s.clusterPool.ClusterStore.ByClusterConfig(c.Request.Context(), cls.ID)
		if err != nil {
			slog.Error("get cluster info failed", slog.Any("error", err))
			continue
		}
		clusterInfo := types.CluserResponse{}
		clusterInfo.Region = cInfo.Region
		clusterInfo.Zone = cInfo.Zone
		clusterInfo.Provider = cInfo.Provider
		clusterInfo.ClusterID = cInfo.ClusterID
		clusterInfo.ClusterName = fmt.Sprintf("cluster%d", index)
		clusterRes = append(clusterRes, clusterInfo)

	}
	c.JSON(http.StatusOK, clusterRes)
}

func (s *K8sHandler) GetClusterInfoByID(c *gin.Context) {
	clusterId := c.Params.ByName("id")
	cInfo, _ := s.clusterPool.ClusterStore.ByClusterID(c.Request.Context(), clusterId)
	clusterInfo := types.CluserResponse{}
	clusterInfo.Region = cInfo.Region
	clusterInfo.Zone = cInfo.Zone
	clusterInfo.Provider = cInfo.Provider
	clusterInfo.ClusterID = cInfo.ClusterID
	clusterInfo.StorageClass = cInfo.StorageClass
	client, err := s.clusterPool.GetClusterByID(c.Request.Context(), clusterId)
	if err != nil {
		slog.Error("fail to get cluster", slog.Any("error", err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	nodes, err := cluster.GetNodeResources(client.Client, s.env)
	if err == nil {
		clusterInfo.Nodes = nodes
	}

	c.JSON(http.StatusOK, clusterInfo)
}

func (s *K8sHandler) getServiceNameFromRequest(c *gin.Context) string {
	return c.Params.ByName("service")
}

func (s *K8sHandler) getPodNameFromRequest(c *gin.Context) string {
	return c.Params.ByName("pod_name")
}

func (s *K8sHandler) GetServiceByName(c *gin.Context) {
	var resp types.StatusResponse
	var request = &types.CheckRequest{}
	err := c.BindJSON(request)
	if err != nil {
		slog.Error("fail to parse input parameters", slog.Any("error", err))
		resp.Code = -1
		resp.Message = "fail to parse input parameters"
		c.JSON(http.StatusOK, resp)
		return
	}
	cluster, err := s.clusterPool.GetClusterByID(c, request.ClusterID)
	if err != nil {
		slog.Error("fail to get cluster config", slog.Any("error", err))
		resp.Code = -1
		resp.Message = "fail to get cluster config"
		c.JSON(http.StatusOK, resp)
		return
	}
	srvName := s.getServiceNameFromRequest(c)
	srv, err := cluster.KnativeClient.ServingV1().Services(s.k8sNameSpace).Get(c.Request.Context(), srvName, metav1.GetOptions{})
	if err != nil {
		k8serr := new(k8serrors.StatusError)
		if errors.As(err, &k8serr) {
			if k8serr.Status().Code == http.StatusNotFound {
				// service not exist
				resp.Code = 0
				resp.Message = "service not exist"
				c.JSON(http.StatusOK, resp)
				return
			}
		}
		// get service with error
		slog.Error("fail to get service with error", slog.Any("error", err))
		resp.Code = -1
		resp.Message = "fail to get service"
		c.JSON(http.StatusOK, resp)
		return
	}

	if srv == nil {
		// service not exist
		resp.Code = 0
		resp.Message = "service not exist"
		c.JSON(http.StatusOK, resp)
		return
	}

	// service exist
	deployIDStr := srv.Annotations[types.ResDeployID]
	deployID, _ := strconv.ParseInt(deployIDStr, 10, 64)
	resp.DeployID = deployID
	resp.Code = 1
	resp.Message = srvName
	if srv.Status.URL != nil {
		resp.Endpoint = srv.Status.URL.URL().String()
	}
	c.JSON(http.StatusOK, resp)
}

func (s *K8sHandler) GetReplica(c *gin.Context) {
	var resp types.ReplicaResponse
	var request = &types.StatusRequest{}
	err := c.BindJSON(request)
	if err != nil {
		slog.Error("fail to parse input parameters", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fail to parse input parameters"})
		return
	}
	cluster, err := s.clusterPool.GetClusterByID(c, request.ClusterID)
	if err != nil {
		slog.Error("fail to get cluster config", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fail to get cluster config"})
		return
	}
	srvName := s.getServiceNameFromRequest(c)
	srv, err := cluster.KnativeClient.ServingV1().Services(s.k8sNameSpace).Get(c.Request.Context(), srvName, metav1.GetOptions{})
	if err != nil {
		// get service with error
		slog.Error("fail to get service", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fail to get service"})
		return
	}

	if srv == nil {
		// service not exist
		slog.Error("service not exist")
		c.JSON(http.StatusNotFound, gin.H{"error": "service not exist"})
		return
	}
	// revisionName := srv.Status.LatestReadyRevisionName
	revisionName := srv.Status.LatestCreatedRevisionName
	if len(revisionName) < 1 {
		slog.Error("fail to get latest created revision")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fail to get latest created revision"})
		return
	}
	revision, err := cluster.KnativeClient.ServingV1().Revisions(s.k8sNameSpace).Get(c.Request.Context(), revisionName, metav1.GetOptions{})
	if err != nil {
		slog.Error("fail to get revision with error", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fail to get revision with error"})
		return
	}

	if revision == nil {
		slog.Error("revision not exist")
		c.JSON(http.StatusNotFound, gin.H{"error": "revision not exist"})
		return
	}
	instList, err := s.s.GetServicePodsWithStatus(c.Request.Context(), *cluster, srvName, s.k8sNameSpace)
	if err != nil {
		slog.Error("fail to get service pod name list", slog.Any("error", err))
		c.JSON(http.StatusNotFound, gin.H{"error": "fail to get service pod name list"})
		return
	}

	// revision exist
	resp.Code = 1
	resp.Message = srvName
	resp.ActualReplica = int(*revision.Status.ActualReplicas)
	resp.DesiredReplica = int(*revision.Status.DesiredReplicas)
	resp.Instances = instList
	c.JSON(http.StatusOK, resp)
}

func (s *K8sHandler) UpdateCluster(c *gin.Context) {
	var resp types.UpdateClusterResponse
	var request = &database.ClusterInfo{}
	err := c.BindJSON(request)
	if err != nil {
		slog.Error("fail to parse input parameters", slog.Any("error", err))
		resp.Code = -1
		resp.Message = "fail to parse input parameters"
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	err = s.clusterPool.ClusterStore.Update(c, *request)
	if err != nil {
		slog.Error("fail to update cluster", slog.Any("error", err))
		resp.Code = -1
		resp.Message = "fail to update cluster"
		c.JSON(http.StatusInternalServerError, resp)
		return
	}
	resp.Code = 0
	resp.Message = "succeed to update cluster"
	c.JSON(http.StatusOK, resp)
}

func (s *K8sHandler) PurgeService(c *gin.Context) {
	var resp types.PurgeResponse
	var request = &types.PurgeRequest{}
	err := c.BindJSON(request)
	if err != nil {
		slog.Error("fail to parse input parameters", slog.Any("error", err))
		resp.Code = -1
		resp.Message = "fail to parse cluster id"
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	cluster, err := s.clusterPool.GetClusterByID(c, request.ClusterID)
	if err != nil {
		slog.Error("fail to get cluster config", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fail to get cluster config"})
		return
	}
	srvName := s.getServiceNameFromRequest(c)
	_, err = cluster.KnativeClient.ServingV1().Services(s.k8sNameSpace).
		Get(c.Request.Context(), srvName, metav1.GetOptions{})
	if err != nil {
		k8serr := new(k8serrors.StatusError)
		if errors.As(err, &k8serr) {
			if k8serr.Status().Code == http.StatusNotFound {
				slog.Info("service not exist", slog.String("srv_name", srvName), slog.Any("k8s_err", k8serr))
			}
		}
		slog.Error("purge service failed, cannot get service info", slog.String("srv_name", srvName), slog.Any("error", err),
			slog.String("srv_name", srvName))
	} else {
		// 1 delete service
		err = s.removeServiceForcely(c, cluster, srvName)
		if err != nil {
			slog.Error("failed to delete service ", slog.String("srv_name", srvName), slog.Any("error", err),
				slog.String("srv_name", srvName))
			resp.Code = -1
			resp.Message = "failed to get service status"
			c.JSON(http.StatusInternalServerError, resp)
			return
		}
	}

	// 2 clean up pvc
	if cluster.StorageClass != "" && request.DeployType == types.FinetuneType {
		err = cluster.Client.CoreV1().PersistentVolumeClaims(s.k8sNameSpace).Delete(c, srvName, metav1.DeleteOptions{})
		if err != nil {
			slog.Error("fail to delete pvc", slog.Any("error", err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "fail to delete pvc"})
			return
		}
		slog.Info("persistent volume claims deleted.", slog.String("srv_name", srvName))
	}
	slog.Info("service deleted.", slog.String("srv_name", srvName))
	resp.Code = 0
	resp.Message = "succeed to clean up service"
	c.JSON(http.StatusOK, resp)
}

func (s *K8sHandler) GetServiceInfo(c *gin.Context) {
	var resp types.ServiceInfoResponse
	var request = &types.ServiceRequest{}
	err := c.BindJSON(request)
	if err != nil {
		slog.Error("fail to parse input parameters", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fail to parse input parameters"})
		return
	}
	cluster, err := s.clusterPool.GetClusterByID(c, request.ClusterID)
	if err != nil {
		slog.Error("fail to get cluster config", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fail to get cluster config"})
		return
	}

	srvName := s.getServiceNameFromRequest(c)
	podNames, err := s.GetServicePods(c.Request.Context(), *cluster, srvName, s.k8sNameSpace, -1)
	if err != nil {
		slog.Error("failed to read image logs, cannot get pods info", slog.Any("error", err), slog.String("srv_name", srvName))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get pods info"})
		return
	}
	resp.PodNames = podNames
	resp.ServiceName = srvName
	c.JSON(http.StatusOK, resp)
}
