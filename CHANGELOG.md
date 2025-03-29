# Changelog

## [1.10.4](https://github.com/y3owk1n/nvs/compare/v1.10.3...v1.10.4) (2025-03-29)


### Bug Fixes

* **cmd.upgrade:** remove the old version before performing upgrade ([#129](https://github.com/y3owk1n/nvs/issues/129)) ([d9c02be](https://github.com/y3owk1n/nvs/commit/d9c02be0b4fd67457508e19edf9a46390c182c62))
* remove debug for `downloadAndInstall` progress in verbose mode ([#127](https://github.com/y3owk1n/nvs/issues/127)) ([f0466a4](https://github.com/y3owk1n/nvs/commit/f0466a437ca0b44dec0a4b40ba9f6b3fe379a14d))

## [1.10.3](https://github.com/y3owk1n/nvs/compare/v1.10.2...v1.10.3) (2025-03-25)


### Bug Fixes

* add 30 mins timeout & cancellation context for install and build ([#123](https://github.com/y3owk1n/nvs/issues/123)) ([a65f6ee](https://github.com/y3owk1n/nvs/commit/a65f6ee0e4a70f9d2334210c4d30a2d25d9e7a7d))
* **builder:** add 1 retry attemp with clean directory if error happens ([#124](https://github.com/y3owk1n/nvs/issues/124)) ([10134a8](https://github.com/y3owk1n/nvs/commit/10134a8b3a4af29700f4806dc4c02e2b2f79c680))
* **builder:** add debug for temporary path for better inspection ([#120](https://github.com/y3owk1n/nvs/issues/120)) ([b08575d](https://github.com/y3owk1n/nvs/commit/b08575d50689bf3e8479b2e0ef9c2c0cdaff7804))
* **builder:** run builder process async ([#122](https://github.com/y3owk1n/nvs/issues/122)) ([57ab7a6](https://github.com/y3owk1n/nvs/commit/57ab7a6e93e90a2a34366c621e3fcece8f92c485))

## [1.10.2](https://github.com/y3owk1n/nvs/compare/v1.10.1...v1.10.2) (2025-03-20)


### Bug Fixes

* **builder:** remove build directory before building from source ([#117](https://github.com/y3owk1n/nvs/issues/117)) ([7e59e1f](https://github.com/y3owk1n/nvs/commit/7e59e1f2e5484f882d3a0e57aca418cbf03c4f10))

## [1.10.1](https://github.com/y3owk1n/nvs/compare/v1.10.0...v1.10.1) (2025-03-14)


### Bug Fixes

* **builder:** make sure to actually install neovim after build instead of just copying the files ([#115](https://github.com/y3owk1n/nvs/issues/115)) ([2a1fbd5](https://github.com/y3owk1n/nvs/commit/2a1fbd5275e3c8e9cc2b16b8207da3c17795a8cd))

## [1.10.0](https://github.com/y3owk1n/nvs/compare/v1.9.1...v1.10.0) (2025-03-14)


### Features

* support commit hash builds for installation ([#112](https://github.com/y3owk1n/nvs/issues/112)) ([1ef7ddf](https://github.com/y3owk1n/nvs/commit/1ef7ddfcb68267c2c60353c80539f1f8fbb3ecc6))


### Bug Fixes

* add prompt icon for confirmation prompt ([#109](https://github.com/y3owk1n/nvs/issues/109)) ([cc63ed9](https://github.com/y3owk1n/nvs/commit/cc63ed94aa1598d2553ec6044df00ef1d7c1bd97))
* **builder:** make `exec.command` and `make` quiet ([#113](https://github.com/y3owk1n/nvs/issues/113)) ([c046918](https://github.com/y3owk1n/nvs/commit/c046918e58ca74a1f4b636710bb6891cb7dbf56d))

## [1.9.1](https://github.com/y3owk1n/nvs/compare/v1.9.0...v1.9.1) (2025-03-12)


### Bug Fixes

* add more debug logs for verbose mode for `nvs env` and `nvs list` ([#107](https://github.com/y3owk1n/nvs/issues/107)) ([47a8175](https://github.com/y3owk1n/nvs/commit/47a8175ce64d75d545362bdde9b5ec2725a7472b))

## [1.9.0](https://github.com/y3owk1n/nvs/compare/v1.8.2...v1.9.0) (2025-03-12)


### Features

* add `nvs env` to show the current configurations ([#105](https://github.com/y3owk1n/nvs/issues/105)) ([d2b09c8](https://github.com/y3owk1n/nvs/commit/d2b09c8ef09501d662124afa28f87a413a742709))
* add `nvs list` and move previous `list` -&gt; `list-remote` ([#103](https://github.com/y3owk1n/nvs/issues/103)) ([d1f600d](https://github.com/y3owk1n/nvs/commit/d1f600dca8d9fd9c9811ac404f4cec9f7200d189))

## [1.8.2](https://github.com/y3owk1n/nvs/compare/v1.8.1...v1.8.2) (2025-03-12)


### Bug Fixes

* make sure getAssetUrl to consider `linux64.tar.gz` pattern ([#94](https://github.com/y3owk1n/nvs/issues/94)) ([49f8ab0](https://github.com/y3owk1n/nvs/commit/49f8ab076cc34dcc20650bceceac38cacb44a8b9))
* **uninstallCmd:** check if uninstalling the current using version and prompt confirmation ([#99](https://github.com/y3owk1n/nvs/issues/99)) ([45b7977](https://github.com/y3owk1n/nvs/commit/45b7977ecd471f73cf7957cbd38e7bc21e815142))
* **uninstallCmd:** prompt remaining version to use after uninstalling current using version ([#100](https://github.com/y3owk1n/nvs/issues/100)) ([42196c5](https://github.com/y3owk1n/nvs/commit/42196c5c13e3356ff38e2aa9dd57e5ebe44f2d7d))
* **useCmd:** try to install the version if it's not installed when trying to switch to it ([#98](https://github.com/y3owk1n/nvs/issues/98)) ([4e03707](https://github.com/y3owk1n/nvs/commit/4e037071f8de0e356ce0034ae98acefa10524bfb))

## [1.8.1](https://github.com/y3owk1n/nvs/compare/v1.8.0...v1.8.1) (2025-03-11)


### Bug Fixes

* **cmd.reset:** do not remove all the content in bin directory but just the symlinked `nvim` ([#90](https://github.com/y3owk1n/nvs/issues/90)) ([bfa5d1e](https://github.com/y3owk1n/nvs/commit/bfa5d1e048a06a40cf16425a8678de110a8d919f))

## [1.8.0](https://github.com/y3owk1n/nvs/compare/v1.7.3...v1.8.0) (2025-03-11)


### Features

* use os-specific directories for storing the files with env var overrides ([#88](https://github.com/y3owk1n/nvs/issues/88)) ([bece743](https://github.com/y3owk1n/nvs/commit/bece743d98ec3cb93b1f89d2fe97a48f09d915c9))

## [1.7.3](https://github.com/y3owk1n/nvs/compare/v1.7.2...v1.7.3) (2025-03-10)


### Bug Fixes

* make sure checksum is not uploaded twice ([#84](https://github.com/y3owk1n/nvs/issues/84)) ([b21fbc7](https://github.com/y3owk1n/nvs/commit/b21fbc7d68fd2ff2499c981e1c614ca42c39e166))

## [1.7.2](https://github.com/y3owk1n/nvs/compare/v1.7.1...v1.7.2) (2025-03-10)


### Bug Fixes

* add checksum for releases ([#82](https://github.com/y3owk1n/nvs/issues/82)) ([2cc9d6d](https://github.com/y3owk1n/nvs/commit/2cc9d6d7d776a802379e7f1e3f0d5ec2ec45cf33))

## [1.7.1](https://github.com/y3owk1n/nvs/compare/v1.7.0...v1.7.1) (2025-03-09)


### Bug Fixes

* add cyan colors to variables ([#78](https://github.com/y3owk1n/nvs/issues/78)) ([42937ef](https://github.com/y3owk1n/nvs/commit/42937efaeca129001b454948cdc6af089a54fbab))
* **cmd.list:** make table compact ([#74](https://github.com/y3owk1n/nvs/issues/74)) ([7f24383](https://github.com/y3owk1n/nvs/commit/7f24383341b18214faf2839c498649690f196d75))
* **cmd.root:** move ctrl+c interuption to debug level ([#76](https://github.com/y3owk1n/nvs/issues/76)) ([e731d64](https://github.com/y3owk1n/nvs/commit/e731d64343dd6968da7186dc0875f8eae50a7a74))
* **utils.LaunchNvimWithConfig:** update error printing with icon ([#77](https://github.com/y3owk1n/nvs/issues/77)) ([81f3428](https://github.com/y3owk1n/nvs/commit/81f3428956c60f07d677c19a19add587a551aaf6))

## [1.7.0](https://github.com/y3owk1n/nvs/compare/v1.6.0...v1.7.0) (2025-03-09)


### Features

* add automatic path setup command ([#68](https://github.com/y3owk1n/nvs/issues/68)) ([a004820](https://github.com/y3owk1n/nvs/commit/a004820148d140cbdfc9a0213b5129672bcd921e))
* add more detailed debug statement for better --verbose visualisation ([#73](https://github.com/y3owk1n/nvs/issues/73)) ([8768ea1](https://github.com/y3owk1n/nvs/commit/8768ea188aa1739fe0145bda2a08d99273e52afc))


### Bug Fixes

* ensure new line if ctrl-c pressed for cancelling operations ([#71](https://github.com/y3owk1n/nvs/issues/71)) ([f3b9275](https://github.com/y3owk1n/nvs/commit/f3b9275d0f8f991583b80b8956a107a6ecd57d19))
* updates text coloring to ensure consistency ([#72](https://github.com/y3owk1n/nvs/issues/72)) ([53a1b89](https://github.com/y3owk1n/nvs/commit/53a1b89f72a6a7eb20d88e2f5959342a647ae8e4))

## [1.6.0](https://github.com/y3owk1n/nvs/compare/v1.5.2...v1.6.0) (2025-03-08)


### Features

* add upgrade indicator for listCmd table ([#63](https://github.com/y3owk1n/nvs/issues/63)) ([8200e7d](https://github.com/y3owk1n/nvs/commit/8200e7d4d50a977fcb57d67ad7835d1a6039bebf))
* improve currentCmd UI ([#65](https://github.com/y3owk1n/nvs/issues/65)) ([ba807fe](https://github.com/y3owk1n/nvs/commit/ba807fea7ccaea0cf0a570991415ddca6e2b822b))
* standardize UI across all commands ([#66](https://github.com/y3owk1n/nvs/issues/66)) ([4953905](https://github.com/y3owk1n/nvs/commit/4953905601e37974bc9f7b854fcac84d83d73063))

## [1.5.2](https://github.com/y3owk1n/nvs/compare/v1.5.1...v1.5.2) (2025-03-08)


### Bug Fixes

* do not run verifying checksum if no checksumURL passed ([#59](https://github.com/y3owk1n/nvs/issues/59)) ([4723269](https://github.com/y3owk1n/nvs/commit/4723269f833f5267e3b40098c595e6fcd7437215))
* return err instead of log if failed to write to cache ([#61](https://github.com/y3owk1n/nvs/issues/61)) ([a15ae75](https://github.com/y3owk1n/nvs/commit/a15ae75aa833b7909e1b0264970da66b8bcef6d9))

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
