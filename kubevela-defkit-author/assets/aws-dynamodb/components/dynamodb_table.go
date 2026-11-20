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

package components

import (
	"github.com/oam-dev/kubevela/pkg/definition/defkit"
)

// The component renders a Table of the AWS Controllers for Kubernetes (ACK)
// DynamoDB controller. Parameter names and shapes follow the controller's CRD
// at release 1.16.0, dynamodb.services.k8s.aws/v1alpha1 Table (required:
// tableName, attributeDefinitions, keySchema). Limits come from the DynamoDB
// API reference. Region and credentials come from the ACK controller's
// configuration, not from parameters.
const (
	dynamodbAPIVersion = "dynamodb.services.k8s.aws/v1alpha1"

	// ackSyncedCondition is True once ACK has reconciled the resource with AWS.
	ackSyncedCondition = "ACK.ResourceSynced"
	// ackTerminalCondition is True when ACK has given up on the resource. Synced
	// keeps its last value then, so health must read both.
	ackTerminalCondition = "ACK.Terminal"

	// ackDeletionPolicyAnnotation tells ACK what to do with the AWS resource
	// when the Kubernetes object is deleted.
	ackDeletionPolicyAnnotation = "services.k8s.aws/deletion-policy"

	keySchemaElementRef = "KeySchemaElement"
	projectionRef       = "Projection"
)

var (
	tableName             *defkit.StringParam
	attributeDefinitions  *defkit.ArrayParam
	keySchema             *defkit.ArrayParam
	billingMode           *defkit.EnumParam
	provisionedThroughput *defkit.MapParam
	onDemandThroughput    *defkit.MapParam
	warmThroughput        *defkit.MapParam
	globalIndexes         *defkit.ArrayParam
	localIndexes          *defkit.ArrayParam
	streamSpecification   *defkit.MapParam
	sseSpecification      *defkit.MapParam
	timeToLive            *defkit.MapParam
	continuousBackups     *defkit.MapParam
	contributorInsights   *defkit.EnumParam
	deletionProtection    *defkit.BoolParam
	tableClass            *defkit.EnumParam
	resourcePolicy        *defkit.StringParam
	tags                  *defkit.ArrayParam
	deletionPolicy        *defkit.EnumParam
)

func init() {
	defkit.Register(DynamodbTable())
}

// keySchemaElement is the shape of one key of a table or an index. The table,
// its global indexes and its local indexes all use it, so it is defined once.
func keySchemaElement() *defkit.MapParam {
	return defkit.Object("keySchemaElement").WithFields(
		defkit.String("attributeName").Required().MinLen(1).MaxLen(255).
			Description("Name of a key attribute. It must also appear in attributeDefinitions."),
		defkit.Enum("keyType").Required().Values("HASH", "RANGE").
			Description("HASH for the partition key, RANGE for the sort key."),
	)
}

// projection is the shape that says which attributes an index copies from the table.
func projection() *defkit.MapParam {
	return defkit.Object("projection").WithFields(
		defkit.Enum("projectionType").Optional().Values("ALL", "KEYS_ONLY", "INCLUDE").
			Description("ALL copies every attribute, KEYS_ONLY only the keys, INCLUDE the keys plus nonKeyAttributes."),
		defkit.Array("nonKeyAttributes").Of(defkit.ParamTypeString).Optional().MinItems(1).MaxItems(20).
			Description("Extra attributes to copy into the index. Only with projectionType INCLUDE."),
	)
}

// throughputFields are the read and write capacity units of a provisioned table or index.
func throughputFields() []defkit.Param {
	return []defkit.Param{
		defkit.Int("readCapacityUnits").Required().Min(1).
			Description("Strongly consistent reads per second."),
		defkit.Int("writeCapacityUnits").Required().Min(1).
			Description("Writes per second."),
	}
}

// onDemandFields cap the request units of an on-demand table or index.
func onDemandFields() []defkit.Param {
	return []defkit.Param{
		defkit.Int("maxReadRequestUnits").Optional().Min(-1).
			Description("Most read request units allowed. -1 removes the limit."),
		defkit.Int("maxWriteRequestUnits").Optional().Min(-1).
			Description("Most write request units allowed. -1 removes the limit."),
	}
}

// warmFields pre-warm a table or index for a known load.
func warmFields() []defkit.Param {
	return []defkit.Param{
		defkit.Int("readUnitsPerSecond").Optional().
			Description("Reads per second the table can serve at once."),
		defkit.Int("writeUnitsPerSecond").Optional().
			Description("Writes per second the table can serve at once."),
	}
}

// DynamodbTable creates the dynamodb-table component definition: an Amazon
// DynamoDB table provisioned through ACK.
func DynamodbTable() *defkit.ComponentDefinition {
	tableName = defkit.String("tableName").Optional().
		Pattern(`^[a-zA-Z0-9_.-]{3,255}$`).
		Description("Name of the table in AWS. Defaults to the component name. Cannot change after creation.")
	attributeDefinitions = defkit.Array("attributeDefinitions").Required().MinItems(1).
		WithFields(
			defkit.String("attributeName").Required().MinLen(1).MaxLen(255).
				Description("Name of the attribute."),
			defkit.Enum("attributeType").Required().Values("S", "N", "B").
				Description("S for string, N for number, B for binary. Quote it in YAML: a bare N is read as false."),
		).
		Description("The attributes used as keys by the table and its indexes. Other attributes are not declared.")
	keySchema = defkit.Array("keySchema").Required().MinItems(1).MaxItems(2).
		WithSchemaRef(keySchemaElementRef).
		Description("The primary key: one HASH key, optionally followed by one RANGE key. Cannot change after creation.")
	billingMode = defkit.Enum("billingMode").Optional().Values("PROVISIONED", "PAY_PER_REQUEST").
		Description("PROVISIONED needs provisionedThroughput; PAY_PER_REQUEST must not have it. DynamoDB uses PROVISIONED when unset.")
	provisionedThroughput = defkit.Object("provisionedThroughput").Optional().
		WithFields(throughputFields()...).
		Description("Capacity of a PROVISIONED table.")
	onDemandThroughput = defkit.Object("onDemandThroughput").Optional().
		WithFields(onDemandFields()...).
		Description("Upper limits for a PAY_PER_REQUEST table.")
	warmThroughput = defkit.Object("warmThroughput").Optional().
		WithFields(warmFields()...).
		Description("Throughput the table is pre-warmed for.")
	globalIndexes = defkit.Array("globalSecondaryIndexes").Optional().MaxItems(20).
		WithFields(
			defkit.String("indexName").Required().Pattern(`^[a-zA-Z0-9_.-]{3,255}$`).
				Description("Name of the index, unique within the table."),
			defkit.Array("keySchema").Required().MinItems(1).WithSchemaRef(keySchemaElementRef).
				Description("Key of the index."),
			defkit.Object("projection").Required().WithSchemaRef(projectionRef).
				Description("Attributes copied into the index."),
			defkit.Object("provisionedThroughput").Optional().WithFields(throughputFields()...).
				Description("Capacity of the index on a PROVISIONED table."),
			defkit.Object("onDemandThroughput").Optional().WithFields(onDemandFields()...).
				Description("Upper limits of the index on a PAY_PER_REQUEST table."),
			defkit.Object("warmThroughput").Optional().WithFields(warmFields()...).
				Description("Throughput the index is pre-warmed for."),
		).
		Description("Global secondary indexes: a different key over the same items.")
	localIndexes = defkit.Array("localSecondaryIndexes").Optional().MaxItems(5).
		WithFields(
			defkit.String("indexName").Required().Pattern(`^[a-zA-Z0-9_.-]{3,255}$`).
				Description("Name of the index, unique within the table."),
			defkit.Array("keySchema").Required().MinItems(1).WithSchemaRef(keySchemaElementRef).
				Description("Key of the index. It starts with the table's HASH key."),
			defkit.Object("projection").Required().WithSchemaRef(projectionRef).
				Description("Attributes copied into the index."),
		).
		Description("Local secondary indexes: another sort key under the table's partition key. Cannot change after creation.")
	streamSpecification = defkit.Object("streamSpecification").Optional().
		WithFields(
			defkit.Bool("streamEnabled").Required().
				Description("Whether DynamoDB Streams records changes to the table."),
			defkit.Enum("streamViewType").Optional().Values("KEYS_ONLY", "NEW_IMAGE", "OLD_IMAGE", "NEW_AND_OLD_IMAGES").
				Description("What each stream record carries."),
		).
		Description("DynamoDB Streams settings.")
	sseSpecification = defkit.Object("sseSpecification").Optional().
		WithFields(
			defkit.Bool("enabled").Optional().
				Description("True encrypts with a KMS key; false or unset uses a key owned by AWS."),
			defkit.Enum("sseType").Optional().Values("AES256", "KMS").
				Description("Encryption type."),
			defkit.String("kmsMasterKeyID").Optional().
				Description("KMS key ID, ARN or alias to use instead of the AWS managed key."),
		).
		Description("Server-side encryption settings.")
	timeToLive = defkit.Object("timeToLive").Optional().
		WithFields(
			defkit.String("attributeName").Required().MinLen(1).MaxLen(255).
				Description("Attribute holding each item's expiry time, in epoch seconds."),
			defkit.Bool("enabled").Required().
				Description("Whether items are deleted after their expiry time."),
		).
		Description("Automatic deletion of expired items.")
	continuousBackups = defkit.Object("continuousBackups").Optional().
		WithFields(
			defkit.Bool("pointInTimeRecoveryEnabled").Optional().
				Description("Whether the table can be restored to any second in the recovery period."),
			defkit.Int("recoveryPeriodInDays").Optional().Min(1).Max(35).
				Description("How far back a restore can go. DynamoDB uses 35 when unset."),
		).
		Description("Point-in-time recovery settings.")
	contributorInsights = defkit.Enum("contributorInsights").Optional().Values("ENABLED", "DISABLED").
		Description("Whether CloudWatch Contributor Insights is on for the table.")
	deletionProtection = defkit.Bool("deletionProtectionEnabled").Optional().
		Description("Refuse to delete the table while enabled.")
	tableClass = defkit.Enum("tableClass").Optional().Values("STANDARD", "STANDARD_INFREQUENT_ACCESS").
		Description("Storage class of the table.")
	resourcePolicy = defkit.String("resourcePolicy").Optional().
		Description("Resource-based IAM policy for the table, as a JSON document of at most 20 KB.")
	tags = defkit.Array("tags").Optional().MaxItems(50).
		WithFields(
			defkit.String("key").Required().MinLen(1).MaxLen(128).
				Description("Tag key."),
			defkit.String("value").Required().MaxLen(256).
				Description("Tag value."),
		).
		Description("AWS tags for the table.")
	deletionPolicy = defkit.Enum("deletionPolicy").Optional().Values("delete", "retain").
		Description("What ACK does with the table in AWS when the component is deleted. ACK deletes it when unset.")

	health := defkit.Health()
	return defkit.NewComponent("dynamodb-table").
		Description("An Amazon DynamoDB table provisioned through the AWS Controllers for Kubernetes (ACK) DynamoDB controller.").
		Labels(map[string]string{"definition.oam.dev/provides": "dynamodb-table"}).
		Workload(dynamodbAPIVersion, "Table").
		HealthPolicyExpr(health.And(
			health.Condition(ackSyncedCondition).IsTrue(),
			health.Not(health.Condition(ackTerminalCondition).IsTrue()),
		)).
		CustomStatus(defkit.Status().
			StringField("tableStatus", "status.tableStatus", "PENDING").
			StringField("arn", "status.ackResourceMetadata.arn", "").
			Message("DynamoDB table \\(tableStatus); arn: \\(arn)").
			Build()).
		Helper(keySchemaElementRef, keySchemaElement()).
		Helper(projectionRef, projection()).
		Params(
			tableName, attributeDefinitions, keySchema,
			billingMode, provisionedThroughput, onDemandThroughput, warmThroughput,
			globalIndexes, localIndexes,
			streamSpecification, sseSpecification, timeToLive, continuousBackups,
			contributorInsights, deletionProtection, tableClass, resourcePolicy,
			tags, deletionPolicy,
		).
		Template(dynamodbTableTemplate)
}

// dynamodbTableTemplate renders the ACK Table. Every parameter keeps the CRD's
// field name and shape, so each one is copied to the same path under spec.
func dynamodbTableTemplate(tpl *defkit.Template) {
	vela := defkit.VelaCtx()

	table := defkit.NewResource(dynamodbAPIVersion, "Table").
		Set("metadata.name", vela.Name()).
		SetIf(deletionPolicy.IsSet(), "metadata.annotations["+ackDeletionPolicyAnnotation+"]", deletionPolicy).
		SetIf(tableName.IsSet(), "spec.tableName", tableName).
		SetIf(tableName.NotSet(), "spec.tableName", vela.Name()).
		Set("spec.attributeDefinitions", attributeDefinitions).
		Set("spec.keySchema", keySchema).
		SetIf(billingMode.IsSet(), "spec.billingMode", billingMode).
		SetIf(provisionedThroughput.IsSet(), "spec.provisionedThroughput", provisionedThroughput).
		SetIf(onDemandThroughput.IsSet(), "spec.onDemandThroughput", onDemandThroughput).
		SetIf(warmThroughput.IsSet(), "spec.warmThroughput", warmThroughput).
		SetIf(globalIndexes.IsSet(), "spec.globalSecondaryIndexes", globalIndexes).
		SetIf(localIndexes.IsSet(), "spec.localSecondaryIndexes", localIndexes).
		SetIf(streamSpecification.IsSet(), "spec.streamSpecification", streamSpecification).
		SetIf(sseSpecification.IsSet(), "spec.sseSpecification", sseSpecification).
		SetIf(timeToLive.IsSet(), "spec.timeToLive", timeToLive).
		SetIf(continuousBackups.IsSet(), "spec.continuousBackups", continuousBackups).
		SetIf(contributorInsights.IsSet(), "spec.contributorInsights", contributorInsights).
		SetIf(deletionProtection.IsSet(), "spec.deletionProtectionEnabled", deletionProtection).
		SetIf(tableClass.IsSet(), "spec.tableClass", tableClass).
		SetIf(resourcePolicy.IsSet(), "spec.resourcePolicy", resourcePolicy).
		SetIf(tags.IsSet(), "spec.tags", tags)
	tpl.Output(table)
}
