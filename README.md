# Terraform Provider for Woodpecker CI

A Terraform provider used to interact with [Woodpecker CI](https://woodpecker-ci.org/) resources.

This fork is based on [Kichiyaki's provider](https://github.com/Kichiyaki/terraform-provider-woodpecker).
It adds management of cron job enabled state and reconciles repository approval
and activation state.

## Install

```hcl
terraform {
  required_providers {
    woodpecker = {
      source  = "registry.terraform.io/jamesbraid/woodpecker"
      version = "0.6.1"
    }
  }
}
```

## Local development

**Prerequisites:**

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.23
- [Docker](https://docs.docker.com/engine/install/) (for tests)
- [pre-commit](https://pre-commit.com/) (optional, but recommended)
- [Node.js](https://nodejs.org/en) (LTS, only needed for commitlint)
- [direnv](https://direnv.net/) (optional, but recommended)

```shell
# if you have direnv installed
direnv allow

# install git hooks and required tools (terraform-plugin-docs, golangci-lint)
make install

# run tests
go test ./...
```

## Contact

Dawid Wysokiński - [contact@dwysokinski.me](mailto:contact@dwysokinski.me)
