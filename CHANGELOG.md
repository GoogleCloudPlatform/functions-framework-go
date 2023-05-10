# Changelog

## [1.7.3](https://github.com/GoogleCloudPlatform/functions-framework-go/compare/v1.7.2...v1.7.3) (2023-05-10)


### Bug Fixes

* attempt to log panic error in one logging entry ([#197](https://github.com/GoogleCloudPlatform/functions-framework-go/issues/197)) ([df1a83d](https://github.com/GoogleCloudPlatform/functions-framework-go/commit/df1a83d30d117ccb2706399873a7aa6e1bc2eb38))

## [1.7.2](https://github.com/GoogleCloudPlatform/functions-framework-go/compare/v1.7.1...v1.7.2) (2023-05-08)


### Bug Fixes

* wrap panic message when log to stderr ([#195](https://github.com/GoogleCloudPlatform/functions-framework-go/issues/195)) ([a1541ce](https://github.com/GoogleCloudPlatform/functions-framework-go/commit/a1541ce7b2b9d2e7ec93833fee4c88a384cca89a))

## [1.7.1](https://github.com/GoogleCloudPlatform/functions-framework-go/compare/v1.7.0...v1.7.1) (2023-04-24)


### Bug Fixes

* add new line to panic stack trace so Error Reporting can ingest the log ([#190](https://github.com/GoogleCloudPlatform/functions-framework-go/issues/190)) ([76466dd](https://github.com/GoogleCloudPlatform/functions-framework-go/commit/76466dd6f852c36c564de88bdf46b1fd6a8c04cd))

## [1.7.0](https://github.com/GoogleCloudPlatform/functions-framework-go/compare/v1.6.1...v1.7.0) (2023-04-18)


### Features

* Add support for strongly typed function signature ([#168](https://github.com/GoogleCloudPlatform/functions-framework-go/issues/168)) ([06264b6](https://github.com/GoogleCloudPlatform/functions-framework-go/commit/06264b6785e5aef394d97e516d5c1819d3e09d91))
* Allow registering multiple functions with one server for local testing ([#154](https://github.com/GoogleCloudPlatform/functions-framework-go/issues/154)) ([fcc3f61](https://github.com/GoogleCloudPlatform/functions-framework-go/commit/fcc3f6159d0d8e29bfeb715b6d1319fedcfb0510))
* configure security score card action ([#169](https://github.com/GoogleCloudPlatform/functions-framework-go/issues/169)) ([e038fee](https://github.com/GoogleCloudPlatform/functions-framework-go/commit/e038fee735ad43d26c86cc5fc5887b42dc52b467))

## [1.6.0](https://github.com/GoogleCloudPlatform/functions-framework-go/compare/v1.5.3...v1.6.0) (2022-08-05)


### Features

* Add release candidate validation ([#124](https://github.com/GoogleCloudPlatform/functions-framework-go/issues/124)) ([4f5e934](https://github.com/GoogleCloudPlatform/functions-framework-go/commit/4f5e9341b8a7ac43d7f18ad499ad326ff585ff06))


### Bug Fixes

* adding a check for null http handler before starting the server ([#138](https://github.com/GoogleCloudPlatform/functions-framework-go/issues/138)) ([5d5bf7a](https://github.com/GoogleCloudPlatform/functions-framework-go/commit/5d5bf7a741528b4a82cbe9c67f48425fe19be444))
* Allow registering multiple functions with one server for local testing. ([#143](https://github.com/GoogleCloudPlatform/functions-framework-go/issues/143)) ([3cab285](https://github.com/GoogleCloudPlatform/functions-framework-go/commit/3cab285f11b6cafced19dd42756dca821a89dda7))
* log CloudEvent function errors to stderr ([#132](https://github.com/GoogleCloudPlatform/functions-framework-go/issues/132)) ([ac973b4](https://github.com/GoogleCloudPlatform/functions-framework-go/commit/ac973b4343f4814abe811d65c0c08e4c0aa4c59e))
* remove obsolete blank prints ([#144](https://github.com/GoogleCloudPlatform/functions-framework-go/issues/144)) ([5c7091f](https://github.com/GoogleCloudPlatform/functions-framework-go/commit/5c7091ff59ebcfd724cdd3c90f4b97c318696040))

### [1.5.3](https://github.com/GoogleCloudPlatform/functions-framework-go/compare/v1.5.2...v1.5.3) (2022-02-10)


### Bug Fixes

* return generic error message when function panics ([#111](https://github.com/GoogleCloudPlatform/functions-framework-go/issues/111)) ([e25c08a](https://github.com/GoogleCloudPlatform/functions-framework-go/commit/e25c08a01bc0b424edcf5e010aa4099c0797020e))

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
