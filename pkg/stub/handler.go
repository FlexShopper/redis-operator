package stub

import (
	"context"
	"github.com/flexshopper/redis-operator/pkg/apis/cache/v1alpha1"
	rConfig "github.com/flexshopper/redis-operator/pkg/config"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/api/apps/v1"
	"crypto/md5"
	"encoding/hex"
)

func NewHandler() sdk.Handler {
	return &Handler{}
}

type Handler struct {
	// Fill me
}

func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *v1alpha1.Redis:
		if event.Deleted {
			return deleteResources(o)
		}

		err := reconcile(o)
		if err != nil && !errors.IsAlreadyExists(err) {
			logrus.Errorf("failed to reconcile redis with error : %v", err)
			return err
		}
	}

	return nil
}

func deleteResources(redis *v1alpha1.Redis) error {

	deployment := getRedisDeployment(redis)

	err :=  sdk.Delete(deployment)
	if err != nil {
		return err
	}

	configMap := getConfigMap(redis)

	err = sdk.Delete(configMap)
	if err != nil {
		return err
	}

	return nil
}

func reconcile(r *v1alpha1.Redis) error {
	err := reconcileConfigMap(r)
	if err != nil {
		return err
	}

	err = reconcileDeployment(r)
	if err != nil {
		return err
	}

	return nil
}

func reconcileConfigMap(r *v1alpha1.Redis) error {
	redis := r.DeepCopy()
	changed := redis.SetDefaults()

	if changed {
		configMap := getConfigMap(redis)
		err := sdk.Update(configMap)

		if errors.IsNotFound(err) {
			err = sdk.Create(configMap)
		}

		if err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
	}

	return nil
}

func reconcileDeployment(r *v1alpha1.Redis) error {
	redis := r.DeepCopy()
	changed := redis.SetDefaults()
	if changed {
		deployment := getRedisDeployment(redis)
		err := sdk.Update(deployment)

		if errors.IsNotFound(err) {
			err = sdk.Create(deployment)
		}

		if err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
	}

	return nil
}

func getRedisDeployment(redis *v1alpha1.Redis) *v1.Deployment {
	replicas := int32(1)
	redisConfigs, err := rConfig.ParseConfig(redis.Spec.DeepCopy())

	if err != nil {
		panic(err)
	}

	configHash := getMd5(redisConfigs)
	labels := getRedisLabels(redis.Name)

	deployment := &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind: "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: redis.Name,
			Namespace: redis.Namespace,
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
	}

	return deployment
}

func getConfigMap(redis *v1alpha1.Redis) *corev1.ConfigMap {
	redisConfigs, err := rConfig.ParseConfig(redis.Spec.DeepCopy())

	if err != nil {
		panic(err)
	}

	rConfigMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind: "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: redis.Name,
			Namespace: redis.Namespace,
		},
		Data: map[string]string{
			"redis.config": redisConfigs,
		},
	}

	return rConfigMap
}

func getRedisLabels(name string) map[string]string {
	return map[string]string{
		"lru-cache": name,
	}
}

func getMd5(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}