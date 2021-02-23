package server

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/ginkgo/reporters"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
)

var (
	testEnv *envtest.Environment
	cli     client.Client
	//TODO: Read it from flag or put true if we have a
	//      KUBECONFIG env var
	useCluster = false

	expectedNamespace = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foowebhook",
		},
	}
	selectedScope = admissionregistrationv1beta1.NamespacedScope
	servicePath   = "/mutatepod"
	failurePolicy = admissionregistrationv1beta1.Fail
	mutatepodURL  = "https://localhost:8443/mutatepod"

	expectedMutatingWebhookConfiguration = admissionregistrationv1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foowebhook",
		},
		Webhooks: []admissionregistrationv1beta1.MutatingWebhook{
			admissionregistrationv1beta1.MutatingWebhook{
				Name: "foowebhook.qinqon.io",
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
					URL: &mutatepodURL,
				},
				FailurePolicy: &failurePolicy,
				Rules: []admissionregistrationv1beta1.RuleWithOperations{
					{
						Rule: admissionregistrationv1beta1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
							Scope:       &selectedScope,
						},
						Operations: []admissionregistrationv1beta1.OperationType{
							admissionregistrationv1beta1.Create,
						},
					},
				},
			},
		},
	}

	expectedSecret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: expectedNamespace.Name,
			Name:      "foowebhook",
		},
	}
)

func createResources() {
	err := cli.Create(context.TODO(), expectedMutatingWebhookConfiguration.DeepCopy())
	ExpectWithOffset(1, err).ToNot(HaveOccurred(), "should success creating mutatingwebhookconfiguration")

}

func deleteResources() {
	_ = cli.Delete(context.TODO(), &expectedMutatingWebhookConfiguration)
	_ = cli.Delete(context.TODO(), &expectedSecret)
}

var _ = BeforeSuite(func() {
	testEnv = &envtest.Environment{
		UseExistingCluster: &useCluster,
	}

	cfg, err := testEnv.Start()
	Expect(err).ToNot(HaveOccurred(), "should success starting testenv")

	cli, err = client.New(cfg, client.Options{})
	Expect(err).ToNot(HaveOccurred(), "should success creating client")

	// Ideally we create/delete the namespace at every test but, envtest
	// cannot delete namespaces [1] so we just create it at the beggining
	// of the test suite.
	//
	// [1] https://book.kubebuilder.io/reference/testing/envtest.html?highlight=envtest#testing-considerations
	By("Create namespace, webhook configuration and service")
	err = cli.Create(context.TODO(), &expectedNamespace)
	Expect(err).ToNot(HaveOccurred(), "should success creating namespace")

})

var _ = AfterSuite(func() {

	if useCluster {
		err := cli.Delete(context.TODO(), &expectedNamespace)
		Expect(err).ToNot(HaveOccurred(), "should success deleting namespace")
	}

	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred(), "should success stopping testenv")
})

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.server.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Server Test Suite", []Reporter{junitReporter, printer.NewlineReporter{}})
}
