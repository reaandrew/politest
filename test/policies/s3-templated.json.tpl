{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowS3ForBucket",
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:PutObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::{{.bucket_name}}",
        "arn:aws:s3:::{{.bucket_name}}/*"
      ],
      "Condition": {
        "StringEquals": {
          "aws:PrincipalTag/Department": "{{.department}}"
        }
      }
    }
  ]
}
