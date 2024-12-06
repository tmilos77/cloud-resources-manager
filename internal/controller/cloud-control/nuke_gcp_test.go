package cloudcontrol

import (
	"fmt"
	cloudcontrolv1beta1 "github.com/kyma-project/cloud-manager/api/cloud-control/v1beta1"
	"github.com/kyma-project/cloud-manager/pkg/common"
	"github.com/kyma-project/cloud-manager/pkg/common/actions/focal"
	"github.com/kyma-project/cloud-manager/pkg/composed"
	kcpiprange "github.com/kyma-project/cloud-manager/pkg/kcp/iprange"
	kcpnetwork "github.com/kyma-project/cloud-manager/pkg/kcp/network"
	kcpnfsinstance "github.com/kyma-project/cloud-manager/pkg/kcp/nfsinstance"
	gcpclient "github.com/kyma-project/cloud-manager/pkg/kcp/provider/gcp/client"
	scopePkg "github.com/kyma-project/cloud-manager/pkg/kcp/scope"
	. "github.com/kyma-project/cloud-manager/pkg/testinfra/dsl"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/api/file/v1"
)

var _ = Describe("Feature: Cleanup orphan resources", func() {

	It("Scenario: KCP Nuke deletes GCP provider resources in the Scope", func() {
		const kymaName = "c2467bcb-ee77-46ab-8f68-f4176ed7eb27"

		scope := &cloudcontrolv1beta1.Scope{}
		backupClient, err := infra.GcpMock().FileBackupClientProvider()(infra.Ctx(), "")
		Expect(err).To(Succeed())

		By("Given Scope exists", func() {
			// Tell Scope reconciler to ignore this kymaName
			scopePkg.Ignore.AddName(kymaName)

			Expect(CreateScopeGcp(infra.Ctx(), infra, scope, WithName(kymaName))).
				To(Succeed(), "failed creating scope")
		})

		cmNetwork := cloudcontrolv1beta1.NewNetworkBuilder().
			WithName("ec2c020f-edec-4bb3-8147-190e27e67ffd").
			WithType(cloudcontrolv1beta1.NetworkTypeCloudResources).
			WithScope(kymaName).
			WithCidr(common.DefaultCloudManagerCidr).
			Build()

		By("And Given CM Network exists", func() {
			kcpnetwork.Ignore.AddName(cmNetwork.Name)

			Expect(CreateObj(infra.Ctx(), infra.KCP().Client(), cmNetwork,
				AddFinalizer(cloudcontrolv1beta1.FinalizerName),
			)).To(Succeed(), "failed creating cm network")
		})

		ipRangeName := "e1b0d66d-4990-4ebf-ad2c-25caab3d2002"
		ipRange := &cloudcontrolv1beta1.IpRange{}

		By("And Given IpRange exists", func() {
			kcpiprange.Ignore.AddName(ipRangeName)

			Eventually(CreateKcpIpRange).
				WithArguments(infra.Ctx(), infra.KCP().Client(), ipRange,
					WithName(ipRangeName),
					AddFinalizer(cloudcontrolv1beta1.FinalizerName),
					WithKcpIpRangeNetwork(cmNetwork.Name),
					WithScope(kymaName),
					WithRemoteRef("foo"),
					WithKcpIpRangeSpecCidr(common.DefaultCloudManagerCidr),
				).
				Should(Succeed(), "failed creating IpRange")
		})

		nfsInstanceName := "87104076-d151-4566-8534-40d913a71569"
		nfsInstance := &cloudcontrolv1beta1.NfsInstance{}

		By("And Given NfsInstance exists", func() {
			kcpnfsinstance.Ignore.AddName(nfsInstanceName)

			Expect(CreateNfsInstance(infra.Ctx(), infra.KCP().Client(), nfsInstance,
				WithName(nfsInstanceName),
				AddFinalizer(cloudcontrolv1beta1.FinalizerName),
				WithRemoteRef("foo"),
				WithScope(kymaName),
				WithIpRange(ipRange.Name),
				WithNfsInstanceGcp(scope.Spec.Region),
				AddFinalizer(cloudcontrolv1beta1.FinalizerName),
			)).To(Succeed(), "failed creating NfsInstance")
		})

		filestoreBackupName := "03b42b69-3ef4-44cc-aa44-fd2f1522784e"
		filestoreBackup := &file.Backup{}
		filestoreBackup.Name = filestoreBackupName
		filestoreBackup.Labels = map[string]string{
			gcpclient.ScopeNameKey: scope.Name,
			gcpclient.ManagedByKey: gcpclient.ManagedByValue,
		}
		anotherBackupName := "42720e1d-91f1-4b77-bee7-53e6f20a6853"
		anotherBackup := &file.Backup{}
		anotherBackup.Name = anotherBackupName
		const anotherScopeName = "238bc7ac-7119-4d88-8a46-850819b7c981"
		anotherBackup.Labels = map[string]string{
			gcpclient.ScopeNameKey: anotherScopeName,
			gcpclient.ManagedByKey: gcpclient.ManagedByValue,
		}
		By(" And Given GcpFilestoreBackup exits for the same scope", func() {
			Expect(
				CreateGcpFileBackupDirectly(
					infra.Ctx(),
					backupClient,
					scope.Spec.Scope.Gcp.Project,
					"any-location",
					filestoreBackup,
				)).
				To(Succeed(), "failed creating GcpFilestoreBackup directly")
		})

		By(" And Given another GcpFilestoreBackup exits for another scope", func() {
			Expect(
				CreateGcpFileBackupDirectly(
					infra.Ctx(),
					backupClient,
					scope.Spec.Scope.Gcp.Project,
					"any-location",
					anotherBackup,
				)).
				To(Succeed(), "failed creating another GcpFilestoreBackup directly")
		})
		nuke := &cloudcontrolv1beta1.Nuke{}

		By("When Nuke for the Scope is created", func() {
			Expect(CreateObj(infra.Ctx(), infra.KCP().Client(), nuke,
				WithName("0e5e1799-f627-4237-b6f0-6adea42131f8"),
				WithScope(kymaName),
			)).To(Succeed())
		})

		By("Then Nuke status state is Deleting", func() {
			Eventually(LoadAndCheck).
				WithArguments(infra.Ctx(), infra.KCP().Client(), nuke, NewObjActions(),
					HavingState("Deleting"),
				).
				Should(Succeed())
		})

		resources := map[string]focal.CommonObject{
			"NfsInstance": nfsInstance,
			"IpRange":     ipRange,
			"Network":     cmNetwork,
		}

		for kind, obj := range resources {
			By(fmt.Sprintf("And Then Nuke status resource %s has Deleting status", kind), func() {
				sk := nuke.Status.GetKindNoCreate(kind)
				Expect(sk).NotTo(BeNil())
				Expect(sk.Kind).To(Equal(kind))
				Expect(sk.Objects).To(HaveLen(1))
				Expect(sk.Objects).To(HaveKeyWithValue(obj.GetName(), cloudcontrolv1beta1.NukeResourceStatusDeleting))
			})

			By(fmt.Sprintf("And Then resource %s has deletion timestamp", kind), func() {
				Expect(LoadAndCheck(infra.Ctx(), infra.KCP().Client(), obj, NewObjActions())).
					To(Succeed())
				Expect(obj.GetDeletionTimestamp()).NotTo(BeNil())
			})
		}

		for kind, obj := range resources {
			By(fmt.Sprintf("When resource %s finalizer is removed", kind), func() {
				removed, err := composed.PatchObjRemoveFinalizer(infra.Ctx(), cloudcontrolv1beta1.FinalizerName, obj, infra.KCP().Client())
				Expect(err).To(Succeed())
				Expect(removed).To(BeTrue())
			})

			By(fmt.Sprintf("Then resource %s does not exist", kind), func() {
				Eventually(IsDeleted).
					WithArguments(infra.Ctx(), infra.KCP().Client(), obj).
					Should(Succeed())
			})

			By(fmt.Sprintf("And Then Nuke status resource %s has state Deleted", kind), func() {
				Eventually(func() error {
					if err := LoadAndCheck(infra.Ctx(), infra.KCP().Client(), nuke, NewObjActions()); err != nil {
						return err
					}
					sk := nuke.Status.GetKindNoCreate(kind)
					if sk == nil {
						return fmt.Errorf("kind %s not found in Nuke status", kind)
					}
					actual := sk.Objects[obj.GetName()]
					if actual == cloudcontrolv1beta1.NukeResourceStatusDeleted {
						return nil
					}
					return fmt.Errorf("expected resource %s/%s to have status Deleted, but found %s", kind, obj.GetName(), actual)
				}).Should(Succeed())
			})
		}

		providerResources := map[string]file.Backup{
			"FilestoreBackup": *filestoreBackup,
		}

		for kind, backup := range providerResources {
			By(fmt.Sprintf("And Then Provider Nuke status resource %s has Deleting status", kind), func() {
				sk := nuke.Status.GetKindNoCreate(kind)
				Expect(sk).NotTo(BeNil())
				Expect(sk.Kind).To(Equal(kind))
				Expect(sk.Objects).To(HaveLen(1))
				Expect(sk.Objects).To(Or(
					HaveKeyWithValue(backup.Name, cloudcontrolv1beta1.NukeResourceStatusDeleting),
					HaveKeyWithValue(backup.Name, cloudcontrolv1beta1.NukeResourceStatusDeleted)))
			})
		}

		for kind, backup := range providerResources {
			By(fmt.Sprintf("And Then provider resource %s does not exist", kind), func() {
				Eventually(func() error {
					backupsOnGcp, err := ListGcpFileBackups(infra.Ctx(), backupClient, scope.Spec.Scope.Gcp.Project, scope.Name)
					if err != nil {
						return err
					}
					Expect(backupsOnGcp).To(HaveLen(0))
					return nil
				}).Should(Succeed())
			})

			By(fmt.Sprintf("And Then Nuke status resource %s has state Deleted", kind), func() {
				Eventually(func() error {
					if err := LoadAndCheck(infra.Ctx(), infra.KCP().Client(), nuke, NewObjActions()); err != nil {
						return err
					}
					sk := nuke.Status.GetKindNoCreate(kind)
					if sk == nil {
						return fmt.Errorf("kind %s not found in Nuke status", kind)
					}
					actual := sk.Objects[backup.Name]
					if actual == cloudcontrolv1beta1.NukeResourceStatusDeleted {
						return nil
					}
					return fmt.Errorf("expected resource %s/%s to have status Deleted, but found %s", kind, backup.Name, actual)
				}).Should(Succeed())
			})
		}
		By("And Then other Backup for other Scope still exists", func() {
			Eventually(func() error {
				backupsOnGcp, err := ListGcpFileBackups(infra.Ctx(), backupClient, scope.Spec.Scope.Gcp.Project, anotherScopeName)
				if err != nil {
					return err
				}
				Expect(backupsOnGcp).To(HaveLen(1))
				return nil
			}).Should(Succeed())
		})

		By("And Then Nuke status state is Completed", func() {
			Eventually(LoadAndCheck).
				WithArguments(infra.Ctx(), infra.KCP().Client(), nuke, NewObjActions(),
					HavingState("Completed"),
				).Should(Succeed())
		})

		By("And Then Scope is deleted", func() {
			Eventually(IsDeleted).
				WithArguments(infra.Ctx(), infra.KCP().Client(), scope).
				Should(Succeed())
		})

		By("// cleanup: Delete Nuke", func() {
			Expect(Delete(infra.Ctx(), infra.KCP().Client(), nuke)).
				To(Succeed())
		})

	})
})