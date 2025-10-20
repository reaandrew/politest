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
