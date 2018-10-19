package stub
// Example operator handle code below method names have been chosen for explicitness
// not idiomatic go-ness

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"github.com/flexshopper/redis-operator/pkg/apis/cache/v1alpha1"
	rConfig "github.com/flexshopper/redis-operator/pkg/config"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	"k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)


func NewHandler() sdk.Handler {
	return &Handler{}
}

type Handler struct {
	// Fill me
}

// This method handles incoming events, we filter for our own and take action
// The incoming event looks like:
// { Deleted: <true|false>, Object: Redis }
func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *v1alpha1.Redis:
		if event.Deleted {
			return deleteResources(o)
		}

		err := createOrUpdateResources(o)
		if err != nil && !errors.IsAlreadyExists(err) {
			logrus.Errorf("failed to reconcile redis with error : %v", err)
			return err
		}
	}

	return nil
}

func deleteResources(redis *v1alpha1.Redis) error {

	deploy, err := getDeploymentDefinition(redis)
	if err != nil {
		return err
	}

	err =  sdk.Delete(deploy)
	if err != nil {
		return err
	}

	cm, err := getConfigMapDefinition(redis)
	if err != nil {
		return err
	}

	err = sdk.Delete(cm)
	if err != nil {
		return err
	}

	svc := getServiceDefinition(redis)
	err = sdk.Delete(svc)
	if err != nil {
		return err
	}

	return nil
}

func createOrUpdateResources(r *v1alpha1.Redis) error {
	err := createOrUpdateConfigMap(r)
	if err != nil {
		return err
	}

	err = createOrUpdateDeployment(r)
	if err != nil {
		return err
	}

	err = createOrUpdateService(r)
	if err != nil {
		return err
	}

	return nil
}

func createOrUpdateService(r *v1alpha1.Redis) error {
	redis := r.DeepCopy()
	changed := redis.SetDefaults()

	if changed {
		svc := getServiceDefinition(redis)

		err := sdk.Update(svc)

		if errors.IsNotFound(err) {
			err = sdk.Create(svc)
		}

		if err != nil && errors.IsAlreadyExists(err) {
			return err
		}
	}

	return nil
}

func createOrUpdateConfigMap(r *v1alpha1.Redis) error {
	redis := r.DeepCopy()
	changed := redis.SetDefaults()

	if changed {
		cm, err := getConfigMapDefinition(redis)
		if err != nil {
			return err
		}

		err = sdk.Update(cm)

		if errors.IsNotFound(err) {
			err = sdk.Create(cm)
		}

		if err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
	}

	return nil
}

func createOrUpdateDeployment(r *v1alpha1.Redis) error {
	redis := r.DeepCopy()
	changed := redis.SetDefaults()
	if changed {
		deploy, err := getDeploymentDefinition(redis)
		if err != nil {
			return err
		}

		err = sdk.Update(deploy)

		if errors.IsNotFound(err) {
			err = sdk.Create(deploy)
		}

		if err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
	}

	return nil
}

func getServiceDefinition(redis *v1alpha1.Redis) *corev1.Service {
	labels := redisLabels(redis.Name)
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind: "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: redis.Name,
			Namespace: redis.Namespace,
			Labels: genericObjectDefinitionLabels(),
		},
		Spec: corev1.ServiceSpec{
			Type: "ClusterIP",
			SessionAffinity: "None",
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name: "redis",
					Port: redis.Spec.Port,
					TargetPort: intstr.IntOrString{
						IntVal: redis.Spec.Port,
					},
					Protocol: "TCP",
				},
			},
		},
	}
}

func getDeploymentDefinition(redis *v1alpha1.Redis) (*v1.Deployment, error) {
	replicas := int32(1)
	redisConfigs, err := rConfig.ParseConfig(redis.Spec.DeepCopy())

	if err != nil {
		return nil, err
	}

	configHash := getMd5(redisConfigs)
	labels := getCombinedLabels(redis.Name)

	return &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind: "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: redis.Name,
			Namespace: redis.Namespace,
			Labels: labels,
		},
		Spec: v1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Annotations: map[string]string{
						"configmap/hash": configHash,
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "redis-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: redis.Name,
									},
									Items: []corev1.KeyToPath{
										{
											Key: "redis.config",
											Path: "redis.conf",
										},
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Image: redis.Spec.Image,
							Name: redis.Name,
							Command: []string{
								"redis-server",
								"/usr/local/etc/redis/redis.conf",
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: redis.Spec.Port,
									Name: "redis",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name: "redis-config",
									MountPath: "/usr/local/etc/redis/",
								},
							},
						},
					},
				},
			},
		},
	}, nil
}

func getConfigMapDefinition(redis *v1alpha1.Redis) (*corev1.ConfigMap, error) {
	redisConfigs, err := rConfig.ParseConfig(redis.Spec.DeepCopy())

	if err != nil {
		return nil, err
	}

	rConfigMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind: "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: redis.Name,
			Namespace: redis.Namespace,
			Labels: genericObjectDefinitionLabels(),
		},
		Data: map[string]string{
			"redis.config": redisConfigs,
		},
	}

	return rConfigMap, nil
}

func getCombinedLabels(name string) map[string]string {
	labels := redisLabels(name)
	genericLabels := genericObjectDefinitionLabels()

	for k, v := range genericLabels {
		labels[k] = v
	}

	return labels
}

func genericObjectDefinitionLabels() map[string]string {
	return map[string]string{
		"flexOperator": "cache",
	}
}

func redisLabels(name string) map[string]string {
	return map[string]string{
		"lru-cache": name,
	}
}

func getMd5(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}