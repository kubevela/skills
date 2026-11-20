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

		It("requires only the attributes and the key", func() {
			for _, p := range comp.GetParams() {
				switch p.Name() {
				case "attributeDefinitions", "keySchema":
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

		It("defines the key and projection shapes once and refers to them", func() {
			Expect(cue).To(MatchRegexp(`#KeySchemaElement:\s*\{`))
			Expect(cue).To(MatchRegexp(`#Projection:\s*\{`))
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
			Entry("billing modes", `billingMode?: "PROVISIONED" | "PAY_PER_REQUEST"`),
			Entry("capacity is at least one", `readCapacityUnits!: int & >=1`),
			Entry("at most 20 global indexes", `globalSecondaryIndexes?: list.MaxItems(20)`),
			Entry("at most 5 local indexes", `localSecondaryIndexes?: list.MaxItems(5)`),
			Entry("recovery period range", `recoveryPeriodInDays?: int & >=1 & <=35`),
			Entry("stream view types", `"KEYS_ONLY" | "NEW_IMAGE" | "OLD_IMAGE" | "NEW_AND_OLD_IMAGES"`),
			Entry("table classes", `tableClass?: "STANDARD" | "STANDARD_INFREQUENT_ACCESS"`),
			Entry("at most 50 tags", `tags?: list.MaxItems(50)`),
			Entry("deletion policies", `deletionPolicy?: "delete" | "retain"`),
		)

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
				WithParam("keySchema", testKeySchema))

			Expect(table.APIVersion()).To(Equal("dynamodb.services.k8s.aws/v1alpha1"))
			Expect(table.Kind()).To(Equal("Table"))
			Expect(table.Get("metadata.name")).To(Equal("orders"))
			Expect(table.Get("spec.tableName")).To(Equal("orders"))
			Expect(table.Get("spec.keySchema")).To(Equal(testKeySchema))
			Expect(table.Get("spec.attributeDefinitions")).To(Equal(testAttributes))
			// Nothing optional appears unless asked for.
			for _, field := range []string{"billingMode", "provisionedThroughput", "tags", "streamSpecification", "sseSpecification"} {
				Expect(table.Get("spec."+field)).To(BeNil(), field)
			}
			Expect(table.Get("metadata.annotations")).To(BeNil())
		})

		It("uses the table name when one is given", func() {
			table := comp.Render(defkit.TestContext().
				WithName("orders").
				WithParam("tableName", "orders-prod").
				WithParam("attributeDefinitions", testAttributes).
				WithParam("keySchema", testKeySchema))

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
	})
})
