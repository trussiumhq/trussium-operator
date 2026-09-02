//go:build e2e
// +build e2e

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

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/trussiumhq/trussium-operator/test/utils"
)

const namespace = "trussium-operator-system"

const serviceAccountName = "trussium-operator-controller-manager"

const metricsServiceName = "trussium-operator-controller-manager-metrics-service"

const metricsRoleBindingName = "trussium-operator-metrics-binding"

const (
	runtimeResourceName = "smoke-runtime"

	runtimeImageRepository = "ghcr.io/trussiumhq/trussium"

	initialRuntimeImageTag  = "1.17.0"
	upgradedRuntimeImageTag = "1.22.0"

	runtimeImage = runtimeImageRepository + ":" + initialRuntimeImageTag

	upgradedRuntimeImage = runtimeImageRepository + ":" + upgradedRuntimeImageTag

	runtimeConfigChecksumAnnotation = "runtime.trussium.io/config-checksum"

	runtimeProviderBaseURLEnvironment = "TRUSSIUM_PROVIDER__BASE_URL"

	initialRuntimeModel = "llama3.2"

	initialRuntimeProviderBaseURL = "http://ollama.default.svc.cluster.local:11434"
	updatedRuntimeProviderBaseURL = "http://ollama-updated.default.svc.cluster.local:11434"

	runtimeUpgradeStartedEvent = "RuntimeUpgradeStarted"

	runtimeUpgradeCompletedEvent = "RuntimeUpgradeCompleted"

	upgradeCompleteReason = "UpgradeComplete"
)

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	BeforeAll(func() {
		By("creating manager namespace")

		cmd := exec.Command(
			"kubectl",
			"create",
			"ns",
			namespace,
		)

		_, err := utils.Run(cmd)
		Expect(err).NotTo(
			HaveOccurred(),
			"Failed to create namespace",
		)

		By(
			"labeling the namespace to enforce the restricted security policy",
		)

		cmd = exec.Command(
			"kubectl",
			"label",
			"--overwrite",
			"ns",
			namespace,
			"pod-security.kubernetes.io/enforce=restricted",
		)

		_, err = utils.Run(cmd)
		Expect(err).NotTo(
			HaveOccurred(),
			"Failed to label namespace with restricted policy",
		)

		By("installing CRDs")

		cmd = exec.Command(
			"make",
			"install",
		)

		_, err = utils.Run(cmd)
		Expect(err).NotTo(
			HaveOccurred(),
			"Failed to install CRDs",
		)

		By("deploying the controller-manager")

		cmd = exec.Command(
			"make",
			"deploy",
			fmt.Sprintf(
				"IMG=%s",
				managerImage,
			),
		)

		_, err = utils.Run(cmd)
		Expect(err).NotTo(
			HaveOccurred(),
			"Failed to deploy the controller-manager",
		)
	})

	AfterAll(func() {
		By(
			"cleaning up the curl pod for metrics",
		)

		cmd := exec.Command(
			"kubectl",
			"delete",
			"pod",
			"curl-metrics",
			"-n",
			namespace,
		)

		_, _ = utils.Run(cmd)

		By(
			"cleaning up the metrics ClusterRoleBinding",
		)

		cmd = exec.Command(
			"kubectl",
			"delete",
			"clusterrolebinding",
			metricsRoleBindingName,
			"--ignore-not-found",
		)

		_, _ = utils.Run(cmd)

		By(
			"undeploying the controller-manager",
		)

		cmd = exec.Command(
			"make",
			"undeploy",
		)

		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")

		cmd = exec.Command(
			"make",
			"uninstall",
		)

		_, _ = utils.Run(cmd)

		By("removing manager namespace")

		cmd = exec.Command(
			"kubectl",
			"delete",
			"ns",
			namespace,
		)

		_, _ = utils.Run(cmd)
	})

	AfterEach(func() {
		specReport := CurrentSpecReport()

		if specReport.Failed() {
			By(
				"Fetching controller manager pod logs",
			)

			cmd := exec.Command(
				"kubectl",
				"logs",
				controllerPodName,
				"-n",
				namespace,
			)

			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(
					GinkgoWriter,
					"Controller logs:\n%s\n",
					controllerLogs,
				)
			} else {
				_, _ = fmt.Fprintf(
					GinkgoWriter,
					"Failed to get Controller logs: %s\n",
					err,
				)
			}

			By(
				"Fetching Kubernetes events",
			)

			cmd = exec.Command(
				"kubectl",
				"get",
				"events",
				"-n",
				namespace,
				"--sort-by=.lastTimestamp",
			)

			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(
					GinkgoWriter,
					"Kubernetes events:\n%s\n",
					eventsOutput,
				)
			} else {
				_, _ = fmt.Fprintf(
					GinkgoWriter,
					"Failed to get Kubernetes events: %s\n",
					err,
				)
			}

			By(
				"Fetching curl-metrics logs",
			)

			cmd = exec.Command(
				"kubectl",
				"logs",
				"curl-metrics",
				"-n",
				namespace,
			)

			metricsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(
					GinkgoWriter,
					"Metrics logs:\n%s\n",
					metricsOutput,
				)
			} else {
				_, _ = fmt.Fprintf(
					GinkgoWriter,
					"Failed to get curl-metrics logs: %s\n",
					err,
				)
			}

			By(
				"Fetching controller manager pod description",
			)

			cmd = exec.Command(
				"kubectl",
				"describe",
				"pod",
				controllerPodName,
				"-n",
				namespace,
			)

			podDescription, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(
					GinkgoWriter,
					"Pod description:\n%s\n",
					podDescription,
				)
			} else {
				_, _ = fmt.Fprintf(
					GinkgoWriter,
					"Failed to describe controller pod: %s\n",
					err,
				)
			}
		}
	})

	SetDefaultEventuallyTimeout(
		2 * time.Minute,
	)

	SetDefaultEventuallyPollingInterval(
		time.Second,
	)

	Context("Manager", func() {
		It(
			"should run successfully",
			func() {
				By(
					"validating that the controller-manager pod is running as expected",
				)

				verifyControllerUp := func(
					g Gomega,
				) {
					cmd := exec.Command(
						"kubectl",
						"get",
						"pods",
						"-l",
						"control-plane=controller-manager",
						"-o",
						"go-template={{ range .items }}"+
							"{{ if not .metadata.deletionTimestamp }}"+
							"{{ .metadata.name }}"+
							"{{ \"\\n\" }}"+
							"{{ end }}"+
							"{{ end }}",
						"-n",
						namespace,
					)

					podOutput, err :=
						utils.Run(cmd)

					g.Expect(err).NotTo(
						HaveOccurred(),
					)

					podNames :=
						utils.GetNonEmptyLines(
							podOutput,
						)

					g.Expect(podNames).To(
						HaveLen(1),
					)

					controllerPodName =
						podNames[0]

					g.Expect(
						controllerPodName,
					).To(
						ContainSubstring(
							"controller-manager",
						),
					)

					cmd = exec.Command(
						"kubectl",
						"get",
						"pods",
						controllerPodName,
						"-o",
						"jsonpath={.status.phase}",
						"-n",
						namespace,
					)

					output, err :=
						utils.Run(cmd)

					g.Expect(err).NotTo(
						HaveOccurred(),
					)

					g.Expect(output).To(
						Equal("Running"),
					)
				}

				Eventually(
					verifyControllerUp,
				).Should(
					Succeed(),
				)
			},
		)

		It(
			"should ensure the metrics endpoint is serving metrics",
			func() {
				By(
					"creating a ClusterRoleBinding for the service account to allow access to metrics",
				)

				cmd := exec.Command(
					"kubectl",
					"create",
					"clusterrolebinding",
					metricsRoleBindingName,
					"--clusterrole=trussium-operator-metrics-reader",
					"--serviceaccount=trussium-operator-system:trussium-operator-controller-manager",
				)

				_, err := utils.Run(cmd)

				Expect(err).NotTo(
					HaveOccurred(),
					"Failed to create ClusterRoleBinding",
				)

				By(
					"validating that the metrics service is available",
				)

				cmd = exec.Command(
					"kubectl",
					"get",
					"service",
					metricsServiceName,
					"-n",
					namespace,
				)

				_, err = utils.Run(cmd)

				Expect(err).NotTo(
					HaveOccurred(),
				)

				By(
					"getting the service account token",
				)

				token, err :=
					serviceAccountToken()

				Expect(err).NotTo(
					HaveOccurred(),
				)

				Expect(token).NotTo(
					BeEmpty(),
				)

				By(
					"ensuring the controller pod is ready",
				)

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"pod",
							controllerPodName,
							"-n",
							namespace,
							"-o",
							"jsonpath={.status.conditions[?(@.type=='Ready')].status}",
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						g.Expect(output).To(
							Equal("True"),
						)
					},
					3*time.Minute,
					time.Second,
				).Should(
					Succeed(),
				)

				By(
					"verifying that the controller manager is serving the metrics server",
				)

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"logs",
							controllerPodName,
							"-n",
							namespace,
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						g.Expect(output).To(
							ContainSubstring(
								"Serving metrics server",
							),
						)
					},
					3*time.Minute,
					time.Second,
				).Should(
					Succeed(),
				)

				// +kubebuilder:scaffold:e2e-metrics-webhooks-readiness

				By(
					"creating the curl-metrics pod to access the metrics endpoint",
				)

				cmd = exec.Command(
					"kubectl",
					"run",
					"curl-metrics",
					"--restart=Never",
					"--namespace",
					namespace,
					"--image=curlimages/curl:latest",
					"--overrides",
					fmt.Sprintf(
						`{
							"spec": {
								"containers": [{
									"name": "curl",
									"image": "curlimages/curl:latest",
									"command": ["/bin/sh", "-c"],
									"args": [
										"for i in $(seq 1 30); do curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics && exit 0 || sleep 2; done; exit 1"
									],
									"securityContext": {
										"readOnlyRootFilesystem": true,
										"allowPrivilegeEscalation": false,
										"capabilities": {
											"drop": ["ALL"]
										},
										"runAsNonRoot": true,
										"runAsUser": 1000,
										"seccompProfile": {
											"type": "RuntimeDefault"
										}
									}
								}],
								"serviceAccountName": "%s"
							}
						}`,
						token,
						metricsServiceName,
						namespace,
						serviceAccountName,
					),
				)

				_, err = utils.Run(cmd)

				Expect(err).NotTo(
					HaveOccurred(),
				)

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"pods",
							"curl-metrics",
							"-o",
							"jsonpath={.status.phase}",
							"-n",
							namespace,
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						g.Expect(output).To(
							Equal("Succeeded"),
						)
					},
					5*time.Minute,
				).Should(
					Succeed(),
				)

				Eventually(
					func(g Gomega) {
						metricsOutput, err :=
							getMetricsOutput()

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						g.Expect(
							metricsOutput,
						).To(
							ContainSubstring(
								"< HTTP/1.1 200 OK",
							),
						)
					},
					2*time.Minute,
				).Should(
					Succeed(),
				)
			},
		)

		It(
			"should reconcile a TrussiumRuntime into managed Kubernetes resources",
			func() {
				runtimeManifest :=
					fmt.Sprintf(`
apiVersion: runtime.trussium.io/v1alpha1
kind: TrussiumRuntime
metadata:
  name: %s
  namespace: %s
spec:
  image:
    repository: %s
    tag: %s
    pullPolicy: IfNotPresent
  replicas: 0
  provider:
    type: ollama
    model: %s
    baseURL: %s
  service:
    type: ClusterIP
    port: 9000
`,
						runtimeResourceName,
						namespace,
						runtimeImageRepository,
						initialRuntimeImageTag,
						initialRuntimeModel,
						initialRuntimeProviderBaseURL,
					)

				cmd := exec.Command(
					"kubectl",
					"apply",
					"-f",
					"-",
				)

				cmd.Stdin =
					strings.NewReader(
						runtimeManifest,
					)

				_, err := utils.Run(cmd)

				Expect(err).NotTo(
					HaveOccurred(),
				)

				managedResources := []string{
					"configmap",
					"serviceaccount",
					"service",
					"deployment",
					"poddisruptionbudget",
				}

				for _, resourceType := range managedResources {
					resourceType := resourceType

					Eventually(
						func(g Gomega) {
							cmd := exec.Command(
								"kubectl",
								"get",
								resourceType,
								runtimeResourceName,
								"-n",
								namespace,
							)

							_, err :=
								utils.Run(cmd)

							g.Expect(err).NotTo(
								HaveOccurred(),
							)
						},
					).Should(
						Succeed(),
					)
				}

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"deployment",
							runtimeResourceName,
							"-n",
							namespace,
							"-o",
							"jsonpath={.spec.replicas}",
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						g.Expect(output).To(
							Equal("0"),
						)
					},
				).Should(
					Succeed(),
				)

				for _, resourceType := range managedResources {
					resourceType := resourceType

					Eventually(
						func(g Gomega) {
							cmd := exec.Command(
								"kubectl",
								"get",
								resourceType,
								runtimeResourceName,
								"-n",
								namespace,
								"-o",
								"jsonpath={.metadata.ownerReferences[0].kind}/{.metadata.ownerReferences[0].name}",
							)

							output, err :=
								utils.Run(cmd)

							g.Expect(err).NotTo(
								HaveOccurred(),
							)

							g.Expect(output).To(
								Equal(
									"TrussiumRuntime/" +
										runtimeResourceName,
								),
							)
						},
					).Should(
						Succeed(),
					)
				}
			},
		)

		It(
			"should run a real Trussium runtime pod and satisfy its health probes",
			func() {
				cmd := exec.Command(
					"kubectl",
					"patch",
					"trussiumruntime",
					runtimeResourceName,
					"-n",
					namespace,
					"--type=merge",
					"-p",
					`{"spec":{"replicas":1}}`,
				)

				_, err := utils.Run(cmd)

				Expect(err).NotTo(
					HaveOccurred(),
				)

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"deployment",
							runtimeResourceName,
							"-n",
							namespace,
							"-o",
							"jsonpath={.status.availableReplicas}",
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						g.Expect(output).To(
							Equal("1"),
						)
					},
					5*time.Minute,
					2*time.Second,
				).Should(
					Succeed(),
				)

				var runtimePodName string

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"pods",
							"-n",
							namespace,
							"-l",
							fmt.Sprintf(
								"app.kubernetes.io/instance=%s",
								runtimeResourceName,
							),
							"-o",
							"jsonpath={.items[0].metadata.name}",
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						g.Expect(output).NotTo(
							BeEmpty(),
						)

						runtimePodName =
							output
					},
				).Should(
					Succeed(),
				)

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"pod",
							runtimePodName,
							"-n",
							namespace,
							"-o",
							"jsonpath={.status.conditions[?(@.type=='Ready')].status}",
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						g.Expect(output).To(
							Equal("True"),
						)
					},
					5*time.Minute,
					2*time.Second,
				).Should(
					Succeed(),
				)

				cmd = exec.Command(
					"kubectl",
					"get",
					"deployment",
					runtimeResourceName,
					"-n",
					namespace,
					"-o",
					"jsonpath={.spec.template.spec.containers[?(@.name=='trussium')].image}",
				)

				output, err :=
					utils.Run(cmd)

				Expect(err).NotTo(
					HaveOccurred(),
				)

				Expect(output).To(
					Equal(runtimeImage),
				)

				probePaths := map[string]string{
					"startupProbe":   "/health/live",
					"livenessProbe":  "/health/live",
					"readinessProbe": "/health/ready",
				}

				for probe, expectedPath := range probePaths {
					cmd = exec.Command(
						"kubectl",
						"get",
						"deployment",
						runtimeResourceName,
						"-n",
						namespace,
						"-o",
						fmt.Sprintf(
							"jsonpath={.spec.template.spec.containers[?(@.name=='trussium')].%s.httpGet.path}",
							probe,
						),
					)

					output, err =
						utils.Run(cmd)

					Expect(err).NotTo(
						HaveOccurred(),
					)

					Expect(output).To(
						Equal(expectedPath),
					)

					cmd = exec.Command(
						"kubectl",
						"get",
						"deployment",
						runtimeResourceName,
						"-n",
						namespace,
						"-o",
						fmt.Sprintf(
							"jsonpath={.spec.template.spec.containers[?(@.name=='trussium')].%s.httpGet.port}",
							probe,
						),
					)

					output, err =
						utils.Run(cmd)

					Expect(err).NotTo(
						HaveOccurred(),
					)

					Expect(output).To(
						Equal("http"),
					)
				}
			},
		)

		It(
			"should roll the runtime pod when provider configuration changes",
			func() {
				var previousPodName string

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"pods",
							"-n",
							namespace,
							"-l",
							fmt.Sprintf(
								"app.kubernetes.io/instance=%s",
								runtimeResourceName,
							),
							"-o",
							"go-template={{ range .items }}"+
								"{{ if not .metadata.deletionTimestamp }}"+
								"{{ .metadata.name }}"+
								"{{ \"\\n\" }}"+
								"{{ end }}"+
								"{{ end }}",
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						podNames :=
							utils.GetNonEmptyLines(
								output,
							)

						g.Expect(podNames).To(
							HaveLen(1),
						)

						previousPodName =
							podNames[0]
					},
				).Should(
					Succeed(),
				)

				cmd := exec.Command(
					"kubectl",
					"get",
					"deployment",
					runtimeResourceName,
					"-n",
					namespace,
					"-o",
					fmt.Sprintf(
						"jsonpath={.spec.template.metadata.annotations.%s}",
						strings.ReplaceAll(
							runtimeConfigChecksumAnnotation,
							".",
							"\\.",
						),
					),
				)

				previousChecksum, err :=
					utils.Run(cmd)

				Expect(err).NotTo(
					HaveOccurred(),
				)

				Expect(
					previousChecksum,
				).NotTo(
					BeEmpty(),
				)

				patch := fmt.Sprintf(
					`{"spec":{"provider":{"baseURL":"%s"}}}`,
					updatedRuntimeProviderBaseURL,
				)

				cmd = exec.Command(
					"kubectl",
					"patch",
					"trussiumruntime",
					runtimeResourceName,
					"-n",
					namespace,
					"--type=merge",
					"-p",
					patch,
				)

				_, err = utils.Run(cmd)

				Expect(err).NotTo(
					HaveOccurred(),
				)

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"configmap",
							runtimeResourceName,
							"-n",
							namespace,
							"-o",
							fmt.Sprintf(
								"jsonpath={.data.%s}",
								strings.ReplaceAll(
									runtimeProviderBaseURLEnvironment,
									".",
									"\\.",
								),
							),
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						g.Expect(output).To(
							Equal(
								updatedRuntimeProviderBaseURL,
							),
						)
					},
				).Should(
					Succeed(),
				)

				var updatedChecksum string

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"deployment",
							runtimeResourceName,
							"-n",
							namespace,
							"-o",
							fmt.Sprintf(
								"jsonpath={.spec.template.metadata.annotations.%s}",
								strings.ReplaceAll(
									runtimeConfigChecksumAnnotation,
									".",
									"\\.",
								),
							),
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						g.Expect(output).NotTo(
							BeEmpty(),
						)

						g.Expect(output).NotTo(
							Equal(
								previousChecksum,
							),
						)

						updatedChecksum =
							output
					},
				).Should(
					Succeed(),
				)

				Expect(
					updatedChecksum,
				).NotTo(
					Equal(previousChecksum),
				)

				cmd = exec.Command(
					"kubectl",
					"get",
					"deployment",
					runtimeResourceName,
					"-n",
					namespace,
					"-o",
					"jsonpath={.spec.template.spec.containers[?(@.name=='trussium')].image}",
				)

				output, err :=
					utils.Run(cmd)

				Expect(err).NotTo(
					HaveOccurred(),
				)

				Expect(output).To(
					Equal(runtimeImage),
				)

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"deployment",
							runtimeResourceName,
							"-n",
							namespace,
							"-o",
							"jsonpath={.metadata.generation}/{.status.observedGeneration}/{.status.updatedReplicas}/{.status.readyReplicas}/{.status.availableReplicas}",
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						parts :=
							strings.Split(
								output,
								"/",
							)

						g.Expect(parts).To(
							HaveLen(5),
						)

						g.Expect(parts[1]).To(
							Equal(parts[0]),
						)

						g.Expect(parts[2]).To(
							Equal("1"),
						)

						g.Expect(parts[3]).To(
							Equal("1"),
						)

						g.Expect(parts[4]).To(
							Equal("1"),
						)
					},
					5*time.Minute,
					2*time.Second,
				).Should(
					Succeed(),
				)

				var replacementPodName string

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"pods",
							"-n",
							namespace,
							"-l",
							fmt.Sprintf(
								"app.kubernetes.io/instance=%s",
								runtimeResourceName,
							),
							"-o",
							"go-template={{ range .items }}"+
								"{{ if not .metadata.deletionTimestamp }}"+
								"{{ .metadata.name }}"+
								"{{ \"\\n\" }}"+
								"{{ end }}"+
								"{{ end }}",
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						podNames :=
							utils.GetNonEmptyLines(
								output,
							)

						replacementPodName = ""

						for _, podName := range podNames {
							if podName !=
								previousPodName {
								replacementPodName =
									podName

								break
							}
						}

						g.Expect(
							replacementPodName,
						).NotTo(
							BeEmpty(),
						)
					},
					5*time.Minute,
					2*time.Second,
				).Should(
					Succeed(),
				)

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"pod",
							replacementPodName,
							"-n",
							namespace,
							"-o",
							"jsonpath={.status.conditions[?(@.type=='Ready')].status}",
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						g.Expect(output).To(
							Equal("True"),
						)
					},
					5*time.Minute,
					2*time.Second,
				).Should(
					Succeed(),
				)

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"pod",
							previousPodName,
							"-n",
							namespace,
							"--ignore-not-found",
							"-o",
							"name",
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						g.Expect(
							strings.TrimSpace(
								output,
							),
						).To(
							BeEmpty(),
						)
					},
					5*time.Minute,
					2*time.Second,
				).Should(
					Succeed(),
				)

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"pods",
							"-n",
							namespace,
							"-l",
							fmt.Sprintf(
								"app.kubernetes.io/instance=%s",
								runtimeResourceName,
							),
							"-o",
							"go-template={{ range .items }}"+
								"{{ if not .metadata.deletionTimestamp }}"+
								"{{ .metadata.name }}"+
								"{{ \"\\n\" }}"+
								"{{ end }}"+
								"{{ end }}",
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						podNames :=
							utils.GetNonEmptyLines(
								output,
							)

						g.Expect(podNames).To(
							ConsistOf(
								replacementPodName,
							),
						)
					},
					5*time.Minute,
					2*time.Second,
				).Should(
					Succeed(),
				)
			},
		)

		It(
			"should complete a real Trussium runtime image upgrade",
			func() {
				var previousPodName string

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"pods",
							"-n",
							namespace,
							"-l",
							fmt.Sprintf(
								"app.kubernetes.io/instance=%s",
								runtimeResourceName,
							),
							"-o",
							"go-template={{ range .items }}"+
								"{{ if not .metadata.deletionTimestamp }}"+
								"{{ .metadata.name }}"+
								"{{ \"\\n\" }}"+
								"{{ end }}"+
								"{{ end }}",
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						podNames :=
							utils.GetNonEmptyLines(
								output,
							)

						g.Expect(podNames).To(
							HaveLen(1),
						)

						previousPodName =
							podNames[0]
					},
				).Should(
					Succeed(),
				)

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"trussiumruntime",
							runtimeResourceName,
							"-n",
							namespace,
							"-o",
							"jsonpath={.status.lastSuccessfulImage}",
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						g.Expect(output).To(
							Equal(runtimeImage),
						)
					},
				).Should(
					Succeed(),
				)

				patch := fmt.Sprintf(
					`{"spec":{"image":{"tag":"%s"}}}`,
					upgradedRuntimeImageTag,
				)

				cmd := exec.Command(
					"kubectl",
					"patch",
					"trussiumruntime",
					runtimeResourceName,
					"-n",
					namespace,
					"--type=merge",
					"-p",
					patch,
				)

				_, err := utils.Run(cmd)

				Expect(err).NotTo(
					HaveOccurred(),
				)

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"deployment",
							runtimeResourceName,
							"-n",
							namespace,
							"-o",
							"jsonpath={.spec.template.spec.containers[?(@.name=='trussium')].image}",
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						g.Expect(output).To(
							Equal(
								upgradedRuntimeImage,
							),
						)
					},
				).Should(
					Succeed(),
				)

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"events",
							"-n",
							namespace,
							"--field-selector",
							fmt.Sprintf(
								"involvedObject.name=%s,reason=%s",
								runtimeResourceName,
								runtimeUpgradeStartedEvent,
							),
							"-o",
							"jsonpath={.items[*].reason}",
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						g.Expect(
							strings.Fields(
								output,
							),
						).To(
							ContainElement(
								runtimeUpgradeStartedEvent,
							),
						)
					},
				).Should(
					Succeed(),
				)

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"deployment",
							runtimeResourceName,
							"-n",
							namespace,
							"-o",
							"jsonpath={.status.availableReplicas}/{.status.updatedReplicas}/{.status.replicas}",
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						g.Expect(output).To(
							Equal("1/1/1"),
						)
					},
					5*time.Minute,
					2*time.Second,
				).Should(
					Succeed(),
				)

				var upgradedPodName string

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"pods",
							"-n",
							namespace,
							"-l",
							fmt.Sprintf(
								"app.kubernetes.io/instance=%s",
								runtimeResourceName,
							),
							"-o",
							"go-template={{ range .items }}"+
								"{{ if not .metadata.deletionTimestamp }}"+
								"{{ .metadata.name }}"+
								"{{ \"\\n\" }}"+
								"{{ end }}"+
								"{{ end }}",
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						podNames :=
							utils.GetNonEmptyLines(
								output,
							)

						for _, podName := range podNames {
							if podName !=
								previousPodName {
								upgradedPodName =
									podName

								break
							}
						}

						g.Expect(
							upgradedPodName,
						).NotTo(
							BeEmpty(),
						)
					},
					5*time.Minute,
					2*time.Second,
				).Should(
					Succeed(),
				)

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"pod",
							upgradedPodName,
							"-n",
							namespace,
							"-o",
							"jsonpath={.status.conditions[?(@.type=='Ready')].status}",
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						g.Expect(output).To(
							Equal("True"),
						)
					},
					5*time.Minute,
					2*time.Second,
				).Should(
					Succeed(),
				)

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"trussiumruntime",
							runtimeResourceName,
							"-n",
							namespace,
							"-o",
							"jsonpath={.status.conditions[?(@.type=='Upgrading')].status}/{.status.conditions[?(@.type=='Upgrading')].reason}",
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						g.Expect(output).To(
							Equal(
								"False/" +
									upgradeCompleteReason,
							),
						)
					},
					5*time.Minute,
					2*time.Second,
				).Should(
					Succeed(),
				)

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"trussiumruntime",
							runtimeResourceName,
							"-n",
							namespace,
							"-o",
							"jsonpath={.status.desiredImage}/{.status.currentImage}/{.status.lastSuccessfulImage}",
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						expected :=
							upgradedRuntimeImage +
								"/" +
								upgradedRuntimeImage +
								"/" +
								upgradedRuntimeImage

						g.Expect(output).To(
							Equal(expected),
						)
					},
				).Should(
					Succeed(),
				)

				Eventually(
					func(g Gomega) {
						cmd := exec.Command(
							"kubectl",
							"get",
							"events",
							"-n",
							namespace,
							"--field-selector",
							fmt.Sprintf(
								"involvedObject.name=%s,reason=%s",
								runtimeResourceName,
								runtimeUpgradeCompletedEvent,
							),
							"-o",
							"jsonpath={.items[*].reason}",
						)

						output, err :=
							utils.Run(cmd)

						g.Expect(err).NotTo(
							HaveOccurred(),
						)

						g.Expect(
							strings.Fields(
								output,
							),
						).To(
							ContainElement(
								runtimeUpgradeCompletedEvent,
							),
						)
					},
				).Should(
					Succeed(),
				)
			},
		)

		// +kubebuilder:scaffold:e2e-webhooks-checks
	})
})

func serviceAccountToken() (
	string,
	error,
) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	secretName := fmt.Sprintf(
		"%s-token-request",
		serviceAccountName,
	)

	tokenRequestFile := filepath.Join(
		"/tmp",
		secretName,
	)

	err := os.WriteFile(
		tokenRequestFile,
		[]byte(tokenRequestRawString),
		os.FileMode(0o644),
	)
	if err != nil {
		return "", err
	}

	var out string

	Eventually(
		func(g Gomega) {
			cmd := exec.Command(
				"kubectl",
				"create",
				"--raw",
				fmt.Sprintf(
					"/api/v1/namespaces/%s/serviceaccounts/%s/token",
					namespace,
					serviceAccountName,
				),
				"-f",
				tokenRequestFile,
			)

			output, err :=
				cmd.CombinedOutput()

			g.Expect(err).NotTo(
				HaveOccurred(),
			)

			var token tokenRequest

			err = json.Unmarshal(
				output,
				&token,
			)

			g.Expect(err).NotTo(
				HaveOccurred(),
			)

			out =
				token.Status.Token
		},
	).Should(
		Succeed(),
	)

	return out, err
}

func getMetricsOutput() (
	string,
	error,
) {
	cmd := exec.Command(
		"kubectl",
		"logs",
		"curl-metrics",
		"-n",
		namespace,
	)

	return utils.Run(cmd)
}

type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}
