## [3.4.5](https://github.com/reaandrew/politest/compare/v3.4.4...v3.4.5) (2025-11-05)

## [3.4.4](https://github.com/reaandrew/politest/compare/v3.4.3...v3.4.4) (2025-10-23)

## [3.4.3](https://github.com/reaandrew/politest/compare/v3.4.2...v3.4.3) (2025-10-23)

## [3.4.2](https://github.com/reaandrew/politest/compare/v3.4.1...v3.4.2) (2025-10-23)

## [3.4.1](https://github.com/reaandrew/politest/compare/v3.4.0...v3.4.1) (2025-10-23)

## [3.4.0](https://github.com/reaandrew/politest/compare/v3.3.0...v3.4.0) (2025-10-23)

### Features

* add --test flag to run specific named tests ([#16](https://github.com/reaandrew/politest/issues/16)) ([33d50ff](https://github.com/reaandrew/politest/commit/33d50ff288a47680abe6044cfcb170bc96fb8ca3)), closes [#15](https://github.com/reaandrew/politest/issues/15)
* send pretty-printed JSON to AWS API for better error messages ([#17](https://github.com/reaandrew/politest/issues/17)) ([e3f4c70](https://github.com/reaandrew/politest/commit/e3f4c706faac0f73e0aa8ece4df71ffeeb00cc8a)), closes [#14](https://github.com/reaandrew/politest/issues/14)
* strip non-IAM fields from policies and add --strict-policy flag ([#18](https://github.com/reaandrew/politest/issues/18)) ([77adc6f](https://github.com/reaandrew/politest/commit/77adc6fff4e0803811a1419730ee309be1ed3492)), closes [#13](https://github.com/reaandrew/politest/issues/13)

### Bug Fixes

* **security:** pin GitHub Actions to commit SHAs in claude workflow ([#28](https://github.com/reaandrew/politest/issues/28)) ([080028e](https://github.com/reaandrew/politest/commit/080028e53bb9255697985eaf3986c107497d4920))

## [3.3.0](https://github.com/reaandrew/politest/compare/v3.2.0...v3.3.0) (2025-10-22)

### Features

* support context key override at test level ([#19](https://github.com/reaandrew/politest/issues/19)) ([c694198](https://github.com/reaandrew/politest/commit/c694198a337ab647085df007d2904a4243b4b10f)), closes [#11](https://github.com/reaandrew/politest/issues/11)

## [3.2.0](https://github.com/reaandrew/politest/compare/v3.1.0...v3.2.0) (2025-10-22)

### Features

* show matched statement source and JSON on test failures ([#22](https://github.com/reaandrew/politest/issues/22)) ([cea0765](https://github.com/reaandrew/politest/commit/cea07651abfaabd9a9c8efb2e8e7cfcdc10575e5))

## [3.1.0](https://github.com/reaandrew/politest/compare/v3.0.0...v3.1.0) (2025-10-21)

### Features

* run integration and example tests on pull requests ([9255327](https://github.com/reaandrew/politest/commit/9255327b7f24e5d69bfa9f1b1bdf807fa8f97346))

## [3.0.0](https://github.com/reaandrew/politest/compare/v2.3.1...v3.0.0) (2025-10-21)

### ⚠ BREAKING CHANGES

* Refactor run() to extract prepareSimulation() function

- Create prepareSimulation() function that handles all AWS-free preparation
- This function loads scenarios, renders templates, processes variables
- Unit tests now call prepareSimulation() instead of run()
- Tests no longer attempt AWS API calls or require credentials
- Tests run in ~3s instead of timing out waiting for AWS
- Fixes CI test failures where AWS credentials are unavailable
- Coverage: 89.2% overall (72.0% main, 96.1% internal)

Integration tests in test/ directory still use full AWS workflow.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>

### Features

* add --debug flag to show file loading and template rendering ([2d32727](https://github.com/reaandrew/politest/commit/2d32727017bffb91f5d7004fd0c46cd552bb411a)), closes [#10](https://github.com/reaandrew/politest/issues/10)

### Bug Fixes

* separate AWS-free logic from AWS calls in tests ([4522bf8](https://github.com/reaandrew/politest/commit/4522bf8e02efc891f5edc8349d20a8c857c86580))

## [2.3.1](https://github.com/reaandrew/politest/compare/v2.3.0...v2.3.1) (2025-10-20)

### Bug Fixes

* allow GitGuardian to gracefully skip on fork PRs ([ba2c8be](https://github.com/reaandrew/politest/commit/ba2c8be7eb96a43d9739f677e635f660a7163175))

## [2.3.0](https://github.com/reaandrew/politest/compare/v2.2.0...v2.3.0) (2025-10-20)

### Features

* make example-tests job output more verbose ([480259c](https://github.com/reaandrew/politest/commit/480259c5929036f2da54e56566c16d7f1270423f))

## [2.2.0](https://github.com/reaandrew/politest/compare/v2.1.11...v2.2.0) (2025-10-20)

### Features

* add GitHub native attestations to SLSA workflow ([81bbad5](https://github.com/reaandrew/politest/commit/81bbad5a03178a7d2539d553f196012f2620cdcc))

## [2.1.11](https://github.com/reaandrew/politest/compare/v2.1.10...v2.1.11) (2025-10-20)

### Bug Fixes

* use PAT token for semantic-release to trigger SLSA workflow ([6fb6e7b](https://github.com/reaandrew/politest/commit/6fb6e7b3b5092c712f0512bcf381b296e5f3de9a))

## [2.1.10](https://github.com/reaandrew/politest/compare/v2.1.9...v2.1.10) (2025-10-20)

### Bug Fixes

* trigger SLSA workflow on release published events ([694bd9b](https://github.com/reaandrew/politest/commit/694bd9be683685ad9c97527b99104fa522554799))

## [2.1.9](https://github.com/reaandrew/politest/compare/v2.1.8...v2.1.9) (2025-10-20)

### Bug Fixes

* use ISO 8601 basic format for build dates in SLSA workflow ([188925f](https://github.com/reaandrew/politest/commit/188925fbdbbeefba5e928af41b583352df979c61))

## [2.1.8](https://github.com/reaandrew/politest/compare/v2.1.7...v2.1.8) (2025-10-20)

### Bug Fixes

* use GO-prefixed environment variable names for SLSA builder ([cc898f7](https://github.com/reaandrew/politest/commit/cc898f76bb28944294f5f26b71f47ff5cff8564a))

## [2.1.7](https://github.com/reaandrew/politest/compare/v2.1.6...v2.1.7) (2025-10-20)

### Bug Fixes

* use newline-separated environment variables in evaluated-envs ([0f437e3](https://github.com/reaandrew/politest/commit/0f437e3ded3524a4dada5acb2095ea315b4e1548))

## [2.1.6](https://github.com/reaandrew/politest/compare/v2.1.5...v2.1.6) (2025-10-20)

### Bug Fixes

* use equals sign for environment variable assignment in evaluated-envs ([515c089](https://github.com/reaandrew/politest/commit/515c089a907e2cc3a402c286cdb7e34ba861f095))

## [2.1.5](https://github.com/reaandrew/politest/compare/v2.1.4...v2.1.5) (2025-10-20)

### Bug Fixes

* pass separate VERSION/COMMIT/BUILD_DATE to SLSA builder ([f588686](https://github.com/reaandrew/politest/commit/f5886862ded564d25402c66186e1cb5ae03e3cbf))

## [2.1.4](https://github.com/reaandrew/politest/compare/v2.1.3...v2.1.4) (2025-10-20)

### Bug Fixes

* use equals sign for SLSA evaluated-envs format ([57f786f](https://github.com/reaandrew/politest/commit/57f786ff6c63fe3963b7623941469e1fa8c135a1))

## [2.1.3](https://github.com/reaandrew/politest/compare/v2.1.2...v2.1.3) (2025-10-20)

### Bug Fixes

* use single ldflags environment variable for SLSA builders ([c1d8d6d](https://github.com/reaandrew/politest/commit/c1d8d6db6cb693baf445d8fe9d30e4cbfbcc682a))

## [2.1.2](https://github.com/reaandrew/politest/compare/v2.1.1...v2.1.2) (2025-10-20)

### Bug Fixes

* format build date for SLSA evaluated-envs compatibility ([9d1043b](https://github.com/reaandrew/politest/commit/9d1043b9f27e1242b01fad71725aa217dd09bad5))

## [2.1.1](https://github.com/reaandrew/politest/compare/v2.1.0...v2.1.1) (2025-10-20)

### Bug Fixes

* use tag references for SLSA builders with NOSONAR exceptions ([878f550](https://github.com/reaandrew/politest/commit/878f55032926229b5df0f54079d7a4d04a75980e))

## [2.1.0](https://github.com/reaandrew/politest/compare/v2.0.0...v2.1.0) (2025-10-20)

### Features

* add example-tests CI job to run examples in parallel with integration tests ([927d6ca](https://github.com/reaandrew/politest/commit/927d6ca900c842b8ab8c957de5ab3e20295d4fc2))
* add version injection and SLSA provenance generation ([7052f94](https://github.com/reaandrew/politest/commit/7052f94023685813d0c8617b344d4c550079e75c))
* implement SLSA Level 3 provenance with official Go builder ([009d491](https://github.com/reaandrew/politest/commit/009d491c3ddfa8f25b117ac049346848181af71c))

### Bug Fixes

* use full commit SHA hashes for GitHub Actions dependencies ([5613349](https://github.com/reaandrew/politest/commit/56133493cfdf5914bb5a49c3cd710b21eca34580))

### Performance Improvements

* optimize Semgrep CI job for faster execution ([f3acf2e](https://github.com/reaandrew/politest/commit/f3acf2e8755115b145140420c941a2b314316d77))

## [2.0.0](https://github.com/reaandrew/politest/compare/v1.2.0...v2.0.0) (2025-10-20)

### ⚠ BREAKING CHANGES

* Legacy format with scenario-level actions, resources, and expect map is no longer supported. All scenarios must use the tests array format. See migration examples in updated documentation.

Changes:
- Remove Actions, Resources, Expect fields from Scenario struct
- Remove RunLegacyFormat function and related code (~116 lines)
- Convert 7 test scenarios (01-07) to collection format
- Update README.md, CLAUDE.md, and WIKI.md to remove legacy format references
- Update all test files to remove legacy format test cases

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>

### Features

* remove legacy format support and standardize on collection format ([47b2f01](https://github.com/reaandrew/politest/commit/47b2f013a3bf273b27f5ae0f349529624eccee20))

## [1.2.0](https://github.com/reaandrew/politest/compare/v1.1.1...v1.2.0) (2025-10-20)

### Features

* add actions array support and comprehensive examples ([a6d7e2e](https://github.com/reaandrew/politest/commit/a6d7e2e206c35fbc29619605fb9b67466614c286))
* add support for ${VAR}, $VAR, and <VAR> variable formats ([cdc7435](https://github.com/reaandrew/politest/commit/cdc743557efc3b1c3bed0001a424f8363e5f3039))

## [1.1.1](https://github.com/reaandrew/politest/compare/v1.1.0...v1.1.1) (2025-10-19)

### Bug Fixes

* address code review feedback - improve validation and security ([4f08962](https://github.com/reaandrew/politest/commit/4f0896228e2d70e31b5a22802ace361aaae9f6b2))

## [1.1.0](https://github.com/reaandrew/politest/compare/v1.0.0...v1.1.0) (2025-10-18)

### Features

* add comprehensive test examples demonstrating all features ([2d67e9f](https://github.com/reaandrew/politest/commit/2d67e9ff5f6766b98ce58ea63160a3a6c7dc1678))
* add resource policy and cross-account testing support ([e2bdfef](https://github.com/reaandrew/politest/commit/e2bdfef301353d95f92c4bcbaca57fb0f68d9e95))
* add test collection format with named test cases ([8e9ffde](https://github.com/reaandrew/politest/commit/8e9ffde812dc12218debb41d90f831e32a2e5ccd))
* optional test names, context examples, and logo in README ([c2c39e1](https://github.com/reaandrew/politest/commit/c2c39e14b6f8a5d0147c219ae1c78f6b6f2e5cbe))

## 1.0.0 (2025-10-18)

### Features

* add lefthook pre-commit hooks ([bee16fc](https://github.com/reaandrew/politest/commit/bee16fc20188c8122a983776532fd89fcb751453))
* implement IAM policy simulation tool with CI/CD ([5b35d6a](https://github.com/reaandrew/politest/commit/5b35d6ae04c7e5a46bfb845563ffa340aa9f3cd0))

### Bug Fixes

* add go.sum with module dependencies ([902395e](https://github.com/reaandrew/politest/commit/902395e2a1e7d9df3a5c3488e049ae5b4f85deee))
* correct test expectations for SCP/RCP explicit denies ([50c4767](https://github.com/reaandrew/politest/commit/50c47674d37bedecf4ac3f84fbe1ddb15ca94435))
* handle NONE quality gate status in SonarCloud check ([5d40579](https://github.com/reaandrew/politest/commit/5d405799b39d7e4d1e7e5c3fc9e10cbd6a0a72af))
* improve SonarCloud failure handling in CI workflow ([34e5287](https://github.com/reaandrew/politest/commit/34e52878129fd9dac83bda547ff0e294d2a7e031))
* prevent quality gate check from running on sonar scan failure ([5b82ad7](https://github.com/reaandrew/politest/commit/5b82ad73b4a37018a2bba7000171d0e21ce75a51))
* prevent test script from exiting on arithmetic expansion ([cb3bf4b](https://github.com/reaandrew/politest/commit/cb3bf4bbc838c2bf9ef9f4694194cabe6833353c))
* remove unsupported context type enums ([68477fa](https://github.com/reaandrew/politest/commit/68477fa4cc0aca89a0d9cc0aa8699c5f04b20368))
* use strings.EqualFold for case-insensitive comparison ([524446e](https://github.com/reaandrew/politest/commit/524446ef8b1decc5b6f521b8d4cfc68dcaf3d63e))
