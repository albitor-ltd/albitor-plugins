# Run by scripts/verify-deploy-byoa-aws.sh --composed against the tree the real composer wrote for a
# BYOA profile: the rendered stack reads that tree's own albitor.yaml and plans, offline.
mock_provider "aws" {
  mock_data "aws_iam_policy_document" {
    defaults = {
      json = "{\"Version\":\"2012-10-17\",\"Statement\":[]}"
    }
  }
}

mock_provider "aws" {
  alias = "us_east_1"
}

mock_provider "archive" {}

variables {
  project_name = "plantrain"
}

run "the_composed_manifest_plans" {
  command = plan

  assert {
    condition     = length(module.site) == 1 && length(module.api) == 1
    error_message = "the default shell has a site and an API"
  }
  assert {
    condition     = module.data.table_names["data"] == "plantrain-dev-data"
    error_message = "the table data-dynamodb declares is created, named from the project"
  }
  assert {
    condition     = module.data.table_env["ALBITOR_TABLE_DATA"] == "plantrain-dev-data"
    error_message = "the store's variable is handed to the function"
  }
}
