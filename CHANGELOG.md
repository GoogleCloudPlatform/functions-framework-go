# Changelog

### [1.5.2](https://www.github.com/GoogleCloudPlatform/functions-framework-go/compare/v1.5.1...v1.5.2) (2021-11-24)


### Bug Fixes

* make metadata.FromContext work again ([#103](https://www.github.com/GoogleCloudPlatform/functions-framework-go/issues/103)) ([2714714](https://www.github.com/GoogleCloudPlatform/functions-framework-go/commit/2714714d9ff985a6b6ed9822c5bc53f9ec8a18f7))

### [1.5.1](https://www.github.com/GoogleCloudPlatform/functions-framework-go/compare/v1.5.0...v1.5.1) (2021-11-17)


### Bug Fixes

* minimize dependencies on 3P libraries ([#101](https://www.github.com/GoogleCloudPlatform/functions-framework-go/issues/101)) ([f5c1abd](https://www.github.com/GoogleCloudPlatform/functions-framework-go/commit/f5c1abdf826826d769ae8661ae8d65cfc48ff288))

## [1.5.0](https://www.github.com/GoogleCloudPlatform/functions-framework-go/compare/v1.4.0...v1.5.0) (2021-11-10)


### Features

* move declarative function API into `functions` package ([#99](https://www.github.com/GoogleCloudPlatform/functions-framework-go/issues/99)) ([8f488f2](https://www.github.com/GoogleCloudPlatform/functions-framework-go/commit/8f488f29af1f7631a3a840c9b61ab6da0773a848))


### Bug Fixes

* let CloudEvent functions be registered multiple times ([#95](https://www.github.com/GoogleCloudPlatform/functions-framework-go/issues/95)) ([0e41555](https://www.github.com/GoogleCloudPlatform/functions-framework-go/commit/0e41555882aec93a322fb87c7a763fe98e78545a))

## [1.4.0](https://www.github.com/GoogleCloudPlatform/functions-framework-go/compare/v1.3.0...v1.4.0) (2021-11-02)


### Features

* init declarative functions go ([#92](https://www.github.com/GoogleCloudPlatform/functions-framework-go/issues/92)) ([ae1bf32](https://www.github.com/GoogleCloudPlatform/functions-framework-go/commit/ae1bf320be8ff6eef0863a5c5961ff9413d011a8))


### Bug Fixes

* use standard RFC3339 time formatting ([#89](https://www.github.com/GoogleCloudPlatform/functions-framework-go/issues/89)) ([8218243](https://www.github.com/GoogleCloudPlatform/functions-framework-go/commit/82182437506b131034137b7d6cbb24e522bd213e))

## [1.3.0](https://www.github.com/GoogleCloudPlatform/functions-framework-go/compare/v1.2.1...v1.3.0) (2021-10-19)


### Features

* enable converting CloudEvent requests to Background Event requests ([#86](https://www.github.com/GoogleCloudPlatform/functions-framework-go/issues/86)) ([c2a9921](https://www.github.com/GoogleCloudPlatform/functions-framework-go/commit/c2a992124fcdf5cefd5a39a4c20d2989c574843e))


### Bug Fixes

* make event marshaling HTTP error codes consistent ([#85](https://www.github.com/GoogleCloudPlatform/functions-framework-go/issues/85)) ([b475137](https://www.github.com/GoogleCloudPlatform/functions-framework-go/commit/b475137216a6870aeeaae8665994064af36dc0f8))
* update 'upcasting' pubsub and firebase event conversion ([#84](https://www.github.com/GoogleCloudPlatform/functions-framework-go/issues/84)) ([1e4b705](https://www.github.com/GoogleCloudPlatform/functions-framework-go/commit/1e4b705eb3fa36bb36e074626a4538c041e05d31))
* use latest conformance test GitHub Action to fix tests ([#82](https://www.github.com/GoogleCloudPlatform/functions-framework-go/issues/82)) ([f5f92b9](https://www.github.com/GoogleCloudPlatform/functions-framework-go/commit/f5f92b9fd789ac57a46634a05ae4c310fabc06f1))

### [1.2.1](https://www.github.com/GoogleCloudPlatform/functions-framework-go/compare/v1.2.0...v1.2.1) (2021-09-07)


### Bug Fixes

* update Firebase Auth subject in CloudEvent conversion ([#68](https://www.github.com/GoogleCloudPlatform/functions-framework-go/issues/68)) ([c36839b](https://www.github.com/GoogleCloudPlatform/functions-framework-go/commit/c36839bd73f90030a351a90404e4ea465cd8c7d7))
