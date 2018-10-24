package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// We'll define some default values we'll reference in SetDefaults
const (
	defaultMaxMemory = "2mb"
	defaultMaxMemoryEvictionPolicy = "allkeys-lru"
	defaultPort = 6379
	defaultImage = "redis:4-alpine"
)


// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type RedisList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Redis `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Redis struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              RedisSpec   `json:"spec"`
	Status            RedisStatus `json:"status,omitempty"`
}


func (redis *Redis) SetDefaults() bool {
	changed := false
	rSpec := &redis.Spec

	if rSpec.MaxMemory == "" {
		rSpec.MaxMemory = defaultMaxMemory
		changed = true
	}

	if rSpec.MaxMemoryEvictionPolicy == "" {
		rSpec.MaxMemoryEvictionPolicy = defaultMaxMemoryEvictionPolicy
		changed = true
	}

	if rSpec.Port == 0 {
		rSpec.Port = defaultPort
		changed = true
	}

	if rSpec.Image == "" {
		rSpec.Image = defaultImage
		changed = true
	}

	return changed
}

type RedisSpec struct {
	Image string `json:"string,omitempty"`
	Port int32 `json:"port,omitempty"`
	PasswordSecret string `json:"passwordSecret,omitempty"`
	MaxMemory string `json:"maxMemory,omitempty"`
	MaxMemoryEvictionPolicy string `json:"maxMemoryEvictionPolicy,omitempty"`
}

type RedisStatus struct {
	Phase string `json:"phase"`
	Errors []string `json:"errors,omitempty"`
}
