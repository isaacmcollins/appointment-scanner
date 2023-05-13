resource "aws_iam_role_policy_attachment" "lambda_policy_attachment" {
  role       = aws_iam_role.lambda_role.name
  policy_arn = aws_iam_policy.lambda_iam_policy.arn
}

resource "aws_iam_role_policy_attachment" "lambda_policy_poller_attachment" {
  role       = aws_iam_role.lambda_poller_role.name
  policy_arn = aws_iam_policy.lambda_poller_policy.arn
}

resource "aws_iam_role_policy_attachment" "scheduler_trigger_attachment" {
  role       = aws_iam_role.scheduler_role.name
  policy_arn = aws_iam_policy.scheduler_iam_trigger_policy.arn
}

resource "aws_iam_role" "scheduler_role" {
  name               = "appointment-scanner-scheduler-role"
  assume_role_policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "Service": "scheduler.amazonaws.com"
            },
            "Action": "sts:AssumeRole"
        }
    ]
}
EOF
  tags               = local.tags
}

resource "aws_iam_role" "lambda_role" {
  name               = "appointment-scanner-lambda-role"
  assume_role_policy = <<EOF
{
 "Version": "2012-10-17",
 "Statement": [
   {
     "Action": "sts:AssumeRole",
     "Principal": {
       "Service": "lambda.amazonaws.com"
     },
     "Effect": "Allow",
     "Sid": ""
   }
 ]
}
EOF
  tags               = local.tags
}

resource "aws_iam_role" "lambda_poller_role" {
  name               = "appointment-scanner-poller-role"
  assume_role_policy = <<EOF
{
 "Version": "2012-10-17",
 "Statement": [
   {
     "Action": "sts:AssumeRole",
     "Principal": {
       "Service": "lambda.amazonaws.com"
     },
     "Effect": "Allow",
     "Sid": ""
   }
 ]
}
EOF
  tags               = local.tags
}


resource "aws_iam_policy" "lambda_iam_policy" {

  name        = "aws_iam_policy_for_terraform_aws_lambda_role"
  path        = "/"
  description = "AWS IAM Policy for managing aws lambda role"
  policy      = <<EOF
{
 "Version": "2012-10-17",
 "Statement": [
   {
     "Action": [
       "logs:CreateLogGroup",
       "logs:CreateLogStream",
       "logs:PutLogEvents"
     ],
     "Resource": "arn:aws:logs:*:*:*",
     "Effect": "Allow"
   }
 ]
}
EOF
  tags        = local.tags
}

resource "aws_iam_policy" "scheduler_iam_trigger_policy" {

  name        = "aws_iam_policy_trigger_poller"
  path        = "/"
  description = "AWS IAM Policy for triggering poller"
  policy      = <<EOF
{
 "Version": "2012-10-17",
 "Statement": [
   {
     "Action": [
       "lambda:InvokeFunction"
     ],
     "Resource": "${aws_lambda_function.poller.arn}",
     "Effect": "Allow"
   }
 ]
}
EOF

  tags = local.tags
}


resource "aws_iam_policy" "lambda_poller_policy" {
  name        = "lambda_poller_iam_policy"
  path        = "/"
  description = "AWS IAM Policy for managing aws lambda role"
  policy      = <<EOF
{
 "Version": "2012-10-17",
 "Statement": [
    {
    	"Effect": "Allow",
    	"Action": [
    		"dynamodb:BatchGetItem",
        "dynamodb:GetItem",
    		"dynamodb:Query",
    		"dynamodb:Scan",
    		"dynamodb:BatchWriteItem",
    		"dynamodb:PutItem",
    		"dynamodb:UpdateItem"
    	],
    	"Resource": "${aws_dynamodb_table.state-store.arn}"
    },
    {
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "arn:aws:logs:*:*:*",
      "Effect": "Allow"
    }
  ]
}
EOF
  tags        = local.tags
}

