package main

import (
	"flag"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"k8s.io/klog"
	"os"
	"secrets-manager/pkg/kube"
	"secrets-manager/pkg/trackers"
	"secrets-manager/version"
)

func main() {
	flag.CommandLine.Parse([]string{})
	klog.SetOutputBySeverity("INFO", ioutil.Discard)
	klog.SetOutputBySeverity("WARNING", ioutil.Discard)

	var watchedNamespace string
	var kubeContext string
	var kubeConfig string

	init := func() {
		err := kube.Init(kube.InitOptions{
			kube.KubeConfigOptions{
				ConfigPath: kubeConfig,
				Context:    kubeContext,
			}})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to initialize kube: %s\n", err)
			os.Exit(1)
		}
		klog.Infof("kube config init success")
	}
	rootCmd := &cobra.Command{Use: "secret-manager"}
	rootCmd.PersistentFlags().StringVarP(&watchedNamespace, "namespace", "n", "kube-secretmanager", "which namespace to watch")
	rootCmd.PersistentFlags().StringVarP(&kubeContext, "kube-context", "", os.Getenv("KUBE_CONTEXT"), "The name of the kubeconfig context to use (can be set with $KUBE_CONTEXT).")
	rootCmd.PersistentFlags().StringVarP(&kubeConfig, "kube-config", "", os.Getenv("KUBE_CONFIG"), "Path to the kubeconfig file (can be set with $KUBE_CONFIG).")

	versionCmd := &cobra.Command{
		Use: "version",
		Run: func(_ *cobra.Command, _ []string) {
			version.PrintVersion(version.GetVersion())
		},
	}
	rootCmd.AddCommand(versionCmd)

	watchCmd := &cobra.Command{
		Use:     "watch",
		Short:   "Watch secrets in namespace and sync to other resources",
		Example: "",
		Run: func(cmd *cobra.Command, args []string) {
			init()
			tracker := trackers.NewTracker(kube.Kubernetes, watchedNamespace)
			tracker.InitResource()
			tracker.Run()
		},
	}
	rootCmd.AddCommand(watchCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
