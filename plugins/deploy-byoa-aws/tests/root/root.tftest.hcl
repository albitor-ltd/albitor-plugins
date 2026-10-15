# The managed root stack, planned offline against the real modules (the verify script points the
# git module sources at this checkout) with the AWS provider mocked. Each run reads a manifest
# fixture, so what the stack renders is a function of albitor.yaml alone.
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
  project_name  = "plantrain"
  environment   = "dev"
  dns_mode      = "none"
  manifest_path = "fixtures/full.yaml"
}

run "a_full_manifest_renders_the_whole_stack" {
  command = plan

  assert {
    condition     = length(module.data.table_names) == 2 && module.data.table_names["data"] == "plantrain-dev-data" && module.data.table_names["orders"] == "plantrain-dev-orders"
    error_message = "one table per services.tables entry, named from the project and environment"
  }
  assert {
    condition     = length(module.api) == 1 && length(module.site) == 1 && length(module.alarms) == 1
    error_message = "site, http and the alarms are all rendered"
  }
  assert {
    condition     = module.api[0].function_name == "plantrain-dev-api"
    error_message = "the function is named from the prefix"
  }
}

run "the_tables_reach_the_function_and_its_role" {
  command = plan

  assert {
    condition     = length(module.data.table_arns) == 2
    error_message = "the role is granted both tables"
  }
  assert {
    condition     = module.data.table_env["ALBITOR_TABLE_DATA"] == "plantrain-dev-data"
    error_message = "the function is handed ALBITOR_TABLE_DATA"
  }
}

run "a_static_manifest_renders_no_api_and_no_alarms_on_one" {
  command = plan
  variables {
    manifest_path = "fixtures/static.yaml"
  }

  assert {
    condition     = length(module.api) == 0 && length(module.site) == 1
    error_message = "no http block, no function and no gateway"
  }
  assert {
    condition     = length(module.data.table_names) == 0
    error_message = "no tables declared, none created"
  }
}

run "an_api_only_manifest_renders_no_site" {
  command = plan
  variables {
    manifest_path = "fixtures/api-only.yaml"
  }

  assert {
    condition     = length(module.api) == 1 && length(module.site) == 0
    error_message = "no site block, no bucket and no distribution"
  }
}

run "a_custom_domain_is_a_choice_made_in_tfvars" {
  command = plan
  variables {
    dns_mode    = "external_delegated"
    domain_name = "app.example.com"
  }

  # The real provider reports the validation record's domain at plan; the mock cannot.
  override_resource {
    target          = module.site[0].aws_acm_certificate.cert[0]
    override_during = plan
    values = {
      domain_validation_options = [{
        domain_name           = "app.example.com"
        resource_record_name  = "_x.app.example.com."
        resource_record_type  = "CNAME"
        resource_record_value = "_y.acm-validations.aws."
      }]
    }
  }

  assert {
    condition     = length(module.site) == 1
    error_message = "the stack plans with a custom domain"
  }
}

run "an_unknown_dns_mode_is_refused" {
  command = plan
  variables {
    dns_mode = "route66"
  }
  expect_failures = [var.dns_mode]
}

run "a_manifest_that_needs_what_the_target_does_not_render_is_refused_auth" {
  command = plan
  variables {
    manifest_path = "fixtures/unsupported-auth.yaml"
  }
  expect_failures = [terraform_data.manifest]
}

run "a_manifest_that_needs_what_the_target_does_not_render_is_refused_objects" {
  command = plan
  variables {
    manifest_path = "fixtures/unsupported-objects.yaml"
  }
  expect_failures = [terraform_data.manifest]
}

run "a_manifest_that_needs_what_the_target_does_not_render_is_refused_schedules" {
  command = plan
  variables {
    manifest_path = "fixtures/unsupported-schedules.yaml"
  }
  expect_failures = [terraform_data.manifest]
}

run "a_manifest_that_needs_what_the_target_does_not_render_is_refused_config" {
  command = plan
  variables {
    manifest_path = "fixtures/unsupported-config.yaml"
  }
  expect_failures = [terraform_data.manifest]
}

run "another_manifest_version_is_refused" {
  command = plan
  variables {
    manifest_path = "fixtures/bad-version.yaml"
  }
  expect_failures = [terraform_data.manifest]
}

run "a_runtime_other_than_go_is_refused" {
  command = plan
  variables {
    manifest_path = "fixtures/bad-runtime.yaml"
  }
  expect_failures = [terraform_data.manifest]
}

run "an_app_with_nothing_to_serve_is_refused" {
  command = plan
  variables {
    manifest_path = "fixtures/nothing-to-serve.yaml"
  }
  expect_failures = [terraform_data.manifest]
}

run "a_static_output_the_pipeline_does_not_ship_is_refused" {
  command = plan
  variables {
    manifest_path = "fixtures/bad-static-output.yaml"
  }
  expect_failures = [terraform_data.manifest]
}

run "an_api_output_the_pipeline_does_not_ship_is_refused" {
  command = plan
  variables {
    manifest_path = "fixtures/bad-api-output.yaml"
  }
  expect_failures = [terraform_data.manifest]
}

run "a_project_name_that_is_not_a_plain_name_is_refused" {
  command = plan
  variables {
    project_name = "Plan Train;"
  }
  expect_failures = [var.project_name]
}
