# Changelog

## [1.5.1](https://github.com/y3owk1n/nvs/compare/v1.5.0...v1.5.1) (2025-03-07)


### Bug Fixes

* add better description for list ([#55](https://github.com/y3owk1n/nvs/issues/55)) ([5674c83](https://github.com/y3owk1n/nvs/commit/5674c8324773dd7f3ff6c82861cbb7a5cc0cfdd2))

## [1.5.0](https://github.com/y3owk1n/nvs/compare/v1.4.0...v1.5.0) (2025-03-07)


### Features

* add current cmd to show current using version ([#46](https://github.com/y3owk1n/nvs/issues/46)) ([a9333a3](https://github.com/y3owk1n/nvs/commit/a9333a3d4338ce14df3d7357fbd349ae039a51a2))
* merge list-installed and list-remote to just list ([#53](https://github.com/y3owk1n/nvs/issues/53)) ([68da105](https://github.com/y3owk1n/nvs/commit/68da105b31712b44d483f55858d0845d81679aee))


### Bug Fixes

* **listRemoteCmd:** remove "type" column ([#50](https://github.com/y3owk1n/nvs/issues/50)) ([6ff5cc1](https://github.com/y3owk1n/nvs/commit/6ff5cc17515e6cb37e32b17a83ce99ee91e932c7))
* **releases.GetAssetURL:** add more pattern for matching on macos and linux older versions ([#52](https://github.com/y3owk1n/nvs/issues/52)) ([71805e5](https://github.com/y3owk1n/nvs/commit/71805e5fb6283dbea9d296e6e69435a03000b607))
* **releases.GetAssetUrl:** change to exact patterns for macos and linux ([#47](https://github.com/y3owk1n/nvs/issues/47)) ([8c9610e](https://github.com/y3owk1n/nvs/commit/8c9610e06cbac246a3fed2992addce9af4dc8cf4))
* **releases:** filter releases to v0.5.0 and above ([#44](https://github.com/y3owk1n/nvs/issues/44)) ([1918d9e](https://github.com/y3owk1n/nvs/commit/1918d9e71bf63f54dd19a930ec08660da07478ee))
* **utils.FindNvimBinary:** windows needs to consider .exe also as their binary ([#49](https://github.com/y3owk1n/nvs/issues/49)) ([54f9450](https://github.com/y3owk1n/nvs/commit/54f9450cff363b8e2fb600b853877a37a129e2b2))

## [1.4.0](https://github.com/y3owk1n/nvs/compare/v1.3.1...v1.4.0) (2025-03-07)


### Features

* add spinner and better UI for install and upgrade ([#41](https://github.com/y3owk1n/nvs/issues/41)) ([5e5934d](https://github.com/y3owk1n/nvs/commit/5e5934d5af943445db6383fb808f3c18f9327b70))
* keep .nvs/bin structure during reset to avoid needs to relink path ([#36](https://github.com/y3owk1n/nvs/issues/36)) ([5489e7d](https://github.com/y3owk1n/nvs/commit/5489e7d9fa0b35c7bb12b69c243bc2bcf1697f5a))
* make list nicer with table ([#35](https://github.com/y3owk1n/nvs/issues/35)) ([e7f7ad4](https://github.com/y3owk1n/nvs/commit/e7f7ad4c66b52c9011649574c8cdac5a04f780b2))
* normalise version to support 0.0.0 or v0.0.0 ([#40](https://github.com/y3owk1n/nvs/issues/40)) ([ecb0c68](https://github.com/y3owk1n/nvs/commit/ecb0c68d6240d69eaed841460f41e659a3b8b073))
* update table ui with more details ([#43](https://github.com/y3owk1n/nvs/issues/43)) ([6848e52](https://github.com/y3owk1n/nvs/commit/6848e529598e97b1aad32d822c8a9050c18a36d6))
* use tablewriter for table UI ([#39](https://github.com/y3owk1n/nvs/issues/39)) ([7666dfa](https://github.com/y3owk1n/nvs/commit/7666dfa125b8f241f1e753a3e60cc3db83fdc159))


### Bug Fixes

* abort switching if trying to switch to current version ([#37](https://github.com/y3owk1n/nvs/issues/37)) ([cc33f2b](https://github.com/y3owk1n/nvs/commit/cc33f2b6f9503dc7a3586dc06ae4de1f36f5b1ca))


### Performance Improvements

* optimise build for smaller binary ([#33](https://github.com/y3owk1n/nvs/issues/33)) ([058b34b](https://github.com/y3owk1n/nvs/commit/058b34b96f13f2fc3333618af9f3a79e9a6822d8))

## [1.3.1](https://github.com/y3owk1n/nvs/compare/v1.3.0...v1.3.1) (2025-03-06)


### Bug Fixes

* **releases.GetCachedReleases:** lower down log level for cache release notification ([#31](https://github.com/y3owk1n/nvs/issues/31)) ([7c3086f](https://github.com/y3owk1n/nvs/commit/7c3086f2a435517bd3a9ca3a8002b1593d234435))

## [1.3.0](https://github.com/y3owk1n/nvs/compare/v1.2.1...v1.3.0) (2025-03-06)


### Features

* add config switcher/opener ([#26](https://github.com/y3owk1n/nvs/issues/26)) ([ef22f3d](https://github.com/y3owk1n/nvs/commit/ef22f3db733de3d9cdc3157ff2dd207b0a6a0d76))

## [1.2.1](https://github.com/y3owk1n/nvs/compare/v1.2.0...v1.2.1) (2025-03-06)


### Bug Fixes

* use cached releases instead to avoid potential api rate limit issue ([#22](https://github.com/y3owk1n/nvs/issues/22)) ([bbf9375](https://github.com/y3owk1n/nvs/commit/bbf9375f0579ed29c99cbe026fcd0c8386da877f))

## [1.2.0](https://github.com/y3owk1n/nvs/compare/v1.1.0...v1.2.0) (2025-03-06)


### Features

* use commit hash as nightly identifier ([#20](https://github.com/y3owk1n/nvs/issues/20)) ([7529a21](https://github.com/y3owk1n/nvs/commit/7529a2116754624f2fd50f72331b0fae990d89d4))

## [1.1.0](https://github.com/y3owk1n/nvs/compare/v1.0.2...v1.1.0) (2025-03-06)


### Features

* add some shorthands for common commands ([#19](https://github.com/y3owk1n/nvs/issues/19)) ([0a06aa1](https://github.com/y3owk1n/nvs/commit/0a06aa1312beba978a44fdccc5a63cf27b2b1ea1))
* add upgrade command for stable and nightly ([#17](https://github.com/y3owk1n/nvs/issues/17)) ([95823c5](https://github.com/y3owk1n/nvs/commit/95823c5c78fed7bcd19ca13db3df45ce1fd62b22))
* **listCmd:** add indicator for the one that being used in the list ([#13](https://github.com/y3owk1n/nvs/issues/13)) ([c3f6e6b](https://github.com/y3owk1n/nvs/commit/c3f6e6bc5fe230c79fc5a6e57b5535ae03dbdd1a))
* **listRemoteCmd:** only show annotations for stable and nightly ([#15](https://github.com/y3owk1n/nvs/issues/15)) ([d71401b](https://github.com/y3owk1n/nvs/commit/d71401b40d478483da88feeda8d43c7984fdb96d))
* remain stable namespaces when installing and using stable ([#16](https://github.com/y3owk1n/nvs/issues/16)) ([7d54bfe](https://github.com/y3owk1n/nvs/commit/7d54bfe095a41c8ba54c0275514fe6f89764b9e4))
* update list command to list-installed for more clarity ([#18](https://github.com/y3owk1n/nvs/issues/18)) ([0a1625d](https://github.com/y3owk1n/nvs/commit/0a1625df1688cf98ceaef9e88acdaad0bcf19dca))

## [1.0.2](https://github.com/y3owk1n/nvs/compare/v1.0.1...v1.0.2) (2025-03-06)


### Bug Fixes

* rename nvsw -&gt; nvs ([#9](https://github.com/y3owk1n/nvs/issues/9)) ([e894271](https://github.com/y3owk1n/nvs/commit/e894271958304947a1ad2e8749766ae38cd5e539))

## [1.0.1](https://github.com/y3owk1n/nvsw/compare/v1.0.0...v1.0.1) (2025-03-06)


### Bug Fixes

* **build:** target main to capture version changes ([#5](https://github.com/y3owk1n/nvsw/issues/5)) ([2896bab](https://github.com/y3owk1n/nvsw/commit/2896babb856365c32c5c37fa634b97efe7c43d53))

## 1.0.0 (2025-03-06)


### Features

* initial nsvm implementation ([#1](https://github.com/y3owk1n/nvsw/issues/1)) ([a2be422](https://github.com/y3owk1n/nvsw/commit/a2be4228ff070b1042b6328b522a9f317a0213c6))


### Bug Fixes

* nvms -&gt; nvsw ([#3](https://github.com/y3owk1n/nvsw/issues/3)) ([4bbfa14](https://github.com/y3owk1n/nvsw/commit/4bbfa1487bd8d4fcdb43d2be410fa7dc1fa57e5e))
