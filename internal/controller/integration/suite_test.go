/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"
	"time"

	eventsv1 "k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	runtimev1alpha1 "github.com/trussiumhq/trussium-operator/api/v1alpha1"
	"github.com/trussiumhq/trussium-operator/internal/controller"
)

const (
	integrationTimeout      = 15 * time.Second
	integrationPollInterval = 100 * time.Millisecond
)

var (
	testEnvironment *envtest.Environment
	testClient      client.Client
	testScheme      *runtime.Scheme
)

func TestMain(m *testing.M) {
	projectRoot, err := integrationProjectRoot()
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"resolve project root: %v\n",
			err,
		)
		os.Exit(1)
	}

	testScheme = runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(testScheme))
	utilruntime.Must(eventsv1.AddToScheme(testScheme))
	utilruntime.Must(runtimev1alpha1.AddToScheme(testScheme))

	testEnvironment = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join(
				projectRoot,
				"config",
				"crd",
				"bases",
			),
		},
		ErrorIfCRDPathMissing: true,
	}

	restConfig, err := testEnvironment.Start()
	if err != nil {
		fmt.Fprintf(
			os.Stderr,
			"start envtest environment: %v\n",
			err,
		)
		os.Exit(1)
	}

	manager, err := ctrl.NewManager(
		restConfig,
		ctrl.Options{
			Scheme: testScheme,
			Metrics: metricsserver.Options{
				BindAddress: "0",
			},
			HealthProbeBindAddress: "0",
			LeaderElection:         false,
		},
	)
	if err != nil {
		_ = testEnvironment.Stop()

		fmt.Fprintf(
			os.Stderr,
			"create controller manager: %v\n",
			err,
		)
		os.Exit(1)
	}

	reconciler := &controller.TrussiumRuntimeReconciler{
		Client: manager.GetClient(),
		Scheme: manager.GetScheme(),
		Recorder: manager.GetEventRecorder(
			"trussiumruntime-integration",
		),
	}

	if err := reconciler.SetupWithManager(manager); err != nil {
		_ = testEnvironment.Stop()

		fmt.Fprintf(
			os.Stderr,
			"register TrussiumRuntime controller: %v\n",
			err,
		)
		os.Exit(1)
	}

	managerContext, cancelManager := context.WithCancel(
		context.Background(),
	)

	managerDone := make(chan error, 1)

	go func() {
		managerDone <- manager.Start(managerContext)
	}()

	cacheContext, cancelCacheWait := context.WithTimeout(
		context.Background(),
		integrationTimeout,
	)

	cacheStarted :=
		manager.GetCache().WaitForCacheSync(cacheContext)

	cancelCacheWait()

	if !cacheStarted {
		cancelManager()
		_ = waitForManagerStop(managerDone)
		_ = testEnvironment.Stop()

		fmt.Fprintln(
			os.Stderr,
			"controller manager cache did not synchronize before timeout",
		)

		os.Exit(1)
	}

	testClient, err = client.New(
		restConfig,
		client.Options{
			Scheme: testScheme,
		},
	)
	if err != nil {
		cancelManager()
		_ = waitForManagerStop(managerDone)
		_ = testEnvironment.Stop()

		fmt.Fprintf(
			os.Stderr,
			"create integration test client: %v\n",
			err,
		)

		os.Exit(1)
	}

	exitCode := m.Run()

	cancelManager()

	if err := waitForManagerStop(managerDone); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"stop controller manager: %v\n",
			err,
		)

		if exitCode == 0 {
			exitCode = 1
		}
	}

	if err := testEnvironment.Stop(); err != nil {
		fmt.Fprintf(
			os.Stderr,
			"stop envtest environment: %v\n",
			err,
		)

		if exitCode == 0 {
			exitCode = 1
		}
	}

	os.Exit(exitCode)
}

func integrationProjectRoot() (string, error) {
	_, currentFile, _, ok := goruntime.Caller(0)
	if !ok {
		return "", errors.New(
			"determine suite source file",
		)
	}

	return filepath.Clean(
		filepath.Join(
			filepath.Dir(currentFile),
			"..",
			"..",
			"..",
		),
	), nil
}

func waitForManagerStop(
	managerDone <-chan error,
) error {
	select {
	case err := <-managerDone:
		if err == nil ||
			errors.Is(err, context.Canceled) {
			return nil
		}

		return err

	case <-time.After(integrationTimeout):
		return errors.New(
			"controller manager did not stop before timeout",
		)
	}
}
