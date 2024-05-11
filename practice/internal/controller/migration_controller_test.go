package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	log "sigs.k8s.io/controller-runtime/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1alpha1 "tony123.tw/api/v1alpha1"
)

var _ = Describe("Migration Controller", func() {
	const (
		resourceName = "test-migration"
		podName      = "dnsutils"
		deployment   = ""
		namespace    = "default"
		destination  = "kubdnode02"
		timeout      = time.Second * 30
		interval     = time.Second * 1
	)

	Context("When reconciling a resource", func() {
		BeforeEach(func() {
			By("creating the custom resource for the Kind Migration")

			err := k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: "default"}, &apiv1alpha1.Migration{})
			if err != nil && errors.IsNotFound(err) {
				ctx := context.Background()
				migration := &apiv1alpha1.Migration{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					TypeMeta: metav1.TypeMeta{
						APIVersion: "api.my.domain/v1alpha1",
						Kind:       "Migration",
					},
					Spec: apiv1alpha1.MigrationSpec{
						Podname:     podName,
						Deployment:  deployment,
						Namespace:   namespace,
						Destination: destination,
					},
				}
				log.Log.Info("Creating a new Migration", "Migration.Namespace", migration.Namespace, "Migration.Name", migration.Name)
				Expect(k8sClient.Create(ctx, migration)).Should(Succeed())
			}
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")

			migrationLookupKey := types.NamespacedName{Name: resourceName, Namespace: "default"}
			migration := &apiv1alpha1.Migration{}

			// We'll need to retry getting this newly created CronJob, given that creation may not immediately happen.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, migrationLookupKey, migration)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(migration.Spec.Podname).Should(Equal(podName))
			Expect(migration.Spec.Deployment).Should(Equal(deployment))
			Expect(migration.Spec.Namespace).Should(Equal(namespace))
			Expect(migration.Spec.Destination).Should(Equal(destination))

			log.Log.Info("Migration created successfully", "Migration.Namespace", migration.Namespace, "Migration.Name", migration.Name)
		})
	})
})
