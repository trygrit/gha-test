# Terraform Pull Request Commenter

A GitHub Action that adds opinionated comments to pull requests based on Terraform command outputs.

## Features

- Automatically comments on pull requests with Terraform command outputs
- Supports `fmt`, `init`, and `plan` commands
- Formats output with syntax highlighting
- Handles both success and failure cases
- Uses GitHub's built-in environment variables

## Usage

```yaml
name: Terraform

on:
  pull_request:
    paths:
      - '**.tf'
      - '**.tfvars'

jobs:
  terraform:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Terraform
        uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: "1.12.0"

      - name: Terraform Init
        id: init
        run: |
          set -o pipefail
          terraform init 2>&1 | tee -a $GITHUB_OUTPUT
          echo "exitcode=$?" >> $GITHUB_OUTPUT

      - name: Post Init Results
        uses: trygrit/gha-test@v1
        with:
          type: init
          output: ${{ steps.init.outputs.stdout }}
          exit_code: ${{ steps.init.outputs.exitcode }}

      - name: Terraform Plan
        id: plan
        run: |
          set -o pipefail
          terraform plan 2>&1 | tee -a $GITHUB_OUTPUT
          echo "exitcode=$?" >> $GITHUB_OUTPUT

      - name: Post Plan Results
        if: always()
        uses: trygrit/gha-test@v1
        with:
          type: plan
          output: ${{ steps.plan.outputs.stdout }}
          exit_code: ${{ steps.plan.outputs.exitcode }}
```

## Inputs

| Name | Description | Required | Default |
|------|-------------|----------|---------|
| `type` | The type of comment (fmt, init, plan) | Yes | - |
| `output` | The terraform command output | Yes | - |
| `exit_code` | The exit code from the terraform command | Yes | "0" |

## Outputs

The action will post a comment on the pull request with the formatted output of the Terraform command.

## Development

### Prerequisites

- Go 1.21 or later
- Docker
- Make

### Building

```bash
make build
```

### Testing

```bash
make test
```

### Releasing

1. Update the version in `action.yml`
2. Create a new tag
3. Push the tag to GitHub

## License

MIT
