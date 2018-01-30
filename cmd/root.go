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
	"flag"

	"github.com/spf13/pflag"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	cfgFile                   string
	logLevel                  = zapcore.InfoLevel
	kubeConfigFile            string
	kubeClientConfigOverrides = &clientcmd.ConfigOverrides{}

	kubeClient *kubernetes.Clientset
	logger     *zap.Logger
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
	SilenceUsage: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logConfig := zap.NewProductionConfig()
		logConfig.Level.SetLevel(logLevel)
		logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		logConfig.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
		logger, _ = logConfig.Build()
		_ = zap.ReplaceGlobals(logger)
		_ = zap.RedirectStdLog(logger)
		defer logger.Sync()

		kubeConfigLoader := clientcmd.NewDefaultClientConfigLoadingRules()
		if kubeConfigFile != "" {
			logger.Info("using specified kube config file", zap.String("file", kubeConfigFile))
			kubeConfigLoader.ExplicitPath = kubeConfigFile
		}
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(kubeConfigLoader, kubeClientConfigOverrides)
		restConfig, err := kubeConfig.ClientConfig()
		if err != nil {
			logger.Fatal("failed to get REST config", zap.Error(err))
		}
		kubeClient = kubernetes.NewForConfigOrDie(restConfig)

		namespace, _, _ := kubeConfig.Namespace()
		logger.Debug("running against namespace", zap.String("namespace", namespace))
		pods, err := kubeClient.Core().Pods(namespace).List(metav1.ListOptions{})
		if err != nil {
			logger.Fatal("failed to list pods", zap.Error(err))
		}
		logger.Info("returned pods", zap.Stringer("pods", pods))
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Fatal("root command failed", zap.Error(err))
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kube-client-template.yaml)")
	rootCmd.PersistentFlags().AddGoFlag(&flag.Flag{
		Name:     "log-level",
		Usage:    "log level",
		Value:    &logLevel,
		DefValue: zapcore.InfoLevel.String(),
	})

	kubernetesFlagSet := pflag.NewFlagSet("Kubernetes configuration", pflag.ContinueOnError)
	clientcmd.BindOverrideFlags(kubeClientConfigOverrides, kubernetesFlagSet, clientcmd.RecommendedConfigOverrideFlags("kubernetes-"))
	kubernetesFlagSet.StringVar(&kubeConfigFile, "kubernetes-config", "", "(optional) absolute path to the kubeconfig file")
	kubernetesFlagSet.VisitAll(func(f *pflag.Flag) {
		_ = kubernetesFlagSet.MarkHidden(f.Name)
	})

	rootCmd.PersistentFlags().AddFlagSet(kubernetesFlagSet)
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
		logger.Info("loaded config from file", zap.String("file", viper.ConfigFileUsed()))
	}
}
