# AWS MFA Credential Helper

This is a simple CLI utility for helping create IAM credentials without MFA from those that do. This is designed to
help when running commands that requires programmatic aws access, but don't support MFA, e.g. serverless framework deployment.

That way this works is by reading source credentials from the `~/.aws/credentials` file, generating a short lives STS session,
and then writing a new profile back, which can then be used like a normal aws profile.

Because this utility is written in Go it operates as a standalone binary that is not reliant on any system dependencies,
this is helpful in not requiring things like Python to be installed when you don't need it for your project.

## Configuration

By default the tool will ask a series of questions, they can however be added as flags. This means you automate the entire process;

```shell
aws-mfa --src=default --dst=default-mfa --device=arn:aws:iam::000000000000:mfa/iamuser --ttl=3600 --overwrite --code=123456
```

| Name        | Description                            |
| ----------- | -------------------------------------- |
| `src`       | Name of the source profile             |
| `dst`       | Name of the destination profile        |
| `device`    | ARN of the MFA device                  |
| `ttl`       | STS Session lifetime                   |
| `code`      | MFA code                               |
| `overwrite` | Overwrite existing destination profile |
