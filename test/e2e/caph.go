/*
Copyright 2022 The Kubernetes Authors.

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
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
)

// CaphClusterDeploymentSpecInput is the input for CaphClusterDeploymentSpec.
type CaphClusterDeploymentSpecInput struct {
	E2EConfig             *clusterctl.E2EConfig
	ClusterctlConfigPath  string
	BootstrapClusterProxy framework.ClusterProxy
	ArtifactFolder        string
	SkipCleanup           bool
	Flavor                string
}

// CaphClusterDeploymentSpec implements a test that verifies that MachineDeployment rolling updates are successful.
func CaphClusterDeploymentSpec(ctx context.Context, inputGetter func() CaphClusterDeploymentSpecInput) {
	var (
		specName         = "caph"
		input            CaphClusterDeploymentSpecInput
		namespace        *corev1.Namespace
		cancelWatches    context.CancelFunc
		clusterResources *clusterctl.ApplyClusterTemplateAndWaitResult
		clusterName      string
	)

	ginkgo.BeforeEach(func() {
		gomega.Expect(ctx).NotTo(gomega.BeNil(), "ctx is required for %s spec", specName)
		input = inputGetter()
		gomega.Expect(input.E2EConfig).ToNot(gomega.BeNil(), "Invalid argument. input.E2EConfig can't be nil when calling %s spec", specName)
		gomega.Expect(input.ClusterctlConfigPath).To(gomega.BeAnExistingFile(), "Invalid argument. input.ClusterctlConfigPath must be an existing file when calling %s spec", specName)
		gomega.Expect(input.BootstrapClusterProxy).ToNot(gomega.BeNil(), "Invalid argument. input.BootstrapClusterProxy can't be nil when calling %s spec", specName)
		gomega.Expect(os.MkdirAll(input.ArtifactFolder, 0750)).To(gomega.Succeed(), "Invalid argument. input.ArtifactFolder can't be created for %s spec", specName)
		gomega.Expect(input.E2EConfig.Variables).To(gomega.HaveKey(KubernetesVersion))
		gomega.Expect(input.E2EConfig.Variables).To(HaveValidVersion(input.E2EConfig.GetVariable(KubernetesVersion)))

		// Setup a Namespace where to host objects for this spec and create a watcher for the namespace events.
		namespace, cancelWatches = setupSpecNamespace(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder)
		clusterResources = new(clusterctl.ApplyClusterTemplateAndWaitResult)

		clusterName = fmt.Sprintf("%s-%s", specName, util.RandomString(6))
	})

	ginkgo.It("Should successfully upgrade Machines upon changes in relevant MachineDeployment fields", func() {
		ginkgo.By("Creating a workload cluster")
		clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
			ClusterProxy: input.BootstrapClusterProxy,
			ConfigCluster: clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(input.ArtifactFolder, "clusters", input.BootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     input.ClusterctlConfigPath,
				KubeconfigPath:           input.BootstrapClusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
				Flavor:                   input.Flavor,
				Namespace:                namespace.Name,
				ClusterName:              clusterName,
				KubernetesVersion:        input.E2EConfig.GetVariable(KubernetesVersion),
				ControlPlaneMachineCount: pointer.Int64Ptr(1),
				WorkerMachineCount:       pointer.Int64Ptr(1),
			},
			WaitForClusterIntervals:      input.E2EConfig.GetIntervals(specName, "wait-cluster"),
			WaitForControlPlaneIntervals: input.E2EConfig.GetIntervals(specName, "wait-control-plane"),
			WaitForMachineDeployments:    input.E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
		}, clusterResources)

		ginkgo.By("Scaling worker node to 3")
		clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
			ClusterProxy: input.BootstrapClusterProxy,
			ConfigCluster: clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(input.ArtifactFolder, "clusters", input.BootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     input.ClusterctlConfigPath,
				KubeconfigPath:           input.BootstrapClusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
				Flavor:                   input.Flavor,
				Namespace:                namespace.Name,
				ClusterName:              clusterName,
				KubernetesVersion:        input.E2EConfig.GetVariable(KubernetesVersion),
				ControlPlaneMachineCount: pointer.Int64Ptr(1),
				WorkerMachineCount:       pointer.Int64Ptr(3),
			},
			WaitForClusterIntervals:      input.E2EConfig.GetIntervals(specName, "wait-cluster"),
			WaitForControlPlaneIntervals: input.E2EConfig.GetIntervals(specName, "wait-control-plane"),
			WaitForMachineDeployments:    input.E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
		}, clusterResources)

		ginkgo.By("Scaling control planes to 3")
		clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
			ClusterProxy: input.BootstrapClusterProxy,
			ConfigCluster: clusterctl.ConfigClusterInput{
				LogFolder:                filepath.Join(input.ArtifactFolder, "clusters", input.BootstrapClusterProxy.GetName()),
				ClusterctlConfigPath:     input.ClusterctlConfigPath,
				KubeconfigPath:           input.BootstrapClusterProxy.GetKubeconfigPath(),
				InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
				Flavor:                   input.Flavor,
				Namespace:                namespace.Name,
				ClusterName:              clusterName,
				KubernetesVersion:        input.E2EConfig.GetVariable(KubernetesVersion),
				ControlPlaneMachineCount: pointer.Int64Ptr(3),
				WorkerMachineCount:       pointer.Int64Ptr(3),
			},
			WaitForClusterIntervals:      input.E2EConfig.GetIntervals(specName, "wait-cluster"),
			WaitForControlPlaneIntervals: input.E2EConfig.GetIntervals(specName, "wait-control-plane"),
			WaitForMachineDeployments:    input.E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
		}, clusterResources)

		ginkgo.It("Should successfully trigger machine deployment remediation", func() {
			ginkgo.By("Creating a workload cluster")

			clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
				ClusterProxy: input.BootstrapClusterProxy,
				ConfigCluster: clusterctl.ConfigClusterInput{
					LogFolder:                filepath.Join(input.ArtifactFolder, "clusters", input.BootstrapClusterProxy.GetName()),
					ClusterctlConfigPath:     input.ClusterctlConfigPath,
					KubeconfigPath:           input.BootstrapClusterProxy.GetKubeconfigPath(),
					InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
					Flavor:                   pointer.StringDeref(input.MDFlavor, "md-remediation"),
					Namespace:                namespace.Name,
					ClusterName:              fmt.Sprintf("%s-%s", specName, util.RandomString(6)),
					KubernetesVersion:        input.E2EConfig.GetVariable(KubernetesVersion),
					ControlPlaneMachineCount: pointer.Int64Ptr(1),
					WorkerMachineCount:       pointer.Int64Ptr(1),
				},
				WaitForClusterIntervals:      input.E2EConfig.GetIntervals(specName, "wait-cluster"),
				WaitForControlPlaneIntervals: input.E2EConfig.GetIntervals(specName, "wait-control-plane"),
				WaitForMachineDeployments:    input.E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
			}, clusterResources)

			ginkgo.By("Setting a machine unhealthy and wait for MachineDeployment remediation")
			framework.DiscoverMachineHealthChecksAndWaitForRemediation(ctx, framework.DiscoverMachineHealthCheckAndWaitForRemediationInput{
				ClusterProxy:              input.BootstrapClusterProxy,
				Cluster:                   clusterResources.Cluster,
				WaitForMachineRemediation: input.E2EConfig.GetIntervals(specName, "wait-machine-remediation"),
			})
		})

		ginkgo.It("Should successfully trigger KCP remediation", func() {
			ginkgo.By("Creating a workload cluster")
			clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
				ClusterProxy: input.BootstrapClusterProxy,
				ConfigCluster: clusterctl.ConfigClusterInput{
					LogFolder:                filepath.Join(input.ArtifactFolder, "clusters", input.BootstrapClusterProxy.GetName()),
					ClusterctlConfigPath:     input.ClusterctlConfigPath,
					KubeconfigPath:           input.BootstrapClusterProxy.GetKubeconfigPath(),
					InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
					Flavor:                   pointer.StringDeref(input.KCPFlavor, "kcp-remediation"),
					Namespace:                namespace.Name,
					ClusterName:              fmt.Sprintf("%s-%s", specName, util.RandomString(6)),
					KubernetesVersion:        input.E2EConfig.GetVariable(KubernetesVersion),
					ControlPlaneMachineCount: pointer.Int64Ptr(3),
					WorkerMachineCount:       pointer.Int64Ptr(1),
				},
				WaitForClusterIntervals:      input.E2EConfig.GetIntervals(specName, "wait-cluster"),
				WaitForControlPlaneIntervals: input.E2EConfig.GetIntervals(specName, "wait-control-plane"),
				WaitForMachineDeployments:    input.E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
			}, clusterResources)

			By("Setting a machine unhealthy and wait for KubeadmControlPlane remediation")
			framework.DiscoverMachineHealthChecksAndWaitForRemediation(ctx, framework.DiscoverMachineHealthCheckAndWaitForRemediationInput{
				ClusterProxy:              input.BootstrapClusterProxy,
				Cluster:                   clusterResources.Cluster,
				WaitForMachineRemediation: input.E2EConfig.GetIntervals(specName, "wait-machine-remediation"),
			})
		})

		ginkgo.It("A node should be forcefully removed if it cannot be drained in time", func() {
			ginkgo.By("Creating a workload cluster")
			controlPlaneReplicas := 3
			clusterctl.ApplyClusterTemplateAndWait(ctx, clusterctl.ApplyClusterTemplateAndWaitInput{
				ClusterProxy: input.BootstrapClusterProxy,
				ConfigCluster: clusterctl.ConfigClusterInput{
					LogFolder:                filepath.Join(input.ArtifactFolder, "clusters", input.BootstrapClusterProxy.GetName()),
					ClusterctlConfigPath:     input.ClusterctlConfigPath,
					KubeconfigPath:           input.BootstrapClusterProxy.GetKubeconfigPath(),
					InfrastructureProvider:   clusterctl.DefaultInfrastructureProvider,
					Flavor:                   pointer.StringDeref(input.Flavor, "node-drain"),
					Namespace:                namespace.Name,
					ClusterName:              fmt.Sprintf("%s-%s", specName, util.RandomString(6)),
					KubernetesVersion:        input.E2EConfig.GetVariable(KubernetesVersion),
					ControlPlaneMachineCount: pointer.Int64Ptr(int64(controlPlaneReplicas)),
					WorkerMachineCount:       pointer.Int64Ptr(1),
				},
				WaitForClusterIntervals:      input.E2EConfig.GetIntervals(specName, "wait-cluster"),
				WaitForControlPlaneIntervals: input.E2EConfig.GetIntervals(specName, "wait-control-plane"),
				WaitForMachineDeployments:    input.E2EConfig.GetIntervals(specName, "wait-worker-nodes"),
			}, clusterResources)
			cluster := clusterResources.Cluster
			controlplane = clusterResources.ControlPlane
			machineDeployments = clusterResources.MachineDeployments
			ginkgo.Expect(machineDeployments[0].Spec.Replicas).To(Equal(pointer.Int32Ptr(1)))

			ginkgo.By("Add a deployment with unevictable pods and podDisruptionBudget to the workload cluster. The deployed pods cannot be evicted in the node draining process.")
			workloadClusterProxy := input.BootstrapClusterProxy.GetWorkloadCluster(ctx, cluster.Namespace, cluster.Name)
			framework.DeployUnevictablePod(ctx, framework.DeployUnevictablePodInput{
				WorkloadClusterProxy:               workloadClusterProxy,
				DeploymentName:                     fmt.Sprintf("%s-%s", "unevictable-pod", util.RandomString(3)),
				Namespace:                          namespace.Name + "-unevictable-workload",
				WaitForDeploymentAvailableInterval: input.E2EConfig.GetIntervals(specName, "wait-deployment-available"),
			})

			ginkgo.By("Scale the machinedeployment down to zero. If we didn't have the NodeDrainTimeout duration, the node drain process would block this operator.")
			// Because all the machines of a machinedeployment can be deleted at the same time, so we only prepare the interval for 1 replica.
			nodeDrainTimeoutMachineDeploymentInterval := getDrainAndDeleteInterval(input.E2EConfig.GetIntervals(specName, "wait-machine-deleted"), machineDeployments[0].Spec.Template.Spec.NodeDrainTimeout, 1)
			for _, md := range machineDeployments {
				framework.ScaleAndWaitMachineDeployment(ctx, framework.ScaleAndWaitMachineDeploymentInput{
					ClusterProxy:              input.BootstrapClusterProxy,
					Cluster:                   cluster,
					MachineDeployment:         md,
					WaitForMachineDeployments: nodeDrainTimeoutMachineDeploymentInterval,
					Replicas:                  0,
				})
			}

			ginkgo.By("Deploy deployment with unevictable pods on control plane nodes.")
			framework.DeployUnevictablePod(ctx, framework.DeployUnevictablePodInput{
				WorkloadClusterProxy:               workloadClusterProxy,
				ControlPlane:                       controlplane,
				DeploymentName:                     fmt.Sprintf("%s-%s", "unevictable-pod", util.RandomString(3)),
				Namespace:                          namespace.Name + "-unevictable-workload",
				WaitForDeploymentAvailableInterval: input.E2EConfig.GetIntervals(specName, "wait-deployment-available"),
			})

			ginkgo.By("Scale down the controlplane of the workload cluster and make sure that nodes running workload can be deleted even the draining process is blocked.")
			// When we scale down the KCP, controlplane machines are by default deleted one by one, so it requires more time.
			nodeDrainTimeoutKCPInterval := getDrainAndDeleteInterval(input.E2EConfig.GetIntervals(specName, "wait-machine-deleted"), controlplane.Spec.MachineTemplate.NodeDrainTimeout, controlPlaneReplicas)
			framework.ScaleAndWaitControlPlane(ctx, framework.ScaleAndWaitControlPlaneInput{
				ClusterProxy:        input.BootstrapClusterProxy,
				Cluster:             cluster,
				ControlPlane:        controlplane,
				Replicas:            1,
				WaitForControlPlane: nodeDrainTimeoutKCPInterval,
			})
		})

		ginkgo.By("PASSED!")
	})

	ginkgo.AfterEach(func() {
		// Dumps all the resources in the spec namespace, then cleanups the cluster object and the spec namespace itself.
		dumpSpecResourcesAndCleanup(ctx, specName, input.BootstrapClusterProxy, input.ArtifactFolder, namespace, cancelWatches, clusterResources.Cluster, input.E2EConfig.GetIntervals, input.SkipCleanup)
		redactLogs(input.E2EConfig.GetVariable)
	})
}
