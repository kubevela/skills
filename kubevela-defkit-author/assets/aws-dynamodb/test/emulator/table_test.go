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

package emulator_test

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/kubevela/cloud-emulators/testkit"
)

const (
	settle = 3 * time.Minute
	poll   = 2 * time.Second
)

// obj and list keep the property literals below readable.
type obj = map[string]any
type list = []any

func attr(name, kind string) obj   { return obj{"attributeName": name, "attributeType": kind} }
func key(name, keyType string) obj { return obj{"attributeName": name, "keyType": keyType} }

// applyTable applies an Application with one dynamodb-table component. The
// Application, the component and the Table object share the name.
func applyTable(ctx context.Context, name string, properties obj) {
	GinkgoHelper()
	Expect(applyComponent(ctx, run.Namespace, name, "dynamodb-table", properties)).To(Succeed())
}

// condition returns the status of one condition of the ACK Table, or "" when absent.
func condition(ctx context.Context, name, conditionType string) (string, error) {
	table, err := cluster.Dynamic.Resource(tableResource).Namespace(run.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return testkit.ACKCondition(table, conditionType), nil
}

// conditionMessage returns the message of one condition of an ACK resource.
func conditionMessage(resource *unstructured.Unstructured, conditionType string) string {
	conditions, _, _ := unstructured.NestedSlice(resource.Object, "status", "conditions")
	for _, c := range conditions {
		if condition, ok := c.(map[string]any); ok && condition["type"] == conditionType {
			message, _ := condition["message"].(string)
			return message
		}
	}
	return ""
}

// expectHealthy waits until ACK reports the table synced and not terminal, and
// KubeVela reports the component healthy.
func expectHealthy(ctx context.Context, name string) {
	GinkgoHelper()
	expectHealthyWithin(ctx, name, settle)
}

func expectHealthyWithin(ctx context.Context, name string, timeout time.Duration) {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		table, err := cluster.Dynamic.Resource(tableResource).Namespace(run.Namespace).Get(ctx, name, metav1.GetOptions{})
		g.Expect(err).NotTo(HaveOccurred())
		ready, reason := testkit.ACKReady(table)
		g.Expect(ready).To(BeTrue(), reason)
		healthy, err := componentsHealthy(ctx, run.Namespace, name)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(healthy).To(BeTrue())
	}).WithTimeout(timeout).WithPolling(poll).Should(Succeed())
}

// eventuallyTable waits until the service's own description of the table
// satisfies check. After a change this, not ACK's condition, is the proof: ACK
// conditions carry no observedGeneration, so a True read right after an update
// may answer the previous spec.
func eventuallyTable(ctx context.Context, name string, check func(g Gomega, table *types.TableDescription)) {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		out, err := ddb.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(name)})
		g.Expect(err).NotTo(HaveOccurred())
		check(g, out.Table)
	}).WithTimeout(settle).WithPolling(poll).Should(Succeed())
}

func describe(ctx context.Context, name string) *types.TableDescription {
	GinkgoHelper()
	out, err := ddb.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(name)})
	Expect(err).NotTo(HaveOccurred())
	return out.Table
}

func tableExists(ctx context.Context, name string) (bool, error) {
	_, err := ddb.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(name)})
	var missing *types.ResourceNotFoundException
	if errors.As(err, &missing) {
		return false, nil
	}
	return err == nil, err
}

var _ = Describe("dynamodb-table", func() {
	It("creates a minimal on-demand table", func(ctx SpecContext) {
		name := run.ID + "-minimal"
		applyTable(ctx, name, obj{
			"attributeDefinitions": list{attr("sessionId", "S")},
			"keySchema":            list{key("sessionId", "HASH")},
			"billingMode":          "PAY_PER_REQUEST",
		})
		expectHealthy(ctx, name)

		table := describe(ctx, name)
		Expect(table.TableStatus).To(Equal(types.TableStatusActive))
		Expect(table.KeySchema).To(ConsistOf(types.KeySchemaElement{AttributeName: aws.String("sessionId"), KeyType: types.KeyTypeHash}))
		Expect(table.BillingModeSummary.BillingMode).To(Equal(types.BillingModePayPerRequest))
	}, SpecTimeout(5*time.Minute))

	It("applies every setting of a provisioned table", func(ctx SpecContext) {
		name := run.ID + "-full"
		applyTable(ctx, name, obj{
			"attributeDefinitions":  list{attr("customerId", "S"), attr("orderId", "S"), attr("createdAt", "N"), attr("status", "S")},
			"keySchema":             list{key("customerId", "HASH"), key("orderId", "RANGE")},
			"billingMode":           "PROVISIONED",
			"provisionedThroughput": obj{"readCapacityUnits": 5, "writeCapacityUnits": 4},
			"globalSecondaryIndexes": list{obj{
				"indexName":             "by-status",
				"keySchema":             list{key("status", "HASH"), key("createdAt", "RANGE")},
				"projection":            obj{"projectionType": "INCLUDE", "nonKeyAttributes": list{"total"}},
				"provisionedThroughput": obj{"readCapacityUnits": 2, "writeCapacityUnits": 3},
			}},
			"localSecondaryIndexes": list{obj{
				"indexName":  "by-created",
				"keySchema":  list{key("customerId", "HASH"), key("createdAt", "RANGE")},
				"projection": obj{"projectionType": "KEYS_ONLY"},
			}},
			"streamSpecification": obj{"streamEnabled": true, "streamViewType": "NEW_AND_OLD_IMAGES"},
			"timeToLive":          obj{"attributeName": "expiresAt", "enabled": true},
			"continuousBackups":   obj{"pointInTimeRecoveryEnabled": true, "recoveryPeriodInDays": 7},
			"sseSpecification":    obj{"enabled": true, "sseType": "KMS"},
			"tableClass":          "STANDARD",
			"tags":                list{obj{"key": "fixture", "value": "emulator-suite"}},
		})
		expectHealthy(ctx, name)

		table := describe(ctx, name)
		By("keys and capacity")
		Expect(table.KeySchema).To(Equal([]types.KeySchemaElement{
			{AttributeName: aws.String("customerId"), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String("orderId"), KeyType: types.KeyTypeRange},
		}))
		Expect(table.AttributeDefinitions).To(HaveLen(4))
		Expect(table.ProvisionedThroughput.ReadCapacityUnits).To(HaveValue(BeEquivalentTo(5)))
		Expect(table.ProvisionedThroughput.WriteCapacityUnits).To(HaveValue(BeEquivalentTo(4)))

		By("global secondary index")
		Expect(table.GlobalSecondaryIndexes).To(HaveLen(1))
		gsi := table.GlobalSecondaryIndexes[0]
		Expect(gsi.IndexName).To(HaveValue(Equal("by-status")))
		Expect(gsi.KeySchema).To(Equal([]types.KeySchemaElement{
			{AttributeName: aws.String("status"), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String("createdAt"), KeyType: types.KeyTypeRange},
		}))
		Expect(gsi.Projection.ProjectionType).To(Equal(types.ProjectionTypeInclude))
		Expect(gsi.Projection.NonKeyAttributes).To(ConsistOf("total"))
		Expect(gsi.ProvisionedThroughput.ReadCapacityUnits).To(HaveValue(BeEquivalentTo(2)))
		Expect(gsi.ProvisionedThroughput.WriteCapacityUnits).To(HaveValue(BeEquivalentTo(3)))

		By("local secondary index")
		Expect(table.LocalSecondaryIndexes).To(HaveLen(1))
		Expect(table.LocalSecondaryIndexes[0].IndexName).To(HaveValue(Equal("by-created")))
		Expect(table.LocalSecondaryIndexes[0].Projection.ProjectionType).To(Equal(types.ProjectionTypeKeysOnly))

		By("stream, encryption and class")
		Expect(table.StreamSpecification.StreamEnabled).To(HaveValue(BeTrue()))
		Expect(table.StreamSpecification.StreamViewType).To(Equal(types.StreamViewTypeNewAndOldImages))
		Expect(table.SSEDescription.Status).To(Equal(types.SSEStatusEnabled))
		Expect(table.SSEDescription.SSEType).To(Equal(types.SSETypeKms))
		Expect(table.TableClassSummary.TableClass).To(Equal(types.TableClassStandard))

		By("time to live")
		ttl, err := ddb.DescribeTimeToLive(ctx, &dynamodb.DescribeTimeToLiveInput{TableName: aws.String(name)})
		Expect(err).NotTo(HaveOccurred())
		Expect(ttl.TimeToLiveDescription.TimeToLiveStatus).To(Equal(types.TimeToLiveStatusEnabled))
		Expect(ttl.TimeToLiveDescription.AttributeName).To(HaveValue(Equal("expiresAt")))

		By("point-in-time recovery")
		backups, err := ddb.DescribeContinuousBackups(ctx, &dynamodb.DescribeContinuousBackupsInput{TableName: aws.String(name)})
		Expect(err).NotTo(HaveOccurred())
		pitr := backups.ContinuousBackupsDescription.PointInTimeRecoveryDescription
		Expect(pitr.PointInTimeRecoveryStatus).To(Equal(types.PointInTimeRecoveryStatusEnabled))
		Expect(pitr.RecoveryPeriodInDays).To(HaveValue(BeEquivalentTo(7)))

		By("tags")
		tags, err := ddb.ListTagsOfResource(ctx, &dynamodb.ListTagsOfResourceInput{ResourceArn: table.TableArn})
		Expect(err).NotTo(HaveOccurred())
		Expect(tags.Tags).To(ContainElement(types.Tag{Key: aws.String("fixture"), Value: aws.String("emulator-suite")}))
	}, SpecTimeout(6*time.Minute))

	It("applies on-demand limits, the infrequent-access class and Contributor Insights", func(ctx SpecContext) {
		name := run.ID + "-ondemand"
		applyTable(ctx, name, obj{
			"attributeDefinitions": list{attr("eventId", "S")},
			"keySchema":            list{key("eventId", "HASH")},
			"billingMode":          "PAY_PER_REQUEST",
			"onDemandThroughput":   obj{"maxReadRequestUnits": 100, "maxWriteRequestUnits": 50},
			"tableClass":           "STANDARD_INFREQUENT_ACCESS",
			"contributorInsights":  "ENABLED",
		})
		expectHealthy(ctx, name)

		table := describe(ctx, name)
		Expect(table.BillingModeSummary.BillingMode).To(Equal(types.BillingModePayPerRequest))
		Expect(table.OnDemandThroughput.MaxReadRequestUnits).To(HaveValue(BeEquivalentTo(100)))
		Expect(table.OnDemandThroughput.MaxWriteRequestUnits).To(HaveValue(BeEquivalentTo(50)))
		Expect(table.TableClassSummary.TableClass).To(Equal(types.TableClassStandardInfrequentAccess))

		insights, err := ddb.DescribeContributorInsights(ctx, &dynamodb.DescribeContributorInsightsInput{TableName: aws.String(name)})
		Expect(err).NotTo(HaveOccurred())
		Expect(insights.ContributorInsightsStatus).To(Equal(types.ContributorInsightsStatusEnabled))
	}, SpecTimeout(5*time.Minute))

	It("uses tableName for the table and the component name for the object", func(ctx SpecContext) {
		name := run.ID + "-named"
		awsName := run.ID + "-named-in-aws"
		applyTable(ctx, name, obj{
			"tableName":            awsName,
			"attributeDefinitions": list{attr("id", "S")},
			"keySchema":            list{key("id", "HASH")},
			"billingMode":          "PAY_PER_REQUEST",
		})
		expectHealthy(ctx, name)

		Expect(describe(ctx, awsName).TableStatus).To(Equal(types.TableStatusActive))
		Expect(tableExists(ctx, name)).To(BeFalse())
	}, SpecTimeout(5*time.Minute))

	It("changes capacity and tags of an existing table", func(ctx SpecContext) {
		name := run.ID + "-update"
		properties := obj{
			"attributeDefinitions":  list{attr("id", "S")},
			"keySchema":             list{key("id", "HASH")},
			"billingMode":           "PROVISIONED",
			"provisionedThroughput": obj{"readCapacityUnits": 1, "writeCapacityUnits": 1},
			"tags":                  list{obj{"key": "stage", "value": "before"}},
		}
		applyTable(ctx, name, properties)
		expectHealthy(ctx, name)
		Expect(describe(ctx, name).ProvisionedThroughput.ReadCapacityUnits).To(HaveValue(BeEquivalentTo(1)))

		properties["provisionedThroughput"] = obj{"readCapacityUnits": 7, "writeCapacityUnits": 6}
		properties["tags"] = list{obj{"key": "stage", "value": "after"}}
		applyTable(ctx, name, properties)

		// ACK conditions carry no observedGeneration, so a True read right after
		// the change may be the old one. The service's own answer is the proof.
		Eventually(func(g Gomega) {
			table, err := ddb.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(name)})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(table.Table.ProvisionedThroughput.ReadCapacityUnits).To(HaveValue(BeEquivalentTo(7)))
			g.Expect(table.Table.ProvisionedThroughput.WriteCapacityUnits).To(HaveValue(BeEquivalentTo(6)))
			tags, err := ddb.ListTagsOfResource(ctx, &dynamodb.ListTagsOfResourceInput{ResourceArn: table.Table.TableArn})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(tags.Tags).To(ContainElement(types.Tag{Key: aws.String("stage"), Value: aws.String("after")}))
		}).WithTimeout(settle).WithPolling(poll).Should(Succeed())
		expectHealthyWithin(ctx, name, velaResync)
	}, SpecTimeout(14*time.Minute))

	It("deletes the table with the Application", func(ctx SpecContext) {
		name := run.ID + "-delete"
		applyTable(ctx, name, obj{
			"attributeDefinitions": list{attr("id", "S")},
			"keySchema":            list{key("id", "HASH")},
			"billingMode":          "PAY_PER_REQUEST",
		})
		expectHealthy(ctx, name)

		Expect(cluster.Dynamic.Resource(applicationResource).Namespace(run.Namespace).Delete(ctx, name, metav1.DeleteOptions{})).To(Succeed())
		Eventually(func() (bool, error) { return tableExists(ctx, name) }).
			WithTimeout(settle).WithPolling(poll).Should(BeFalse())
	}, SpecTimeout(6*time.Minute))

	It("keeps the table when deletionPolicy is retain", func(ctx SpecContext) {
		name := run.ID + "-retain"
		applyTable(ctx, name, obj{
			"attributeDefinitions": list{attr("id", "S")},
			"keySchema":            list{key("id", "HASH")},
			"billingMode":          "PAY_PER_REQUEST",
			"deletionPolicy":       "retain",
		})
		expectHealthy(ctx, name)

		Expect(cluster.Dynamic.Resource(applicationResource).Namespace(run.Namespace).Delete(ctx, name, metav1.DeleteOptions{})).To(Succeed())
		Eventually(func() error {
			_, err := cluster.Dynamic.Resource(tableResource).Namespace(run.Namespace).Get(ctx, name, metav1.GetOptions{})
			return err
		}).WithTimeout(settle).WithPolling(poll).Should(HaveOccurred(), "the Table object goes away")
		Expect(tableExists(ctx, name)).To(BeTrue(), "the table stays in the service")

		// The fixture is this suite's to clean up, since ACK no longer owns it.
		_, err := ddb.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: aws.String(name)})
		Expect(err).NotTo(HaveOccurred())
	}, SpecTimeout(6*time.Minute))

	It("is unhealthy when the service rejects the table for good", func(ctx SpecContext) {
		name := run.ID + "-rejected"
		// The key names an attribute that attributeDefinitions does not declare.
		// The schema cannot see that; DynamoDB refuses it and ACK stops trying.
		applyTable(ctx, name, obj{
			"attributeDefinitions": list{attr("id", "S")},
			"keySchema":            list{key("missing", "HASH")},
			"billingMode":          "PAY_PER_REQUEST",
		})

		Eventually(func() (string, error) { return condition(ctx, name, "ACK.Terminal") }).
			WithTimeout(settle).WithPolling(poll).Should(Equal("True"))
		Consistently(func() (bool, error) { return componentsHealthy(ctx, run.Namespace, name) }).
			WithTimeout(10 * time.Second).WithPolling(poll).Should(BeFalse())
		Expect(tableExists(ctx, name)).To(BeFalse())
	}, SpecTimeout(6*time.Minute))

	It("pre-warms an on-demand table", func(ctx SpecContext) {
		name := run.ID + "-warm"
		applyTable(ctx, name, obj{
			"attributeDefinitions": list{attr("id", "S")},
			"keySchema":            list{key("id", "HASH")},
			"billingMode":          "PAY_PER_REQUEST",
			// Above what DynamoDB gives a new on-demand table (12000 and 4000).
			"warmThroughput": obj{"readUnitsPerSecond": 13000, "writeUnitsPerSecond": 5000},
		})
		expectHealthy(ctx, name)

		eventuallyTable(ctx, name, func(g Gomega, table *types.TableDescription) {
			g.Expect(table.WarmThroughput).NotTo(BeNil())
			g.Expect(table.WarmThroughput.ReadUnitsPerSecond).To(HaveValue(BeEquivalentTo(13000)))
			g.Expect(table.WarmThroughput.WriteUnitsPerSecond).To(HaveValue(BeEquivalentTo(5000)))
		})
	}, SpecTimeout(8*time.Minute))

	It("pre-warms a global secondary index of an on-demand table", func(ctx SpecContext) {
		name := run.ID + "-gsi-warm"
		applyTable(ctx, name, obj{
			"attributeDefinitions": list{attr("id", "S"), attr("status", "S")},
			"keySchema":            list{key("id", "HASH")},
			"billingMode":          "PAY_PER_REQUEST",
			"globalSecondaryIndexes": list{obj{
				"indexName":      "by-status",
				"keySchema":      list{key("status", "HASH")},
				"projection":     obj{"projectionType": "ALL"},
				"warmThroughput": obj{"readUnitsPerSecond": 13000, "writeUnitsPerSecond": 5000},
			}},
		})
		expectHealthy(ctx, name)

		eventuallyTable(ctx, name, func(g Gomega, table *types.TableDescription) {
			g.Expect(table.GlobalSecondaryIndexes).To(HaveLen(1))
			gsi := table.GlobalSecondaryIndexes[0]
			g.Expect(gsi.WarmThroughput).NotTo(BeNil())
			g.Expect(gsi.WarmThroughput.ReadUnitsPerSecond).To(HaveValue(BeEquivalentTo(13000)))
			g.Expect(gsi.WarmThroughput.WriteUnitsPerSecond).To(HaveValue(BeEquivalentTo(5000)))
		})
	}, SpecTimeout(8*time.Minute))

	// Endpoint-limited in AWS Emulator 1.0.533: the emulator refuses a table whose
	// index carries onDemandThroughput. This spec pins the refusal. When it fails
	// because the table was created, the emulator has caught up: replace it with
	// a read-back of GlobalSecondaryIndexes[].OnDemandThroughput and move the
	// lever to "verified end to end" in docs/coverage.md.
	It("cannot cap a global secondary index of an on-demand table on this emulator", func(ctx SpecContext) {
		name := run.ID + "-gsi-ondemand"
		applyTable(ctx, name, obj{
			"attributeDefinitions": list{attr("id", "S"), attr("status", "S")},
			"keySchema":            list{key("id", "HASH")},
			"billingMode":          "PAY_PER_REQUEST",
			"globalSecondaryIndexes": list{obj{
				"indexName":          "by-status",
				"keySchema":          list{key("status", "HASH")},
				"projection":         obj{"projectionType": "ALL"},
				"onDemandThroughput": obj{"maxReadRequestUnits": 40, "maxWriteRequestUnits": 20},
			}},
		})

		Eventually(func(g Gomega) {
			table, err := cluster.Dynamic.Resource(tableResource).Namespace(run.Namespace).Get(ctx, name, metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(testkit.ACKCondition(table, "ACK.ResourceSynced")).NotTo(Equal("True"),
				"the emulator created the table: it now supports onDemandThroughput on an index, see the comment above this spec")
			g.Expect(conditionMessage(table, "ACK.Recoverable")).To(ContainSubstring("NotImplemented"))
		}).WithTimeout(settle).WithPolling(poll).Should(Succeed())
		Expect(tableExists(ctx, name)).To(BeFalse())
	}, SpecTimeout(6*time.Minute))

	It("turns deletion protection on and off", func(ctx SpecContext) {
		name := run.ID + "-protected"
		properties := obj{
			"attributeDefinitions":      list{attr("id", "S")},
			"keySchema":                 list{key("id", "HASH")},
			"billingMode":               "PAY_PER_REQUEST",
			"deletionProtectionEnabled": true,
		}
		applyTable(ctx, name, properties)
		// A protected table cannot be deleted, so the suite could not clean up
		// after itself: switch protection off again whatever happens below.
		DeferCleanup(func(ctx SpecContext) {
			properties["deletionProtectionEnabled"] = false
			applyTable(ctx, name, properties)
			eventuallyTable(ctx, name, func(g Gomega, table *types.TableDescription) {
				g.Expect(table.DeletionProtectionEnabled).To(HaveValue(BeFalse()))
			})
		}, NodeTimeout(4*time.Minute))
		expectHealthy(ctx, name)

		eventuallyTable(ctx, name, func(g Gomega, table *types.TableDescription) {
			g.Expect(table.DeletionProtectionEnabled).To(HaveValue(BeTrue()))
		})
		_, err := ddb.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: aws.String(name)})
		Expect(err).To(HaveOccurred(), "the service refuses to delete a protected table")
	}, SpecTimeout(8*time.Minute))

	It("attaches a resource policy", func(ctx SpecContext) {
		name := run.ID + "-policy"
		arn := "arn:aws:dynamodb:" + testkit.AWSEmulatorRegion + ":000000000000:table/" + name
		applyTable(ctx, name, obj{
			"attributeDefinitions": list{attr("id", "S")},
			"keySchema":            list{key("id", "HASH")},
			"billingMode":          "PAY_PER_REQUEST",
			"resourcePolicy": `{"Version":"2012-10-17","Statement":[{"Sid":"fixture","Effect":"Allow",` +
				`"Principal":{"AWS":"arn:aws:iam::000000000000:root"},"Action":"dynamodb:GetItem","Resource":"` + arn + `"}]}`,
		})
		expectHealthy(ctx, name)

		Eventually(func(g Gomega) {
			out, err := ddb.GetResourcePolicy(ctx, &dynamodb.GetResourcePolicyInput{ResourceArn: aws.String(arn)})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(out.Policy).To(HaveValue(And(ContainSubstring("dynamodb:GetItem"), ContainSubstring("fixture"))))
		}).WithTimeout(settle).WithPolling(poll).Should(Succeed())
	}, SpecTimeout(8*time.Minute))

	It("encrypts with the KMS key named by kmsMasterKeyID", func(ctx SpecContext) {
		name := run.ID + "-kms"
		created, err := kmsAPI.CreateKey(ctx, &kms.CreateKeyInput{Description: aws.String("fixture for " + name)})
		Expect(err).NotTo(HaveOccurred())
		keyARN := aws.ToString(created.KeyMetadata.Arn)
		DeferCleanup(func(ctx SpecContext) {
			_, _ = kmsAPI.ScheduleKeyDeletion(ctx, &kms.ScheduleKeyDeletionInput{KeyId: aws.String(keyARN), PendingWindowInDays: aws.Int32(7)})
		})

		applyTable(ctx, name, obj{
			"attributeDefinitions": list{attr("id", "S")},
			"keySchema":            list{key("id", "HASH")},
			"billingMode":          "PAY_PER_REQUEST",
			"sseSpecification":     obj{"enabled": true, "sseType": "KMS", "kmsMasterKeyID": keyARN},
		})
		expectHealthy(ctx, name)

		eventuallyTable(ctx, name, func(g Gomega, table *types.TableDescription) {
			g.Expect(table.SSEDescription).NotTo(BeNil())
			g.Expect(table.SSEDescription.SSEType).To(Equal(types.SSETypeKms))
			g.Expect(table.SSEDescription.KMSMasterKeyArn).To(HaveValue(Equal(keyARN)))
		})
	}, SpecTimeout(8*time.Minute))

	It("adds and removes a global secondary index on an existing table", func(ctx SpecContext) {
		name := run.ID + "-gsi-change"
		without := obj{
			"attributeDefinitions": list{attr("id", "S")},
			"keySchema":            list{key("id", "HASH")},
			"billingMode":          "PAY_PER_REQUEST",
		}
		applyTable(ctx, name, without)
		expectHealthy(ctx, name)
		Expect(describe(ctx, name).GlobalSecondaryIndexes).To(BeEmpty())

		By("adding an index, with the attribute it is keyed on")
		applyTable(ctx, name, obj{
			"attributeDefinitions": list{attr("id", "S"), attr("status", "S")},
			"keySchema":            list{key("id", "HASH")},
			"billingMode":          "PAY_PER_REQUEST",
			"globalSecondaryIndexes": list{obj{
				"indexName":  "by-status",
				"keySchema":  list{key("status", "HASH")},
				"projection": obj{"projectionType": "KEYS_ONLY"},
			}},
		})
		eventuallyTable(ctx, name, func(g Gomega, table *types.TableDescription) {
			g.Expect(table.GlobalSecondaryIndexes).To(HaveLen(1))
			g.Expect(table.GlobalSecondaryIndexes[0].IndexName).To(HaveValue(Equal("by-status")))
			g.Expect(table.GlobalSecondaryIndexes[0].IndexStatus).To(Equal(types.IndexStatusActive))
		})

		By("removing it again")
		applyTable(ctx, name, without)
		eventuallyTable(ctx, name, func(g Gomega, table *types.TableDescription) {
			g.Expect(table.GlobalSecondaryIndexes).To(BeEmpty())
		})
	}, SpecTimeout(10*time.Minute))

	It("switches an on-demand table to provisioned capacity", func(ctx SpecContext) {
		name := run.ID + "-billing"
		properties := obj{
			"attributeDefinitions": list{attr("id", "S")},
			"keySchema":            list{key("id", "HASH")},
			"billingMode":          "PAY_PER_REQUEST",
		}
		applyTable(ctx, name, properties)
		expectHealthy(ctx, name)
		Expect(describe(ctx, name).BillingModeSummary.BillingMode).To(Equal(types.BillingModePayPerRequest))

		properties["billingMode"] = "PROVISIONED"
		properties["provisionedThroughput"] = obj{"readCapacityUnits": 3, "writeCapacityUnits": 2}
		applyTable(ctx, name, properties)
		eventuallyTable(ctx, name, func(g Gomega, table *types.TableDescription) {
			g.Expect(table.ProvisionedThroughput.ReadCapacityUnits).To(HaveValue(BeEquivalentTo(3)))
			g.Expect(table.ProvisionedThroughput.WriteCapacityUnits).To(HaveValue(BeEquivalentTo(2)))
		})
	}, SpecTimeout(8*time.Minute))
})
