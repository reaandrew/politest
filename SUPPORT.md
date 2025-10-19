# Support

## Getting Help

If you need help with `politest`, here are your options:

### Documentation

- **[README.md](README.md)** - Quick start guide and usage examples
- **[CLAUDE.md](CLAUDE.md)** - Detailed architecture and implementation guide
- **[Wiki](docs/wiki/)** - Comprehensive documentation including:
  - [Getting Started](docs/wiki/Getting-Started.md)
  - [Installation and Setup](docs/wiki/Installation-and-Setup.md)
  - [Scenario Formats](docs/wiki/Scenario-Formats.md)
  - [Template Variables](docs/wiki/Template-Variables.md)
  - [API Reference](docs/wiki/API-Reference.md)
- **[ARCHITECTURE.md](docs/ARCHITECTURE.md)** - Technical architecture and design decisions

### GitHub Issues

For bugs, feature requests, or questions:

1. **Search existing issues** - Your question may already be answered
   - [All issues](https://github.com/reaandrew/politest/issues)
   - [Bug reports](https://github.com/reaandrew/politest/labels/bug)
   - [Feature requests](https://github.com/reaandrew/politest/labels/enhancement)

2. **Create a new issue** using the appropriate template:
   - [Bug Report](.github/ISSUE_TEMPLATE/bug_report.md)
   - [Feature Request](.github/ISSUE_TEMPLATE/feature_request.md)

### Common Questions

#### Q: How do I test cross-account policies?

See the [Scenario Formats](docs/wiki/Scenario-Formats.md) documentation for examples using `caller_arn` and `resource_owner`.

#### Q: Why are my context conditions not working as expected?

AWS SimulateCustomPolicy has limitations. Some complex conditions may not evaluate correctly in simulation. See the troubleshooting section in [Getting Started](docs/wiki/Getting-Started.md).

#### Q: How do I debug policy evaluation?

Use the `--save` flag to capture the raw AWS API response:
```bash
politest --scenario test.yml --save /tmp/response.json
```

Then inspect the `MatchedStatements` field to see which policy statements applied.

#### Q: Can I use this tool in CI/CD?

Yes! Use `expect:` assertions in your scenarios and check the exit code. The tool exits with code 1 if any assertions fail. See [CONTRIBUTING.md](CONTRIBUTING.md) for CI integration examples.

### Response Time

- **Bug reports** - We aim to respond within 48 hours
- **Feature requests** - We review and prioritize weekly
- **Security issues** - Report via [SECURITY.md](SECURITY.md) (24-hour response for critical issues)

### Contributing

Want to contribute? See [CONTRIBUTING.md](CONTRIBUTING.md) for:
- Development setup
- Coding standards
- Pull request process
- Testing requirements

### Security Vulnerabilities

**Do not report security vulnerabilities through public GitHub issues.**

Please follow the process in [SECURITY.md](SECURITY.md) to report security issues responsibly.

### AWS IAM Policy Assistance

This tool helps you test IAM policies, but does not provide general AWS IAM consulting. For AWS-specific help:

- [AWS IAM Documentation](https://docs.aws.amazon.com/IAM/latest/UserGuide/)
- [AWS Support](https://aws.amazon.com/support/)
- [AWS re:Post](https://repost.aws/)

### Project Maintainers

This project is actively maintained. We review issues and pull requests regularly.

**Note:** This is an open-source project maintained by volunteers. While we strive to help everyone, response times may vary based on complexity and maintainer availability.
