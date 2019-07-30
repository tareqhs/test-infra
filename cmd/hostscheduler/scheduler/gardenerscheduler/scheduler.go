//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gardenerscheduler

import (
	"context"
	"errors"
	"flag"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/cmd/hostscheduler/scheduler"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	Name = "gardener"
)

var kubeconfigPath string

var Register scheduler.Register = func(m scheduler.Registrations) {
	if m == nil {
		m = make(scheduler.Registrations)
	}
	m[Name] = &scheduler.Registration{
		Interface: registerScheduler,
		Flags:     registerFlags,
	}
}

var registerFlags scheduler.RegisterFlagsFunc = func(fs *flag.FlagSet) {
	fs.StringVar(&kubeconfigPath, "kubeconfig", "", "Path to the gardener cluster kubeconfigPath")
}

var registerScheduler scheduler.RegisterInterfaceFromArgsFunc = func(ctx context.Context, logger *logrus.Logger) (scheduler.Interface, error) {

	if kubeconfigPath == "" {
		return nil, errors.New("no kubeconfig is specified")
	}
	logger.Debugf("Kubeconig path: %s", kubeconfigPath)

	return New(ctx, logger, kubeconfigPath)
}

func New(_ context.Context, logger *logrus.Logger, kubeconfigPath string) (*gardenerscheduler, error) {

	k8sClient, err := kubernetes.NewClientFromFile("", kubeconfigPath, client.Options{
		Scheme: kubernetes.GardenScheme,
	})
	if err != nil {
		return nil, err
	}

	namespace, err := getNamespaceOfKubeconfig(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return &gardenerscheduler{
		client:    k8sClient,
		logger:    logger,
		namespace: namespace,
	}, nil
}

var _ scheduler.Interface = &gardenerscheduler{}