// Copyright © 2018 Jimmi Dyson <jimmidyson@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	cfgFile                   string
	interactive               bool
	kubeConfigFile            string
	kubeClientConfigOverrides = &clientcmd.ConfigOverrides{}

	kubeClient *kubernetes.Clientset
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kube-client-template",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		kubeConfigLoader := clientcmd.NewDefaultClientConfigLoadingRules()
		if kubeConfigFile != "" {
			kubeConfigLoader.ExplicitPath = kubeConfigFile
		}
		var kubeConfig clientcmd.ClientConfig
		if interactive {
			kubeConfig = clientcmd.NewInteractiveDeferredLoadingClientConfig(kubeConfigLoader, kubeClientConfigOverrides, os.Stdin)
		} else {
			kubeConfig = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(kubeConfigLoader, kubeClientConfigOverrides)
		}

		restConfig, err := kubeConfig.ClientConfig()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		kubeClient = kubernetes.NewForConfigOrDie(restConfig)

		namespace, _, _ := kubeConfig.Namespace()
		fmt.Println(kubeClient.Core().Pods(namespace).List(metav1.ListOptions{}))
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kube-client-template.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&interactive, "interactive", "i", false, "interactive")

	clientcmd.BindOverrideFlags(kubeClientConfigOverrides, rootCmd.PersistentFlags(), clientcmd.RecommendedConfigOverrideFlags(""))
	_, err := homedir.Dir()
	if err == nil {
		rootCmd.PersistentFlags().StringVar(&kubeConfigFile, clientcmd.RecommendedConfigPathFlag, clientcmd.RecommendedHomeFile, "(optional) absolute path to the kubeconfig file")
	} else {
		rootCmd.PersistentFlags().StringVar(&kubeConfigFile, clientcmd.RecommendedConfigPathFlag, "", "(optional) absolute path to the kubeconfig file")
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err == nil {
			viper.AddConfigPath(home)
		}

		viper.AddConfigPath(".")
		viper.SetConfigName(".kube-client-template")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
