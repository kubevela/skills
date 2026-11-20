//go:build emulator

/*
Copyright 2026 The KubeVela Authors.

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

// Package emulator_test applies dynamodb-table Applications to a disposable
// k3d or kind cluster and reads every setting back from the AWS Emulator with
// the DynamoDB API. The cluster needs KubeVela and an ACK DynamoDB controller
// installed pointing at the emulator; the vela CLI must be on PATH or in
// VELA_BIN. The emulator is deployed when the cluster has none.
//
// Run with: go test -tags emulator ./test/emulator/...
package emulator_test

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubevela/cloud-emulators/testkit"
)

// emulatorVersion is the kubevela/cloud-emulators release deployed when the
// cluster has no emulator. docs/coverage.md records results for this version.
const emulatorVersion = "1.0.533"

var tableResource = schema.GroupVersionResource{Group: "dynamodb.services.k8s.aws", Version: "v1alpha1", Resource: "tables"}

var (
	cluster *testkit.Cluster
	run     *testkit.Run
	ddb     *dynamodb.Client
	kmsAPI  *kms.Client

	stopForward func()
)

func TestEmulator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "dynamodb-table against the AWS Emulator")
}

var _ = BeforeSuite(func(ctx SpecContext) {
	var err error
	By("connecting to a disposable k3d or kind cluster")
	cluster, err = testkit.Connect()
	Expect(err).NotTo(HaveOccurred())

	By("deploying the AWS Emulator when the cluster has none")
	Expect(cluster.EnsureAWSEmulator(ctx, emulatorVersion)).To(Succeed())

	By("checking the ACK DynamoDB controller talks to the emulator and nothing else")
	Expect(cluster.RequireACKOnEmulator(ctx, "dynamodb")).To(Succeed())

	By("installing the dynamodb-table definition from this module")
	_, file, _, ok := runtime.Caller(0)
	Expect(ok).To(BeTrue())
	Expect(applyModule(ctx, filepath.Join(filepath.Dir(file), "..", ".."))).To(Succeed())

	By("creating a namespace for this run")
	run, err = cluster.NewRun(ctx, "dynamodb")
	Expect(err).NotTo(HaveOccurred())

	By("opening a connection to the emulator")
	var endpoint string
	endpoint, stopForward, err = cluster.ForwardAWSEmulator(ctx)
	Expect(err).NotTo(HaveOccurred())
	keys := credentials.NewStaticCredentialsProvider(testkit.AWSEmulatorAccessKeyID, testkit.AWSEmulatorSecretAccessKey, "")
	ddb = dynamodb.New(dynamodb.Options{Region: testkit.AWSEmulatorRegion, BaseEndpoint: aws.String(endpoint), Credentials: keys})
	kmsAPI = kms.New(kms.Options{Region: testkit.AWSEmulatorRegion, BaseEndpoint: aws.String(endpoint), Credentials: keys})
	Eventually(func() error {
		_, err := ddb.ListTables(ctx, &dynamodb.ListTablesInput{})
		return err
	}).WithTimeout(30 * time.Second).WithPolling(time.Second).Should(Succeed())
}, NodeTimeout(8*time.Minute))

var _ = AfterSuite(func(ctx SpecContext) {
	// First, so the connection is closed even when the cleanup below fails.
	if stopForward != nil {
		stopForward()
	}
	if cluster != nil && run != nil {
		By("deleting this run's Applications and waiting for ACK to delete their tables")
		Expect(cluster.EndRun(ctx, run,
			[]schema.GroupVersionResource{applicationResource},
			[]schema.GroupVersionResource{tableResource})).To(Succeed())
	}
}, NodeTimeout(5*time.Minute))
