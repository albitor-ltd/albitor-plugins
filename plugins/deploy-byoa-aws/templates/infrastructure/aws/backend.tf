# Partial backend: the bucket, key and region are supplied at `terraform init` time with
# -backend-config by .github/workflows/deploy.yml, because the state bucket is per account
# (albitor-tfstate-<account id>) and not knowable when the repository is composed.
#
# Locking is S3-native (use_lockfile): a sibling object `<key>.tflock` in the same bucket. Every
# plan and apply passes -lock-timeout so a live lock is waited out; a stale one, left by a killed
# apply, is cleared by scripts/tf-unlock-stale.sh before init.
terraform {
  backend "s3" {
    use_lockfile = true
    encrypt      = true
  }
}
