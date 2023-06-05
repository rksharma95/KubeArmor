package controllers

import (
	"context"
	"strings"
	"sync"
	"time"

	deployments "github.com/kubearmor/KubeArmor/deployments/get"
	opv1 "github.com/kubearmor/KubeArmor/pkg/KubeArmorOperator/api/operator.kubearmor.com/v1"
	opv1client "github.com/kubearmor/KubeArmor/pkg/KubeArmorOperator/client/clientset/versioned"
	opv1Informer "github.com/kubearmor/KubeArmor/pkg/KubeArmorOperator/client/informers/externalversions"
	"github.com/kubearmor/KubeArmor/pkg/KubeArmorOperator/common"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var informer informers.SharedInformerFactory
var deployment_uuid types.UID
var deployment_name string = "kubearmor-operator"
var PathPrefix string

type ClusterWatcher struct {
	Nodes          []Node
	NodesLock      *sync.Mutex
	Log            *zap.SugaredLogger
	Client         *kubernetes.Clientset
	ExtClient      *apiextensionsclientset.Clientset
	Opv1Client     *opv1client.Clientset
	Daemonsets     map[string]int
	DaemonsetsLock *sync.Mutex
}
type Node struct {
	Name           string
	Enforcer       string
	Runtime        string
	RuntimeSocket  string
	RuntimeStorage string
	Arch           string
}

func NewClusterWatcher(client *kubernetes.Clientset, log *zap.SugaredLogger, extClient *apiextensionsclientset.Clientset, opv1Client *opv1client.Clientset, pathPrefix, deploy_name string) *ClusterWatcher {
	if informer == nil {
		informer = informers.NewSharedInformerFactory(client, 0)
	}
	if deployment_uuid == "" {
		deploy, err := client.AppsV1().Deployments(common.Namespace).Get(context.Background(), deployment_name, v1.GetOptions{})
		if err != nil {
			log.Warnf("Cannot get deployment %s, error=%s", deployment_name, err.Error())
		} else {
			deployment_uuid = deploy.GetUID()
		}
	}
	PathPrefix = pathPrefix
	deployment_name = deploy_name
	return &ClusterWatcher{
		Nodes:          []Node{},
		Daemonsets:     make(map[string]int),
		Log:            log,
		NodesLock:      &sync.Mutex{},
		DaemonsetsLock: &sync.Mutex{},
		Client:         client,
		ExtClient:      extClient,
		Opv1Client:     opv1Client,
	}
}

func (clusterWatcher *ClusterWatcher) WatchNodes() {
	log := clusterWatcher.Log
	nodeInformer := informer.Core().V1().Nodes().Informer()
	nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if node, ok := obj.(*corev1.Node); ok {
				runtime := node.Status.NodeInfo.ContainerRuntimeVersion
				runtime = strings.Split(runtime, ":")[0]
				if val, ok := node.Labels[common.OsLabel]; ok && val == "linux" {
					log.Infof("Installing snitch on node %s", node.Name)
					_, err := clusterWatcher.Client.BatchV1().Jobs(common.Namespace).Create(context.Background(), deploySnitch(node.Name, runtime), v1.CreateOptions{})
					if err != nil {
						log.Errorf("Cannot run snitch on node %s, error=%s", node.Name, err.Error())
						return
					}
					log.Infof("Snitch was installed on node %s", node.Name)
				}
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if node, ok := newObj.(*corev1.Node); ok {
				oldRand := ""
				if old, ok := oldObj.(*corev1.Node); ok {
					oldRand = old.Labels[common.RandLabel]
				}
				if val, ok := node.Labels[common.OsLabel]; ok && val == "linux" && oldRand != node.Labels[common.RandLabel] {
					newNode := Node{}
					if val, ok := node.Labels[common.EnforcerLabel]; ok {
						newNode.Enforcer = val
					}
					if val, ok := node.Labels[common.ArchLabel]; ok {
						newNode.Arch = val
					}
					if val, ok := node.Labels[common.RuntimeLabel]; ok {
						newNode.Runtime = val
					}
					if val, ok := node.Labels[common.SocketLabel]; ok {
						newNode.RuntimeSocket = val
					}
					if val, ok := node.Labels[common.RuntimeStorageLabel]; ok {
						newNode.RuntimeStorage = val
					}

					clusterWatcher.NodesLock.Lock()
					nbNodes := len(clusterWatcher.Nodes)
					i := 0
					nodeModified := false
					for i < nbNodes && newNode.Name != clusterWatcher.Nodes[i].Name {
						i++
					}
					if i == len(clusterWatcher.Nodes) {
						clusterWatcher.Nodes = append(clusterWatcher.Nodes, newNode)
					} else {
						if clusterWatcher.Nodes[i].Arch != newNode.Arch ||
							clusterWatcher.Nodes[i].Enforcer != newNode.Enforcer ||
							clusterWatcher.Nodes[i].Name != newNode.Name ||
							clusterWatcher.Nodes[i].Runtime != newNode.Runtime ||
							clusterWatcher.Nodes[i].RuntimeSocket != newNode.RuntimeSocket ||
							clusterWatcher.Nodes[i].RuntimeStorage != newNode.RuntimeStorage {
							clusterWatcher.Nodes[i] = newNode
							nodeModified = true
							clusterWatcher.Log.Infof("Node %s was updated", node.Name)
						}
					}
					clusterWatcher.NodesLock.Unlock()
					if nodeModified {
						clusterWatcher.UpdateDaemonsets(common.DeletAction, newNode.Enforcer, newNode.Runtime, newNode.RuntimeSocket, newNode.RuntimeStorage)
					}
					clusterWatcher.UpdateDaemonsets(common.AddAction, newNode.Enforcer, newNode.Runtime, newNode.RuntimeSocket, newNode.RuntimeStorage)
				}
			} else {
				log.Errorf("Cannot convert object to node struct")
				log.Error(newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if node, ok := obj.(*corev1.Node); ok {
				deletedNode := Node{}
				clusterWatcher.NodesLock.Lock()
				for i, n := range clusterWatcher.Nodes {
					if n.Name == node.Name {
						clusterWatcher.Nodes = append(clusterWatcher.Nodes[:i], clusterWatcher.Nodes[i+1:]...)
						deletedNode = n
						break
					}
				}
				clusterWatcher.NodesLock.Unlock()
				clusterWatcher.UpdateDaemonsets(common.DeletAction, deletedNode.Enforcer, deletedNode.Runtime, deletedNode.RuntimeSocket, deletedNode.RuntimeStorage)
			}
		},
	})

	nodeInformer.Run(wait.NeverStop)
}

func (clusterWatcher *ClusterWatcher) UpdateDaemonsets(action, enforcer, runtime, socket, runtimeStorage string) {
	clusterWatcher.Log.Info("updating daemonset")
	daemonsetName := strings.Join([]string{
		"kubearmor",
		strings.ReplaceAll(enforcer, ".", "-"),
		runtime,
		common.ShortSHA(socket),
	}, "-")
	newDaemonSet := false
	deleteDaemonSet := false
	clusterWatcher.DaemonsetsLock.Lock()
	if action == common.AddAction {
		clusterWatcher.Daemonsets[daemonsetName]++
		_, err := clusterWatcher.Client.AppsV1().DaemonSets(common.Namespace).Get(context.Background(), daemonsetName, v1.GetOptions{})
		if err != nil {
			newDaemonSet = true
		}
	} else if action == common.DeletAction {
		if val, ok := clusterWatcher.Daemonsets[daemonsetName]; ok {
			if val < 2 {
				clusterWatcher.Daemonsets[daemonsetName] = 0
				deleteDaemonSet = true
			} else {
				clusterWatcher.Daemonsets[daemonsetName]--
			}
		}
	}
	clusterWatcher.DaemonsetsLock.Unlock()

	if deleteDaemonSet {
		err := clusterWatcher.Client.AppsV1().DaemonSets(common.Namespace).Delete(context.Background(), daemonsetName, v1.DeleteOptions{})
		if err != nil {
			clusterWatcher.Log.Warnf("Cannot delete daemonset %s, error=%s", daemonsetName, err.Error())
		}
	}
	if newDaemonSet {
		daemonset := generateDaemonset(daemonsetName, enforcer, runtime, socket, runtimeStorage)
		_, err := clusterWatcher.Client.AppsV1().DaemonSets(common.Namespace).Create(context.Background(), daemonset, v1.CreateOptions{})
		if err != nil {
			clusterWatcher.Log.Warnf("Cannot Create daemonset %s, error=%s", daemonsetName, err.Error())
		}
	}

}

func (clusterWatcher *ClusterWatcher) WatchConfigCrd() {

	factory := opv1Informer.NewSharedInformerFactoryWithOptions(clusterWatcher.Opv1Client,
		time.Duration(5*time.Second),
		opv1Informer.WithNamespace(common.Namespace))

	informer := factory.Operator().V1().Configs().Informer()

	if informer == nil {
		clusterWatcher.Log.Warn("Failed to initialize Config informer")
		return
	}

	var firstRun = true

	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				configCrdList, err := clusterWatcher.Opv1Client.OperatorV1().Configs(common.Namespace).List(context.Background(), metav1.ListOptions{})
				if err != nil {
					clusterWatcher.Log.Warn("Failed to list Operator Config CRDs")
					return
				}
				for _, cfg := range configCrdList.Items {
					// if there's any crd with Running status
					// mark it as current operating config crd
					if cfg.Status.Phase == common.RUNNING {
						clusterWatcher.Log.Infof("%s is operatingcrd", cfg.Name)
						common.OperatigConfigCrd = &cfg
						if firstRun {
							clusterWatcher.Log.Info("1. it is first run")
							go clusterWatcher.WatchRequiredResources()
							firstRun = false
						}
						break
					}
				}
				if cfg, ok := obj.(*opv1.Config); ok {
					// if there's no operating crd exist
					if common.OperatigConfigCrd == nil {
						clusterWatcher.Log.Info("currently there's no operating crd")
						common.OperatigConfigCrd = cfg
						clusterWatcher.Log.Infof("%s is now set as operating crd", cfg.Name)
						UpdateConfigMapData(&cfg.Spec)
						// update status to (Installation) Created
						go clusterWatcher.UpdateCrdStatus(cfg.Name, common.CREATED, common.CREATED_MSG)
						go clusterWatcher.WatchRequiredResources()
						firstRun = false
					}
					// if it's not the operating crd
					// update this crd status as Error and return
					if cfg.Name != common.OperatigConfigCrd.Name {
						clusterWatcher.Log.Infof("%s is invalid there's already an operating crd exists", cfg.Name)
						go clusterWatcher.UpdateCrdStatus(cfg.Name, common.ERROR, common.MULTIPLE_CRD_ERR_MSG)
						return
					}

				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				if cfg, ok := newObj.(*opv1.Config); ok {
					// update configmap if it's operating crd
					if common.OperatigConfigCrd != nil && cfg.Name == common.OperatigConfigCrd.Name {
						configChanged := UpdateConfigMapData(&cfg.Spec)
						if configChanged {
							clusterWatcher.Log.Infof("config changed to %s", cfg.Spec)
						}
						if !configChanged && cfg.Status != oldObj.(*opv1.Config).Status {
							clusterWatcher.Log.Infof("config not changed only status has been updated from %s to %s",
								oldObj.(*opv1.Config).Status, cfg.Status)
							return
						}
						if configChanged {
							clusterWatcher.Log.Info("operating crds config changed")
							// update status to Updating
							go clusterWatcher.UpdateCrdStatus(cfg.Name, common.UPDATING, common.UPDATING_MSG)
							clusterWatcher.UpdateKubeArmorConfigMap(cfg)
						}
					}
				}
			},
			DeleteFunc: func(obj interface{}) {
				if cfg, ok := obj.(*opv1.Config); ok {
					if common.OperatigConfigCrd != nil && cfg.Name == common.OperatigConfigCrd.Name {
						clusterWatcher.Log.Info("operating crd gets deleted")
						common.OperatigConfigCrd = nil
					}
				}
			},
		},
	)

	go informer.Run(wait.NeverStop)

	if ok := cache.WaitForCacheSync(wait.NeverStop, informer.HasSynced); !ok {
		clusterWatcher.Log.Warn("Failed to wait for cache sync")
	}
}

func (clusterWatcher *ClusterWatcher) UpdateCrdStatus(cfg, phase, message string) {
	err := wait.ExponentialBackoff(wait.Backoff{Steps: 5, Duration: 500 * time.Millisecond}, func() (bool, error) {
		configCrd, err := clusterWatcher.Opv1Client.OperatorV1().Configs(common.Namespace).Get(context.Background(), cfg, metav1.GetOptions{})
		if err != nil {
			clusterWatcher.Log.Warnf("1. error getting the crd %s\n", err)
			// retry the update
			return false, nil
		}
		// update status only if there's any change
		newStatus := opv1.ConfigStatus{
			Phase:   phase,
			Message: message,
		}
		if configCrd.Status != newStatus {
			clusterWatcher.Log.Infof("Current status %s changed to %s\n", configCrd.Status, newStatus)
			configCrd.Status = newStatus
			_, err = clusterWatcher.Opv1Client.OperatorV1().Configs(common.Namespace).UpdateStatus(context.Background(), configCrd, metav1.UpdateOptions{})
			if err != nil {
				// retry the update
				clusterWatcher.Log.Warnf("2. error updating the status %s\n", err)
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil {
		clusterWatcher.Log.Errorf("Error updating the ConfigCRD status %s", err)
		return
	}
	clusterWatcher.Log.Info("Config CRD Status Updated Successfully")
}

func (clusterWatcher *ClusterWatcher) UpdateKubeArmorConfigMap(cfg *opv1.Config) {
	err := wait.ExponentialBackoff(wait.Backoff{Steps: 5, Duration: 500 * time.Millisecond}, func() (bool, error) {
		cm, err := clusterWatcher.Client.CoreV1().ConfigMaps(common.Namespace).Get(context.Background(), deployments.KubeArmorConfigMapName, metav1.GetOptions{})
		if err != nil {
			if isNotfound(err) {
				return true, nil
			}
			// retry the update
			return false, nil
		}
		cm.Data = common.ConfigMapData
		_, err = clusterWatcher.Client.CoreV1().ConfigMaps(common.Namespace).Update(context.Background(), cm, metav1.UpdateOptions{})
		if err != nil {
			// retry the update
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		clusterWatcher.Log.Errorf("Error updating the Config %s", err)
		go clusterWatcher.UpdateCrdStatus(cfg.Name, common.ERROR, common.UPDATION_FAILED_ERR_MSG)
		return
	}
	go clusterWatcher.UpdateCrdStatus(cfg.Name, common.RUNNING, common.RUNNING_MSG)
	clusterWatcher.Log.Info("KubeArmor Config Updated Successfully")
}

func UpdateConfigMapData(config *opv1.ConfigSpec) bool {
	updated := false
	if config.DefaultFilePosture != "" {
		if common.ConfigMapData[common.ConfigDefaultFilePosture] != string(config.DefaultFilePosture) {
			common.ConfigMapData[common.ConfigDefaultFilePosture] = string(config.DefaultFilePosture)
			updated = true
		}
	}
	if config.DefaultCapabilitiesPosture != "" {
		if common.ConfigMapData[common.ConfigDefaultCapabilitiesPosture] != string(config.DefaultCapabilitiesPosture) {
			common.ConfigMapData[common.ConfigDefaultCapabilitiesPosture] = string(config.DefaultCapabilitiesPosture)
			updated = true
		}
	}
	if config.DefaultNetworkPosture != "" {
		if common.ConfigMapData[common.ConfigDefaultNetworkPosture] != string(config.DefaultNetworkPosture) {
			common.ConfigMapData[common.ConfigDefaultNetworkPosture] = string(config.DefaultNetworkPosture)
			updated = true
		}
	}
	if config.DefaultVisibility != "" {
		if common.ConfigMapData[common.ConfigVisibility] != string(config.DefaultVisibility) {
			common.ConfigMapData[common.ConfigVisibility] = string(config.DefaultVisibility)
			updated = true
		}
	}
	return updated
}
