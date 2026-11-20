import (
	"list"
	"strings"
)

"dynamodb-table": {
	type: "component"
	annotations: {}
	labels: {
		"definition.oam.dev/provides": "dynamodb-table"
	}
	description: "An Amazon DynamoDB table provisioned through the AWS Controllers for Kubernetes (ACK) DynamoDB controller."
	attributes: {
		workload: {
			definition: {
				apiVersion: "dynamodb.services.k8s.aws/v1alpha1"
				kind:       "Table"
			}
			type: "tables.dynamodb.services.k8s.aws"
		}
		status: {
			customStatus: #"""
				tableStatus: *"PENDING" | string
				if context.output.status.tableStatus != _|_ {
					tableStatus: context.output.status.tableStatus
				}
				arn: *"" | string
				if context.output.status.ackResourceMetadata.arn != _|_ {
					arn: context.output.status.ackResourceMetadata.arn
				}
				message: "DynamoDB table \(tableStatus); arn: \(arn)"
				"""#
			healthPolicy: #"""
				_ack_2eresourcesyncedCond: [ for c in *context.output.status.conditions | [] if c.type != _|_ if c.type == "ACK.ResourceSynced" { c } ]
				_ack_2eterminalCond: [ for c in *context.output.status.conditions | [] if c.type != _|_ if c.type == "ACK.Terminal" { c } ]
				isHealth: (len([ for c in _ack_2eresourcesyncedCond if c.status != _|_ if c.status == "True" { c } ]) > 0) && (!(len([ for c in _ack_2eterminalCond if c.status != _|_ if c.status == "True" { c } ]) > 0))
				"""#
		}
	}
}
template: {
	output: {
		apiVersion: "dynamodb.services.k8s.aws/v1alpha1"
		kind:       "Table"
		metadata: {
			name: context.name
			if parameter["deletionPolicy"] != _|_ {
				annotations: {
					"services.k8s.aws/deletion-policy": parameter.deletionPolicy
				}
			}
		}
		spec: {
			if parameter["tableName"] != _|_ {
				tableName: parameter.tableName
			}
			if parameter["tableName"] == _|_ {
				tableName: context.name
			}
			attributeDefinitions: parameter.attributeDefinitions
			keySchema: parameter.keySchema
			if parameter["billingMode"] != _|_ {
				billingMode: parameter.billingMode
			}
			if parameter["continuousBackups"] != _|_ {
				continuousBackups: parameter.continuousBackups
			}
			if parameter["contributorInsights"] != _|_ {
				contributorInsights: parameter.contributorInsights
			}
			if parameter["deletionProtectionEnabled"] != _|_ {
				deletionProtectionEnabled: parameter.deletionProtectionEnabled
			}
			if parameter["globalSecondaryIndexes"] != _|_ {
				globalSecondaryIndexes: parameter.globalSecondaryIndexes
			}
			if parameter["localSecondaryIndexes"] != _|_ {
				localSecondaryIndexes: parameter.localSecondaryIndexes
			}
			if parameter["onDemandThroughput"] != _|_ {
				onDemandThroughput: parameter.onDemandThroughput
			}
			if parameter["provisionedThroughput"] != _|_ {
				provisionedThroughput: parameter.provisionedThroughput
			}
			if parameter["resourcePolicy"] != _|_ {
				resourcePolicy: parameter.resourcePolicy
			}
			if parameter["sseSpecification"] != _|_ {
				sseSpecification: parameter.sseSpecification
			}
			if parameter["streamSpecification"] != _|_ {
				streamSpecification: parameter.streamSpecification
			}
			if parameter["tableClass"] != _|_ {
				tableClass: parameter.tableClass
			}
			if parameter["tags"] != _|_ {
				tags: parameter.tags
			}
			if parameter["timeToLive"] != _|_ {
				timeToLive: parameter.timeToLive
			}
			if parameter["warmThroughput"] != _|_ {
				warmThroughput: parameter.warmThroughput
			}
		}
	}
	parameter: {
		// +usage=Name of the table in AWS, 3 to 255 characters. Defaults to the component name, so set it when that name is shorter than 3 characters. Cannot change after creation.
		tableName?: string & =~"^[a-zA-Z0-9_.-]{3,255}$"
		// +usage=The attributes used as keys by the table and its indexes. Other attributes are not declared.
		attributeDefinitions!: list.MinItems(1) & [...{
			// +usage=Name of the attribute.
			attributeName!: string & strings.MinRunes(1) & strings.MaxRunes(255)
			// +usage=S for string, N for number, B for binary. Quote it in YAML: a bare N is read as false.
			attributeType!: "S" | "N" | "B"
		}]
		// +usage=The primary key: one HASH key, optionally followed by one RANGE key. Cannot change after creation.
		keySchema!: list.MinItems(1) & list.MaxItems(2) & [...#KeySchemaElement]
		// +usage=How the table is billed. PROVISIONED needs provisionedThroughput; PAY_PER_REQUEST must not have it.
		billingMode!: "PROVISIONED" | "PAY_PER_REQUEST"
		// +usage=Capacity of a PROVISIONED table. Required with PROVISIONED, not allowed with PAY_PER_REQUEST.
		provisionedThroughput?: {
			// +usage=Strongly consistent reads per second.
			readCapacityUnits!: int & >=1
			// +usage=Writes per second.
			writeCapacityUnits!: int & >=1
		}
		// +usage=Upper limits for a PAY_PER_REQUEST table. Not allowed with PROVISIONED.
		onDemandThroughput?: {
			// +usage=Most read request units allowed. At least 1, or -1 to remove the limit.
			maxReadRequestUnits?: int & >=-1
			// +usage=Most write request units allowed. At least 1, or -1 to remove the limit.
			maxWriteRequestUnits?: int & >=-1
			if maxReadRequestUnits != _|_ {
				_validate: {
					"maxReadRequestUnits must be at least 1, or -1 to remove the limit": true
					if maxReadRequestUnits == 0 {
						"maxReadRequestUnits must be at least 1, or -1 to remove the limit": false
					}
				}
			}
			if maxWriteRequestUnits != _|_ {
				_validate: {
					"maxWriteRequestUnits must be at least 1, or -1 to remove the limit": true
					if maxWriteRequestUnits == 0 {
						"maxWriteRequestUnits must be at least 1, or -1 to remove the limit": false
					}
				}
			}
		}
		// +usage=Throughput the table is pre-warmed for.
		warmThroughput?: {
			// +usage=Reads per second the table can serve at once.
			readUnitsPerSecond?: int
			// +usage=Writes per second the table can serve at once.
			writeUnitsPerSecond?: int
		}
		// +usage=Global secondary indexes: a different key over the same items.
		globalSecondaryIndexes?: list.MaxItems(20) & [...{
			// +usage=Name of the index, unique within the table.
			indexName!: string & =~"^[a-zA-Z0-9_.-]{3,255}$"
			// +usage=Key of the index.
			keySchema!: list.MinItems(1) & [...#KeySchemaElement]
			// +usage=Attributes copied into the index.
			projection!: {
				// +usage=ALL copies every attribute, KEYS_ONLY only the keys, INCLUDE the keys plus nonKeyAttributes.
				projectionType?: "ALL" | "KEYS_ONLY" | "INCLUDE"
				// +usage=Extra attributes to copy into the index. Required with projectionType INCLUDE, not allowed otherwise.
				nonKeyAttributes?: list.MinItems(1) & list.MaxItems(20) & [...string]
				if projectionType != _|_ {
					_validate: {
						"nonKeyAttributes is required when projectionType is INCLUDE": true
						if projectionType == "INCLUDE" && nonKeyAttributes == _|_ {
							"nonKeyAttributes is required when projectionType is INCLUDE": false
						}
					}
				}
				if projectionType != _|_ && nonKeyAttributes != _|_ {
					_validate: {
						"nonKeyAttributes is only allowed when projectionType is INCLUDE": true
						if projectionType != "INCLUDE" {
							"nonKeyAttributes is only allowed when projectionType is INCLUDE": false
						}
					}
				}
			}
			// +usage=Capacity of the index on a PROVISIONED table.
			provisionedThroughput?: {
				// +usage=Strongly consistent reads per second.
				readCapacityUnits!: int & >=1
				// +usage=Writes per second.
				writeCapacityUnits!: int & >=1
			}
			// +usage=Upper limits of the index on a PAY_PER_REQUEST table.
			onDemandThroughput?: {
				// +usage=Most read request units allowed. At least 1, or -1 to remove the limit.
				maxReadRequestUnits?: int & >=-1
				// +usage=Most write request units allowed. At least 1, or -1 to remove the limit.
				maxWriteRequestUnits?: int & >=-1
				if maxReadRequestUnits != _|_ {
					_validate: {
						"maxReadRequestUnits must be at least 1, or -1 to remove the limit": true
						if maxReadRequestUnits == 0 {
							"maxReadRequestUnits must be at least 1, or -1 to remove the limit": false
						}
					}
				}
				if maxWriteRequestUnits != _|_ {
					_validate: {
						"maxWriteRequestUnits must be at least 1, or -1 to remove the limit": true
						if maxWriteRequestUnits == 0 {
							"maxWriteRequestUnits must be at least 1, or -1 to remove the limit": false
						}
					}
				}
			}
			// +usage=Throughput the index is pre-warmed for.
			warmThroughput?: {
				// +usage=Reads per second the table can serve at once.
				readUnitsPerSecond?: int
				// +usage=Writes per second the table can serve at once.
				writeUnitsPerSecond?: int
			}
		}]
		// +usage=Local secondary indexes: another sort key under the table's partition key. Cannot change after creation.
		localSecondaryIndexes?: list.MaxItems(5) & [...{
			// +usage=Name of the index, unique within the table.
			indexName!: string & =~"^[a-zA-Z0-9_.-]{3,255}$"
			// +usage=Key of the index. It starts with the table's HASH key.
			keySchema!: list.MinItems(1) & [...#KeySchemaElement]
			// +usage=Attributes copied into the index.
			projection!: {
				// +usage=ALL copies every attribute, KEYS_ONLY only the keys, INCLUDE the keys plus nonKeyAttributes.
				projectionType?: "ALL" | "KEYS_ONLY" | "INCLUDE"
				// +usage=Extra attributes to copy into the index. Required with projectionType INCLUDE, not allowed otherwise.
				nonKeyAttributes?: list.MinItems(1) & list.MaxItems(20) & [...string]
				if projectionType != _|_ {
					_validate: {
						"nonKeyAttributes is required when projectionType is INCLUDE": true
						if projectionType == "INCLUDE" && nonKeyAttributes == _|_ {
							"nonKeyAttributes is required when projectionType is INCLUDE": false
						}
					}
				}
				if projectionType != _|_ && nonKeyAttributes != _|_ {
					_validate: {
						"nonKeyAttributes is only allowed when projectionType is INCLUDE": true
						if projectionType != "INCLUDE" {
							"nonKeyAttributes is only allowed when projectionType is INCLUDE": false
						}
					}
				}
			}
		}]
		// +usage=DynamoDB Streams settings.
		streamSpecification?: {
			// +usage=Whether DynamoDB Streams records changes to the table.
			streamEnabled!: bool
			// +usage=What each stream record carries.
			streamViewType?: "KEYS_ONLY" | "NEW_IMAGE" | "OLD_IMAGE" | "NEW_AND_OLD_IMAGES"
		}
		// +usage=Server-side encryption settings.
		sseSpecification?: {
			// +usage=True encrypts with a KMS key; false or unset uses a key owned by AWS.
			enabled?: bool
			// +usage=Encryption type.
			sseType?: "AES256" | "KMS"
			// +usage=KMS key ID, ARN or alias to use instead of the AWS managed key.
			kmsMasterKeyID?: string
		}
		// +usage=Automatic deletion of expired items.
		timeToLive?: {
			// +usage=Attribute holding each item's expiry time, in epoch seconds.
			attributeName!: string & strings.MinRunes(1) & strings.MaxRunes(255)
			// +usage=Whether items are deleted after their expiry time.
			enabled!: bool
		}
		// +usage=Point-in-time recovery settings.
		continuousBackups?: {
			// +usage=Whether the table can be restored to any second in the recovery period.
			pointInTimeRecoveryEnabled?: bool
			// +usage=How far back a restore can go. DynamoDB uses 35 when unset.
			recoveryPeriodInDays?: int & >=1 & <=35
		}
		// +usage=Whether CloudWatch Contributor Insights is on for the table.
		contributorInsights?: "ENABLED" | "DISABLED"
		// +usage=Refuse to delete the table while enabled.
		deletionProtectionEnabled?: bool
		// +usage=Storage class of the table.
		tableClass?: "STANDARD" | "STANDARD_INFREQUENT_ACCESS"
		// +usage=Resource-based IAM policy for the table, as a JSON document of at most 20 KB.
		resourcePolicy?: string & strings.MaxRunes(20480)
		// +usage=AWS tags for the table.
		tags?: list.MaxItems(50) & [...{
			// +usage=Tag key.
			key!: string & strings.MinRunes(1) & strings.MaxRunes(128)
			// +usage=Tag value.
			value!: string & strings.MaxRunes(256)
		}]
		// +usage=What ACK does with the table in AWS when the component is deleted. ACK deletes it when unset.
		deletionPolicy?: "delete" | "retain"
		_validate: {
			"provisionedThroughput is required when billingMode is PROVISIONED": true
			if parameter.billingMode == "PROVISIONED" && parameter["provisionedThroughput"] == _|_ {
				"provisionedThroughput is required when billingMode is PROVISIONED": false
			}
		}
		_validate: {
			"provisionedThroughput is not allowed when billingMode is PAY_PER_REQUEST": true
			if parameter.billingMode == "PAY_PER_REQUEST" && parameter["provisionedThroughput"] != _|_ {
				"provisionedThroughput is not allowed when billingMode is PAY_PER_REQUEST": false
			}
		}
		_validate: {
			"onDemandThroughput is not allowed when billingMode is PROVISIONED": true
			if parameter.billingMode == "PROVISIONED" && parameter["onDemandThroughput"] != _|_ {
				"onDemandThroughput is not allowed when billingMode is PROVISIONED": false
			}
		}
	}
	#KeySchemaElement: {
		// +usage=Name of a key attribute. It must also appear in attributeDefinitions.
		attributeName!: string & strings.MinRunes(1) & strings.MaxRunes(255)
		// +usage=HASH for the partition key, RANGE for the sort key.
		keyType!: "HASH" | "RANGE"
	}
}
