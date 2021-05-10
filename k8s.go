package k8s_api

import (
	"context"
	"encoding/json"
	"scp-ctrl/common/nf_profile"
	"scp-ctrl/common/util"
	"strconv"

	//"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	//"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/util/retry"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/client-go/rest"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

type Obj_K8sCmd struct {
	Obj_client_set  *kubernetes.Clientset
	Obj_mclient_set *metrics.Clientset
}

func Set_K8sCmdInstance(configBytes []byte) *Obj_K8sCmd {
	obj_instance := new(Obj_K8sCmd)

	obj_cluster_cfg, obj_ret := clientcmd.RESTConfigFromKubeConfig(configBytes)
	if obj_ret != nil {
		util.LOG_ERR("[WATCH_K8S] RESTConfigFromKubeConfig() %s failed", obj_ret.Error())
		return nil
	}
	util.LOG_SVC("[WATCH_K8S] JOIN KUBERNETES SERVICE[%s]", obj_cluster_cfg.Host)

	// creates the clientset
	obj_instance.Obj_client_set, obj_ret = kubernetes.NewForConfig(obj_cluster_cfg)
	if obj_ret != nil {
		util.LOG_ERR("[WATCH_K8S] kubernetes.NewForConfig() %s failed\n%s", obj_ret.Error(), util.Conv_struct_to_json_string(obj_cluster_cfg))
		return nil
	}

	// creates the metrics clientset
	obj_instance.Obj_mclient_set, obj_ret = metrics.NewForConfig(obj_cluster_cfg)
	if obj_ret != nil {
		util.LOG_ERR("[WATCH_K8S] kubernetes.NewForConfig() %s failed\n%s", obj_ret.Error(), util.Conv_struct_to_json_string(obj_cluster_cfg))
		return nil
	}

	return obj_instance
}

func Set_K8sCmdInstance_config(obj_cluster_cfg *rest.Config) *Obj_K8sCmd {
	var obj_ret error = nil

	obj_instance := new(Obj_K8sCmd)

	// creates the clientset
	obj_instance.Obj_client_set, obj_ret = kubernetes.NewForConfig(obj_cluster_cfg)
	if obj_ret != nil {
		util.LOG_ERR("[WATCH_K8S] kubernetes.NewForConfig() %s failed\n%s", obj_ret.Error(), util.Conv_struct_to_json_string(obj_cluster_cfg))
		return nil
	}

	// creates the metrics clientset
	obj_instance.Obj_mclient_set, obj_ret = metrics.NewForConfig(obj_cluster_cfg)
	if obj_ret != nil {
		util.LOG_ERR("[WATCH_K8S] kubernetes.NewForConfig() %s failed\n%s", obj_ret.Error(), util.Conv_struct_to_json_string(obj_cluster_cfg))
		return nil
	}

	return obj_instance
}

func (obj_k8s_cmd *Obj_K8sCmd) Get_pod_rsc(namespace string, node_name string, pod_name string) *Pod_rsc_info {
	var total_memory float64 = 0
	var total_cpu float64 = 0

	node_client := obj_k8s_cmd.Obj_client_set.CoreV1().Nodes()
	result, err := node_client.Get(context.TODO(), node_name, metav1.GetOptions{})
	if err != nil {
		util.LOG_ERR("Fail node_client.Get(): %v", err)
		return nil
	}

	node_cpu, ok := result.Status.Capacity.Cpu().AsInt64()
	if !ok {
		util.LOG_ERR("Fail result.Status.Capacity.Cpu().AsInt64()")
		return nil
	}

	node_memory, ok := result.Status.Capacity.Memory().AsInt64()
	if !ok {
		util.LOG_ERR("Fail result.Status.Capacity.Memory().AsInt64()")
		return nil
	}

	//util.LOG_ERR("NODE CPU %d", node_cpu)
	//util.LOG_ERR("NODE MEMORY %d", node_memory)

	podMetric, err := obj_k8s_cmd.Obj_mclient_set.MetricsV1beta1().PodMetricses(namespace).Get(context.TODO(), pod_name, metav1.GetOptions{})
	if err != nil {
		util.LOG_ERR("Fail Get PodMetricses :%s, %s, %s", namespace, pod_name, err)
		return nil
	}

	podContainers := podMetric.Containers

	for _, container := range podContainers {
		cpuQuantity, ok := container.Usage.Cpu().AsInt64()
		if !ok {
			cpuQuantity := container.Usage.Cpu().AsDec()
			c_cpu, err := strconv.ParseFloat(cpuQuantity.String(), 64)
			if err == nil {
				total_cpu += c_cpu
			}
		} else {
			total_cpu += float64(cpuQuantity)
		}

		memQuantity, ok := container.Usage.Memory().AsInt64()
		if !ok {
			memQuantity := container.Usage.Memory().AsDec()
			c_mem, err := strconv.ParseFloat(memQuantity.String(), 64)
			if err == nil {
				total_memory += c_mem
			}
		} else {
			total_memory += float64(memQuantity)
		}
	}

	var rsc_info Pod_rsc_info

	if total_cpu > 0 && node_cpu > 0 {
		rsc_info.Cpu = total_cpu * 100 / float64(node_cpu)
	}
	if total_memory > 0 && node_memory > 0 {
		rsc_info.Memory = total_memory * 100 / float64(node_memory)
	}

	return &rsc_info
}

//func (obj_k8s_cmd *Obj_K8sCmd) Upd_deploy_replicas(namespace string, deploy_info *st_deploy_info) bool {
//	deploymentsClient := obj_k8s_cmd.Obj_client_set.AppsV1().Deployments(namespace)
//
//	var deploy_name string
//
//	if deploy_info.instance != "" {
//		deploy_name = deploy_info.instance
//	} else {
//		deploy_name = deploy_info.deploy_name
//	}
//
//	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
//		// Retrieve the latest version of Deployment before attempting update
//		// RetryOnConflict uses exponential backoff to avoid exhausting the apiserver
//		result, getErr := deploymentsClient.Get(context.TODO(), deploy_name, metav1.GetOptions{})
//		if getErr != nil {
//			util.LOG_ERR("Failed to get latest version of Deployment: %v", getErr)
//			return getErr
//		}
//
//		if deploy_info.replicas > 0 {
//			result.Spec.Replicas = int32Ptr(deploy_info.replicas)
//		}
//		//result.Spec.Replicas = (*result.Spec.Replicas + 1) // reduce replica count
//		//result.Spec.Template.Spec.Containers[0].Image = "nginx:1.13" // change nginx version
//		_, updateErr := deploymentsClient.Update(context.TODO(), result, metav1.UpdateOptions{})
//		return updateErr
//	})
//	if retryErr != nil {
//		util.LOG_ERR("Update failed: %v", retryErr)
//		return false
//	}
//	util.LOG_DBG("UPDATE DEPLOYMENT %s", deploy_name)
//	return true
//}
//
//func (obj_k8s_cmd *Obj_K8sCmd) Patch_deploy(namespace string, deploy_info *st_deploy_info) {
//
//	deploymentsClient := obj_k8s_cmd.Obj_client_set.AppsV1().Deployments(namespace)
//
//	/* Patch */
//	patchData := map[string]interface{}{}
//
//	patchData["spec"] = map[string]interface{}{
//		"template": map[string]interface{}{
//			"metadata": map[string]interface{}{
//				"annotations": map[string]interface{}{
//					"kubectl.kubernetes.io/restartedAt": time.Now().Format(time.Stamp),
//				},
//			},
//		},
//	}
//
//	encodedData, err := json.Marshal(patchData)
//	if err != nil {
//		util.LOG_ERR("Failed json.Marshal")
//		return
//	}
//
//	var deploy_name string
//
//	if deploy_info.instance != "" {
//		deploy_name = deploy_info.instance
//	} else {
//		deploy_name = deploy_info.deploy_name
//	}
//
//	_, err = deploymentsClient.Patch(context.TODO(), deploy_name, types.MergePatchType, encodedData, metav1.PatchOptions{})
//
//	if err != nil {
//		util.LOG_ERR("Patch failed: %v", err)
//		return
//	}
//	util.LOG_DBG("PATCH DEPLOYMENT %s", deploy_name)
//	return
//}
//
//func (obj_k8s_cmd *Obj_K8sCmd) Del_deploy(namespace string, deploy_info *st_deploy_info) {
//
//	deploymentsClient := obj_k8s_cmd.Obj_client_set.AppsV1().Deployments(namespace)
//
//	var deploy_name string
//
//	if deploy_info.instance != "" {
//		deploy_name = deploy_info.instance
//	} else {
//		deploy_name = deploy_info.deploy_name
//	}
//
//	deletePolicy := metav1.DeletePropagationForeground
//	if err := deploymentsClient.Delete(context.TODO(), deploy_name, metav1.DeleteOptions{
//		PropagationPolicy: &deletePolicy,
//	}); err != nil {
//		util.LOG_ERR("Delete failed: %v", err)
//	}
//
//	util.LOG_DBG("DELETE DEPLOYMENT %s", deploy_name)
//}

func (obj_k8s_cmd *Obj_K8sCmd) Get_deploy(namespace string, deploy_name string) {

	deploymentsClient := obj_k8s_cmd.Obj_client_set.AppsV1().Deployments(namespace)

	if _, err := deploymentsClient.Get(context.TODO(), deploy_name, metav1.GetOptions{}); err != nil {
		util.LOG_ERR("Get Deployment failed: %v", err)
		return
	}

	util.LOG_DBG("GET DEPLOYMENT %s", deploy_name)
}

func (obj_k8s_cmd *Obj_K8sCmd) Upd_configmap_cnfd(namespace string, nf_prof *nf_profile.St_nf_prof) {

	ConfigmapsClient := obj_k8s_cmd.Obj_client_set.CoreV1().ConfigMaps(namespace)

	cm, err := ConfigmapsClient.Get(context.TODO(), nf_prof.ConfigmapName, metav1.GetOptions{})
	if err != nil {
		util.LOG_ERR("Search failed: %s %s %v", namespace, nf_prof.ConfigmapName, err)
		return
	}

	b_json, _ := json.Marshal(nf_prof)

	cm.Data["cnfDescriptor"] = string(b_json)

	cm, err = ConfigmapsClient.Update(context.TODO(), cm, metav1.UpdateOptions{})
	if err != nil {
		util.LOG_ERR("Update failed: %s %v %v", namespace, cm, err)
		return
	}

	util.LOG_SVC("UPDATE CONFIGMAP (namespace:%s, name:%s)", namespace, nf_prof.ConfigmapName)

	return
}

func (obj_k8s_cmd *Obj_K8sCmd) Get_service(namespace string, svc_name string) *corev1.Service {

	ServicesClient := obj_k8s_cmd.Obj_client_set.CoreV1().Services(namespace)

	service, err := ServicesClient.Get(context.TODO(), svc_name, metav1.GetOptions{})
	if err != nil {
		util.LOG_ERR("Search failed: %s %s %v", namespace, svc_name, err)
		return nil
	}

	return service
}

func (obj_k8s_cmd *Obj_K8sCmd) Get_service_clusterip(namespace string, svc_name string) string {
	service := obj_k8s_cmd.Get_service(namespace, svc_name)
	if service != nil {
		return service.Spec.ClusterIP
	}

	return ""
}

//func (obj_k8s_cmd *Obj_K8sCmd) Destroy() {
//	return
//}

func int32Ptr(i int32) *int32 { return &i }
