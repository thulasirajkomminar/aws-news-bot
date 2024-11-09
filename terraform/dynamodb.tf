module "table" {
  source  = "schubergphilis/mcaf-dynamodb/aws"
  version = "~> 0.2.0"

  name               = local.name
  hash_key           = "GUID"
  kms_key_arn        = null
  ttl_attribute_name = "ExpiresAt"
  ttl_enabled        = true
  tags               = local.tags

  attributes = [
    {
      name = "GUID"
      type = "S"
    },
  ]
}
