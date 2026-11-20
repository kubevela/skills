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

package components_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/oam-dev/kubevela/pkg/definition/defkit"
	. "github.com/oam-dev/kubevela/pkg/definition/defkit/testing/matchers"

	"github.com/kubevela/skills/kubevela-defkit-author/assets/aws-dynamodb/components"
)

var (
	testAttributes = []any{
		map[string]any{"attributeName": "orderId", "attributeType": "S"},
		map[string]any{"attributeName": "createdAt", "attributeType": "N"},
	}
	testKeySchema = []any{
		map[string]any{"attributeName": "orderId", "keyType": "HASH"},
		map[string]any{"attributeName": "createdAt", "keyType": "RANGE"},
	}
)

var _ = Describe("dynamodb-table component", func() {
	var comp *defkit.ComponentDefinition

	BeforeEach(func() {
		comp = components.DynamodbTable()
	})

	Describe("definition", func() {
		It("is an ACK DynamoDB Table workload with a provides label", func() {
			Expect(comp.GetName()).To(Equal("dynamodb-table"))
			Expect(comp.GetWorkload().APIVersion()).To(Equal("dynamodb.services.k8s.aws/v1alpha1"))
			Expect(comp.GetWorkload().Kind()).To(Equal("Table"))
			Expect(comp.GetLabels()).To(HaveKeyWithValue("definition.oam.dev/provides", "dynamodb-table"))
		})

		It("requires only the attributes, the key and the billing mode", func() {
			for _, p := range comp.GetParams() {
				switch p.Name() {
				case "attributeDefinitions", "keySchema", "billingMode":
					Expect(p.IsRequired()).To(BeTrue(), p.Name())
				default:
					Expect(p.IsRequired()).To(BeFalse(), p.Name())
				}
			}
		})

		It("exposes every Table field the definition covers, and no credential", func() {
			for _, name := range []string{
				"tableName", "attributeDefinitions", "keySchema", "billingMode",
				"provisionedThroughput", "onDemandThroughput", "warmThroughput",
				"globalSecondaryIndexes", "localSecondaryIndexes", "streamSpecification",
				"sseSpecification", "timeToLive", "continuousBackups", "contributorInsights",
				"deletionProtectionEnabled", "tableClass", "resourcePolicy", "tags",
				"deletionPolicy",
			} {
				Expect(comp).To(HaveParamNamed(name))
			}
			for _, p := range comp.GetParams() {
				Expect(p.Name()).NotTo(ContainSubstring("secret"), "no secret parameter")
				Expect(p.Name()).NotTo(ContainSubstring("accessKey"), "no access key parameter")
			}
		})

		It("sets no default the service does not document", func() {
			for _, p := range comp.GetParams() {
				Expect(p.HasDefault()).To(BeFalse(), p.Name())
			}
		})

		It("is healthy only when ACK is synced and not terminal", func() {
			Expect(comp.GetHealthPolicy()).To(ContainSubstring("ACK.ResourceSynced"))
			Expect(comp.GetHealthPolicy()).To(ContainSubstring("ACK.Terminal"))
			Expect(comp.GetCustomStatus()).To(ContainSubstring("status.tableStatus"))
			Expect(comp.GetCustomStatus()).To(ContainSubstring("status.ackResourceMetadata.arn"))
		})
	})

	Describe("generated schema", func() {
		var cue string

		BeforeEach(func() {
			cue = comp.ToCue()
		})

		It("defines the key shape once and refers to it", func() {
			Expect(cue).To(MatchRegexp(`#KeySchemaElement:\s*\{`))
			// The table, the global indexes and the local indexes share the key shape.
			Expect(cue).To(MatchRegexp(`(?s)keySchema!:.*\[\.\.\.#KeySchemaElement\].*keySchema!:.*\[\.\.\.#KeySchemaElement\].*keySchema!:.*\[\.\.\.#KeySchemaElement\]`))
			// One list wrapper per use: a helper built from an array would nest two.
			Expect(cue).NotTo(ContainSubstring("[...[..."))
		})

		DescribeTable("carries the service's limits",
			func(fragment string) {
				Expect(cue).To(ContainSubstring(fragment))
			},
			Entry("table name pattern", `tableName?: string & =~"^[a-zA-Z0-9_.-]{3,255}$"`),
			Entry("one or two key elements", `keySchema!: list.MinItems(1) & list.MaxItems(2)`),
			Entry("key types", `keyType!: "HASH" | "RANGE"`),
			Entry("attribute types", `attributeType!: "S" | "N" | "B"`),
			Entry("billing modes, and that one must be chosen", `billingMode!: "PROVISIONED" | "PAY_PER_REQUEST"`),
			Entry("a resource policy of at most 20 KB", `resourcePolicy?: string & strings.MaxRunes(20480)`),
			Entry("capacity is at least one", `readCapacityUnits!: int & >=1`),
			Entry("at most 20 global indexes", `globalSecondaryIndexes?: list.MaxItems(20)`),
			Entry("at most 5 local indexes", `localSecondaryIndexes?: list.MaxItems(5)`),
			Entry("recovery period range", `recoveryPeriodInDays?: int & >=1 & <=35`),
			Entry("stream view types", `"KEYS_ONLY" | "NEW_IMAGE" | "OLD_IMAGE" | "NEW_AND_OLD_IMAGES"`),
			Entry("table classes", `tableClass?: "STANDARD" | "STANDARD_INFREQUENT_ACCESS"`),
			Entry("at most 50 tags", `tags?: list.MaxItems(50)`),
			Entry("deletion policies", `deletionPolicy?: "delete" | "retain"`),
		)

		// Rules that span parameters are validators: the message is the error a
		// user sees when the Application breaks the rule.
		DescribeTable("rejects combinations the service refuses",
			func(message, condition string) {
				Expect(cue).To(ContainSubstring(`"` + message + `": false`))
				Expect(cue).To(ContainSubstring(condition))
			},
			Entry("PROVISIONED without capacity",
				"provisionedThroughput is required when billingMode is PROVISIONED",
				`parameter.billingMode == "PROVISIONED" && parameter["provisionedThroughput"] == _|_`),
			Entry("PAY_PER_REQUEST with capacity",
				"provisionedThroughput is not allowed when billingMode is PAY_PER_REQUEST",
				`parameter.billingMode == "PAY_PER_REQUEST" && parameter["provisionedThroughput"] != _|_`),
			Entry("PROVISIONED with request limits",
				"onDemandThroughput is not allowed when billingMode is PROVISIONED",
				`parameter.billingMode == "PROVISIONED" && parameter["onDemandThroughput"] != _|_`),
			Entry("INCLUDE without attributes to include",
				"nonKeyAttributes is required when projectionType is INCLUDE",
				`projectionType == "INCLUDE" && nonKeyAttributes == _|_`),
			Entry("attributes to include with another projection type",
				"nonKeyAttributes is only allowed when projectionType is INCLUDE",
				`projectionType != "INCLUDE"`),
			Entry("a request limit of zero",
				"maxReadRequestUnits must be at least 1, or -1 to remove the limit",
				`maxReadRequestUnits == 0`),
		)

		It("carries the projection rules in both kinds of index", func() {
			Expect(strings.Count(cue, `"nonKeyAttributes is required when projectionType is INCLUDE": false`)).To(Equal(2))
		})

		It("copies every parameter to the CRD field of the same name", func() {
			for _, field := range []string{
				"attributeDefinitions", "keySchema", "billingMode", "provisionedThroughput",
				"onDemandThroughput", "warmThroughput", "globalSecondaryIndexes",
				"localSecondaryIndexes", "streamSpecification", "sseSpecification",
				"timeToLive", "continuousBackups", "contributorInsights",
				"deletionProtectionEnabled", "tableClass", "resourcePolicy", "tags",
			} {
				Expect(cue).To(ContainSubstring(field+": parameter."+field), field)
			}
			Expect(cue).To(ContainSubstring(`"services.k8s.aws/deletion-policy": parameter.deletionPolicy`))
		})
	})

	Describe("render", func() {
		It("renders a minimal table named after the component", func() {
			table := comp.Render(defkit.TestContext().
				WithName("orders").
				WithNamespace("team-a").
				WithParam("attributeDefinitions", testAttributes).
				WithParam("keySchema", testKeySchema).
				WithParam("billingMode", "PAY_PER_REQUEST"))

			Expect(table.APIVersion()).To(Equal("dynamodb.services.k8s.aws/v1alpha1"))
			Expect(table.Kind()).To(Equal("Table"))
			Expect(table.Get("metadata.name")).To(Equal("orders"))
			Expect(table.Get("spec.tableName")).To(Equal("orders"))
			Expect(table.Get("spec.keySchema")).To(Equal(testKeySchema))
			Expect(table.Get("spec.attributeDefinitions")).To(Equal(testAttributes))
			// Nothing optional appears unless asked for.
			Expect(table.Get("spec.billingMode")).To(Equal("PAY_PER_REQUEST"))
			for _, field := range []string{
				"provisionedThroughput", "onDemandThroughput", "warmThroughput", "globalSecondaryIndexes",
				"localSecondaryIndexes", "streamSpecification", "sseSpecification", "timeToLive",
				"continuousBackups", "contributorInsights", "deletionProtectionEnabled", "tableClass",
				"resourcePolicy", "tags",
			} {
				Expect(table.Get("spec."+field)).To(BeNil(), field)
			}
			Expect(table.Get("metadata.annotations")).To(BeNil())
		})

		It("uses the table name when one is given", func() {
			table := comp.Render(defkit.TestContext().
				WithName("orders").
				WithParam("tableName", "orders-prod").
				WithParam("attributeDefinitions", testAttributes).
				WithParam("keySchema", testKeySchema).
				WithParam("billingMode", "PAY_PER_REQUEST"))

			Expect(table.Get("metadata.name")).To(Equal("orders"))
			Expect(table.Get("spec.tableName")).To(Equal("orders-prod"))
		})

		It("renders the optional settings under the CRD's field names", func() {
			throughput := map[string]any{"readCapacityUnits": 5, "writeCapacityUnits": 5}
			stream := map[string]any{"streamEnabled": true, "streamViewType": "NEW_AND_OLD_IMAGES"}
			ttl := map[string]any{"attributeName": "expiresAt", "enabled": true}
			tags := []any{map[string]any{"key": "team", "value": "orders"}}
			gsi := []any{map[string]any{
				"indexName":  "by-created",
				"keySchema":  []any{map[string]any{"attributeName": "createdAt", "keyType": "HASH"}},
				"projection": map[string]any{"projectionType": "KEYS_ONLY"},
			}}

			table := comp.Render(defkit.TestContext().
				WithName("orders").
				WithParam("attributeDefinitions", testAttributes).
				WithParam("keySchema", testKeySchema).
				WithParam("billingMode", "PROVISIONED").
				WithParam("provisionedThroughput", throughput).
				WithParam("globalSecondaryIndexes", gsi).
				WithParam("streamSpecification", stream).
				WithParam("timeToLive", ttl).
				WithParam("deletionProtectionEnabled", true).
				WithParam("tableClass", "STANDARD").
				WithParam("tags", tags).
				WithParam("deletionPolicy", "retain"))

			Expect(table.Get("spec.billingMode")).To(Equal("PROVISIONED"))
			Expect(table.Get("spec.provisionedThroughput")).To(Equal(throughput))
			Expect(table.Get("spec.globalSecondaryIndexes")).To(Equal(gsi))
			Expect(table.Get("spec.streamSpecification")).To(Equal(stream))
			Expect(table.Get("spec.timeToLive")).To(Equal(ttl))
			Expect(table.Get("spec.deletionProtectionEnabled")).To(Equal(true))
			Expect(table.Get("spec.tableClass")).To(Equal("STANDARD"))
			Expect(table.Get("spec.tags")).To(Equal(tags))
			Expect(table.Get("metadata.annotations")).To(HaveKeyWithValue("services.k8s.aws/deletion-policy", "retain"))
		})
		It("renders the settings of an on-demand table under the CRD's field names", func() {
			limits := map[string]any{"maxReadRequestUnits": 100, "maxWriteRequestUnits": -1}
			warm := map[string]any{"readUnitsPerSecond": 13000, "writeUnitsPerSecond": 5000}
			lsi := []any{map[string]any{
				"indexName":  "by-created",
				"keySchema":  testKeySchema,
				"projection": map[string]any{"projectionType": "INCLUDE", "nonKeyAttributes": []any{"total"}},
			}}
			sse := map[string]any{"enabled": true, "sseType": "KMS", "kmsMasterKeyID": "alias/fixture"}
			backups := map[string]any{"pointInTimeRecoveryEnabled": true, "recoveryPeriodInDays": 7}
			policy := `{"Version":"2012-10-17","Statement":[]}`

			table := comp.Render(defkit.TestContext().
				WithName("events").
				WithParam("attributeDefinitions", testAttributes).
				WithParam("keySchema", testKeySchema).
				WithParam("billingMode", "PAY_PER_REQUEST").
				WithParam("onDemandThroughput", limits).
				WithParam("warmThroughput", warm).
				WithParam("localSecondaryIndexes", lsi).
				WithParam("sseSpecification", sse).
				WithParam("continuousBackups", backups).
				WithParam("contributorInsights", "ENABLED").
				WithParam("resourcePolicy", policy))

			Expect(table.Get("spec.onDemandThroughput")).To(Equal(limits))
			Expect(table.Get("spec.warmThroughput")).To(Equal(warm))
			Expect(table.Get("spec.localSecondaryIndexes")).To(Equal(lsi))
			Expect(table.Get("spec.sseSpecification")).To(Equal(sse))
			Expect(table.Get("spec.continuousBackups")).To(Equal(backups))
			Expect(table.Get("spec.contributorInsights")).To(Equal("ENABLED"))
			Expect(table.Get("spec.resourcePolicy")).To(Equal(policy))
			Expect(table.Get("spec.provisionedThroughput")).To(BeNil())
		})
	})
})
