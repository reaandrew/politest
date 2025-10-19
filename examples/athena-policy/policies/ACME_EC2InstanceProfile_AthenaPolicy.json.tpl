{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AthenaListActions",
      "Effect": "Allow",
      "Action": [
        "athena:ListEngineVersions",
        "athena:ListTagsForResource",
        "athena:ListWorkGroups"
      ],
      "Resource": "*"
    },
    {
      "Sid": "AthenaWorkgroupActions",
      "Effect": "Allow",
      "Action": [
        "athena:BatchGetNamedQuery",
        "athena:BatchGetPreparedStatement",
        "athena:BatchGetQueryExecution",
        "athena:CreateNamedQuery",
        "athena:CreatePreparedStatement",
        "athena:DeleteNamedQuery",
        "athena:DeletePreparedStatement",
        "athena:GetNamedQuery",
        "athena:GetPreparedStatement",
        "athena:GetQueryExecution",
        "athena:GetQueryResults",
        "athena:GetQueryResultsStream",
        "athena:GetQueryRuntimeStatistics",
        "athena:GetWorkGroup",
        "athena:ListNamedQueries",
        "athena:ListPreparedStatements",
        "athena:ListQueryExecutions",
        "athena:StartQueryExecution",
        "athena:StopQueryExecution",
        "athena:UpdateNamedQuery",
        "athena:UpdatePreparedStatement"
      ],
      "Resource": [
        "arn:aws:athena:{{.region}}:{{.dns_hub_dev_account_id}}:workgroup/*",
        "arn:aws:athena:{{.region}}:{{.dns_hub_live_account_id}}:workgroup/*",
        "arn:aws:athena:{{.region}}:{{.dummy_account_id}}:workgroup/*"
      ],
      "Condition": {
        "StringEquals": {
          "aws:CalledVia": "athena.amazonaws.com"
        },
        "StringLike": {
          "aws:ResourceOrgPaths": "{{.data_plane_nonprod_org_path}}/*"
        },
        "StringEqualsIfExists": {
          "aws:VpceAccount": "{{.dns_hub_dev_account_id}}"
        }
      }
    },
    {
      "Sid": "DenyAthenaOnPrimaryWorkgroup",
      "Effect": "Deny",
      "Action": "athena:*",
      "Resource": "arn:aws:athena:*:*:workgroup/primary"
    },
    {
      "Sid": "DenyAthenaWithLiveVpce",
      "Effect": "Deny",
      "Action": "athena:*",
      "Resource": "arn:aws:athena:*:*:workgroup/*",
      "Condition": {
        "StringEquals": {
          "aws:VpceAccount": "{{.dns_hub_live_account_id}}"
        },
        "StringLike": {
          "aws:ResourceOrgPaths": "{{.data_plane_nonprod_org_path}}/*"
        }
      }
    },
    {
      "Sid": "DenyAthenaOnProdOrgPath",
      "Effect": "Deny",
      "Action": "athena:*",
      "Resource": "arn:aws:athena:*:*:workgroup/*",
      "Condition": {
        "StringLike": {
          "aws:ResourceOrgPaths": "{{.data_plane_prod_org_path}}/*"
        }
      }
    },
    {
      "Sid": "S3BucketReadActions",
      "Effect": "Allow",
      "Action": [
        "s3:GetBucketLocation",
        "s3:ListBucket"
      ],
      "Resource": "arn:aws:s3:::*",
      "Condition": {
        "StringEquals": {
          "aws:CalledVia": "athena.amazonaws.com",
          "s3:ResourceAccount": "{{.dummy_account_id}}"
        }
      }
    },
    {
      "Sid": "S3ObjectActions",
      "Effect": "Allow",
      "Action": [
        "s3:AbortMultipartUpload",
        "s3:GetObject",
        "s3:ListMultipartUploadParts",
        "s3:PutObject"
      ],
      "Resource": "arn:aws:s3:::*/*",
      "Condition": {
        "StringEquals": {
          "aws:CalledVia": "athena.amazonaws.com",
          "s3:ResourceAccount": "{{.dummy_account_id}}"
        },
        "StringLike": {
          "s3:x-amz-server-side-encryption-aws-kms-key-id": "arn:aws:kms:{{.region}}:{{.dummy_account_id}}:key/*"
        }
      }
    },
    {
      "Sid": "DenyS3BucketManagement",
      "Effect": "Deny",
      "Action": [
        "s3:CreateBucket",
        "s3:CreateBucketMetadataTableConfiguration",
        "s3:DeleteBucket",
        "s3:DeleteBucketMetadataTableConfiguration",
        "s3:DeleteBucketPolicy",
        "s3:DeleteBucketWebsite",
        "s3:PutAccelerateConfiguration",
        "s3:PutAnalyticsConfiguration",
        "s3:PutBucketAcl",
        "s3:PutBucketCORS",
        "s3:PutBucketLogging",
        "s3:PutBucketNotification",
        "s3:PutBucketObjectLockConfiguration",
        "s3:PutBucketOwnershipControls",
        "s3:PutBucketPolicy",
        "s3:PutBucketPublicAccessBlock",
        "s3:PutBucketRequestPayment",
        "s3:PutBucketTagging",
        "s3:PutBucketVersioning",
        "s3:PutBucketWebsite",
        "s3:PutEncryptionConfiguration",
        "s3:PutIntelligentTieringConfiguration",
        "s3:PutInventoryConfiguration",
        "s3:PutLifecycleConfiguration",
        "s3:PutMetricsConfiguration"
      ],
      "Resource": "arn:aws:s3:::{{.dummy_account_id}}-*"
    },
    {
      "Sid": "DenyS3ObjectManagementOnAccountBuckets",
      "Effect": "Deny",
      "Action": [
        "s3:DeleteObject",
        "s3:DeleteObjectTagging",
        "s3:DeleteObjectVersion",
        "s3:DeleteObjectVersionTagging",
        "s3:GetObject",
        "s3:GetObjectAcl",
        "s3:GetObjectAttributes",
        "s3:GetObjectLegalHold",
        "s3:GetObjectRetention",
        "s3:GetObjectTagging",
        "s3:GetObjectTorrent",
        "s3:GetObjectVersion",
        "s3:GetObjectVersionAcl",
        "s3:GetObjectVersionAttributes",
        "s3:GetObjectVersionForReplication",
        "s3:GetObjectVersionTagging",
        "s3:GetObjectVersionTorrent",
        "s3:PutObject",
        "s3:PutObjectAcl",
        "s3:PutObjectLegalHold",
        "s3:PutObjectRetention",
        "s3:PutObjectTagging",
        "s3:PutObjectVersionAcl",
        "s3:PutObjectVersionTagging",
        "s3:RestoreObject"
      ],
      "Resource": "arn:aws:s3:::{{.dummy_account_id}}-*/*"
    },
    {
      "Sid": "DenyS3ObjectAcl",
      "Effect": "Deny",
      "Action": [
        "s3:PutObjectAcl",
        "s3:PutObjectVersionAcl"
      ],
      "Resource": "arn:aws:s3:::*/*"
    },
    {
      "Sid": "DenyS3WithoutCalledVia",
      "Effect": "Deny",
      "Action": [
        "s3:AbortMultipartUpload",
        "s3:BypassGovernanceRetention",
        "s3:DeleteObject",
        "s3:DeleteObjectTagging",
        "s3:DeleteObjectVersion",
        "s3:DeleteObjectVersionTagging",
        "s3:GetObject",
        "s3:GetObjectAcl",
        "s3:GetObjectAttributes",
        "s3:GetObjectLegalHold",
        "s3:GetObjectRetention",
        "s3:GetObjectTagging",
        "s3:GetObjectVersion",
        "s3:GetObjectVersionAcl",
        "s3:GetObjectVersionAttributes",
        "s3:GetObjectVersionTagging",
        "s3:ListMultipartUploadParts",
        "s3:PutObject",
        "s3:PutObjectAcl",
        "s3:PutObjectLegalHold",
        "s3:PutObjectRetention",
        "s3:PutObjectTagging",
        "s3:PutObjectVersionAcl",
        "s3:PutObjectVersionTagging",
        "s3:RestoreObject"
      ],
      "Resource": "arn:aws:s3:::*/*",
      "Condition": {
        "StringNotEquals": {
          "aws:CalledVia": "athena.amazonaws.com"
        }
      }
    },
    {
      "Sid": "DenyS3PutObjectWithoutEncryption",
      "Effect": "Deny",
      "Action": "s3:PutObject",
      "Resource": "arn:aws:s3:::*/*",
      "Condition": {
        "Null": {
          "s3:x-amz-server-side-encryption-aws-kms-key-id": "true"
        }
      }
    },
    {
      "Sid": "DenyS3WrongResourceAccount",
      "Effect": "Deny",
      "Action": "s3:*",
      "Resource": "arn:aws:s3:::*",
      "Condition": {
        "StringNotEquals": {
          "s3:ResourceAccount": "{{.dummy_account_id}}"
        }
      }
    }
  ]
}
