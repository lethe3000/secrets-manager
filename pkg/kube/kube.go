package kube

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"secrets-manager/pkg/utils"

	// load auth plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/exec"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp" // only required to authenticate against GKE clusters
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	_ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

const (
	kubeTokenFilePath     = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	kubeNamespaceFilePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

var (
	Kubernetes kubernetes.Interface
)

type InitOptions struct {
	KubeConfigOptions
}

func Init(opts InitOptions) error {
	config, err := GetKubeConfig(opts.KubeConfigOptions)
	if err != nil {
		return err
	}

	if config != nil {
		clientset, err := kubernetes.NewForConfig(config.Config)
		if err != nil {
			return err
		}
		Kubernetes = clientset
	}
	return nil
}

type KubeConfigOptions struct {
	Context          string
	ConfigPath       string
	ConfigDataBase64 string
}

type KubeConfig struct {
	Config           *rest.Config
	Context          string
	DefaultNamespace string
}

func GetKubeConfig(opts KubeConfigOptions) (*KubeConfig, error) {
	// Try to load from kubeconfig in flags or from ~/.kube/config
	config, outOfClusterErr := getOutOfClusterConfig(opts.Context, opts.ConfigPath, opts.ConfigDataBase64)

	if config == nil {
		if hasInClusterConfig() {
			// Try to configure as inCluster
			if config, err := getInClusterConfig(); err != nil {
				if opts.ConfigPath != "" || opts.Context != "" || opts.ConfigDataBase64 != "" {
					if outOfClusterErr != nil {
						return nil, fmt.Errorf("out-of-cluster config error: %v, in-cluster config error: %v", outOfClusterErr, err)
					}
				} else {
					klog.Info("config as in cluster fail")
					return nil, err
				}
			} else if config != nil {
				return config, nil
			}
		} else {
			// if not in cluster return outOfCluster error
			klog.Info("config as out of cluster fail")
			if outOfClusterErr != nil {
				return nil, outOfClusterErr
			}
		}

		return nil, nil
	}
	klog.Info("config cluster success")
	return config, outOfClusterErr
}

type GetAllContextsClientsOptions struct {
	KubeConfig string
}

func makeOutOfClusterClientConfigError(configPath, context string, err error) error {
	baseErrMsg := fmt.Sprintf("out-of-cluster configuration problem")

	if configPath != "" {
		baseErrMsg += fmt.Sprintf(", custom kube config path is %q", configPath)
	}

	if context != "" {
		baseErrMsg += fmt.Sprintf(", custom kube context is %q", context)
	}

	return fmt.Errorf("%s: %s", baseErrMsg, err)
}

func GetClientConfig(context string, configPath string, configData []byte) (clientcmd.ClientConfig, error) {
	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}
	if context != "" {
		overrides.CurrentContext = context
	}

	if configData != nil {
		if config, err := clientcmd.Load(configData); err != nil {
			return nil, fmt.Errorf("unable to load config data: %s", err)
		} else {
			return clientcmd.NewDefaultClientConfig(*config, overrides), nil
		}
	}

	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	if configPath != "" {
		rules.ExplicitPath = configPath
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides), nil
}

func hasInClusterConfig() bool {
	token, _ := utils.FileExists(kubeTokenFilePath)
	ns, _ := utils.FileExists(kubeNamespaceFilePath)
	return token && ns
}

func getOutOfClusterConfig(context, configPath, configDataBase64 string) (*KubeConfig, error) {
	res := &KubeConfig{}

	var configData []byte
	if configDataBase64 != "" {
		if data, err := base64.StdEncoding.DecodeString(configDataBase64); err != nil {
			return nil, fmt.Errorf("unable to decode base64 config data: %s", err)
		} else {
			configData = data
		}
	}

	clientConfig, err := GetClientConfig(context, configPath, configData)
	if err != nil {
		return nil, makeOutOfClusterClientConfigError(configPath, context, err)
	}

	if ns, _, err := clientConfig.Namespace(); err != nil {
		return nil, fmt.Errorf("cannot determine default kubernetes namespace: %s", err)
	} else {
		res.DefaultNamespace = ns
	}

	if config, err := clientConfig.ClientConfig(); err != nil {
		return nil, makeOutOfClusterClientConfigError(configPath, context, err)
	} else if config == nil {
		return nil, nil
	} else {
		res.Config = config
	}

	if context == "" {
		if rc, err := clientConfig.RawConfig(); err != nil {
			return nil, fmt.Errorf("cannot get raw kubernetes config: %s", err)
		} else {
			res.Context = rc.CurrentContext
		}
	} else {
		res.Context = context
	}

	return res, nil
}

func getInClusterConfig() (*KubeConfig, error) {
	res := &KubeConfig{}

	if config, err := rest.InClusterConfig(); err != nil {
		return nil, fmt.Errorf("in-cluster configuration problem: %s", err)
	} else {
		res.Config = config
	}

	if data, err := ioutil.ReadFile(kubeNamespaceFilePath); err != nil {
		return nil, fmt.Errorf("in-cluster configuration problem: cannot determine default kubernetes namespace: error reading %s: %s", kubeNamespaceFilePath, err)
	} else {
		res.DefaultNamespace = string(data)
	}

	return res, nil
}
