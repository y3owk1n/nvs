# Changelog

## [1.13.0](https://github.com/y3owk1n/nvs/compare/v1.12.1...v1.13.0) (2026-02-15)


### Features

* bump golang and deps with tablewrite API changes ([#194](https://github.com/y3owk1n/nvs/issues/194)) ([b444f9a](https://github.com/y3owk1n/nvs/commit/b444f9a8bf887c7481f3164b14ea99b1f76bcb7b))
* **downloader:** add streaming checksum verification ([#207](https://github.com/y3owk1n/nvs/issues/207)) ([1a753a9](https://github.com/y3owk1n/nvs/commit/1a753a9fa632b0e11ee1263605f6b5ce245429e0))
* **filesystem:** add file-based locking for concurrent operations ([#211](https://github.com/y3owk1n/nvs/issues/211)) ([57443b0](https://github.com/y3owk1n/nvs/commit/57443b0065b17c2480cb0fc70d63a8908fef471d))


### Bug Fixes

* add host validation for GitHub mirror URL ([#215](https://github.com/y3owk1n/nvs/issues/215)) ([62878b3](https://github.com/y3owk1n/nvs/commit/62878b388ec0b56af0fd4e829ae7109244b124cd))
* allow deferred functions to execute on interrupt ([#210](https://github.com/y3owk1n/nvs/issues/210)) ([61818e6](https://github.com/y3owk1n/nvs/commit/61818e692143f142da54889afc584ead56902978))
* **archive:** preserve io.Copy errors when close also fails ([#203](https://github.com/y3owk1n/nvs/issues/203)) ([4942022](https://github.com/y3owk1n/nvs/commit/4942022c71e09ca199712e4b0e0cf1453f24fedf))
* **builder:** add context timeout to checkRequiredTools ([#212](https://github.com/y3owk1n/nvs/issues/212)) ([f4e6207](https://github.com/y3owk1n/nvs/commit/f4e62079d74c49a8d150579cbb23efdabc3246cd))
* **builder:** add WaitGroup tracking to prevent goroutine leak ([#213](https://github.com/y3owk1n/nvs/issues/213)) ([5f9ef03](https://github.com/y3owk1n/nvs/commit/5f9ef039ad362e90266eab3e65984f7431899e7f))
* **builder:** use unique build ID to prevent concurrent directory conflicts ([#202](https://github.com/y3owk1n/nvs/issues/202)) ([653692b](https://github.com/y3owk1n/nvs/commit/653692b6144353d4724467e58cc2ced429ec1580))
* **downloader:** improves observability of close errors ([#200](https://github.com/y3owk1n/nvs/issues/200)) ([9de095a](https://github.com/y3owk1n/nvs/commit/9de095a9749bc5a39963b6c54b39fb25bcfc157a))
* propagate context cancellation to build commands ([#204](https://github.com/y3owk1n/nvs/issues/204)) ([64d964c](https://github.com/y3owk1n/nvs/commit/64d964c03ccff5570def3e40fa9e9a2964972ace))
* return error from InitConfig instead of using panic ([#214](https://github.com/y3owk1n/nvs/issues/214)) ([7950708](https://github.com/y3owk1n/nvs/commit/79507086e98fc14866bdeff78faaf33a6a68089a))
* **symlink:** capture stderr from Windows mklink fallback ([#209](https://github.com/y3owk1n/nvs/issues/209)) ([353bc6e](https://github.com/y3owk1n/nvs/commit/353bc6eca69790672a8ae05e1682e40e0c7222e5))
* **use:** improve error handling when activation fails after auto-install ([#198](https://github.com/y3owk1n/nvs/issues/198)) ([0a414b1](https://github.com/y3owk1n/nvs/commit/0a414b18c08643c4b1ed7e441af870c69eeb6117))
* **utils:** handle broken symlinks and add Windows compatibility ([#201](https://github.com/y3owk1n/nvs/issues/201)) ([6c1de2b](https://github.com/y3owk1n/nvs/commit/6c1de2b58fd0435ae8cad169137d919168147cd9))
* **version:** report rollback failures during upgrade ([#199](https://github.com/y3owk1n/nvs/issues/199)) ([889926d](https://github.com/y3owk1n/nvs/commit/889926de98c36340e4215ad91e8a4ba308020f30))

## [1.12.1](https://github.com/y3owk1n/nvs/compare/v1.12.0...v1.12.1) (2025-12-13)


### Bug Fixes

* respect TTL for global cache in ls-remote command ([#189](https://github.com/y3owk1n/nvs/issues/189)) ([b33046a](https://github.com/y3owk1n/nvs/commit/b33046a2da9356234be80a974f73d6e0bc173230))
* use explicit version instead of latest in install scripts ([#191](https://github.com/y3owk1n/nvs/issues/191)) ([1b92419](https://github.com/y3owk1n/nvs/commit/1b924192e3f8b2a10a2bc493daa105458844af3a))

## [1.12.0](https://github.com/y3owk1n/nvs/compare/v1.11.0...v1.12.0) (2025-12-11)


### Features

* add GitHub Action for nvs setup ([#181](https://github.com/y3owk1n/nvs/issues/181)) ([d0f30fe](https://github.com/y3owk1n/nvs/commit/d0f30fee032ab6432fdbdd84544746437fa9b407))
* add global cache for releases ([#183](https://github.com/y3owk1n/nvs/issues/183)) ([acbdfbb](https://github.com/y3owk1n/nvs/commit/acbdfbb6865573998ebb9e0bd63836511bcc348e))


### Bug Fixes

* **cache:** parse global cache JSON with correct field casing ([#187](https://github.com/y3owk1n/nvs/issues/187)) ([e652323](https://github.com/y3owk1n/nvs/commit/e652323e4a248d7cfc0b3628a32ae92d534e4fac))
* ensure update version CI is commitable to main [skip ci] ([#184](https://github.com/y3owk1n/nvs/issues/184)) ([7e3f979](https://github.com/y3owk1n/nvs/commit/7e3f979658764167bf2b604af058e8d4869420be))

## [1.11.0](https://github.com/y3owk1n/nvs/compare/v1.10.7...v1.11.0) (2025-12-06)


### Features

* add --pick flag for interactive version selection ([#177](https://github.com/y3owk1n/nvs/issues/177)) ([e14d63f](https://github.com/y3owk1n/nvs/commit/e14d63fa1a10bd61448b7c2b94ef4d75749cb1a5))
* add GitHub mirror support and run command ([#163](https://github.com/y3owk1n/nvs/issues/163)) ([bd4bff3](https://github.com/y3owk1n/nvs/commit/bd4bff3db7acce4f5172eb23fc02bbe58595eef0))
* add shell integration and doctor command for enhanced user experience ([#166](https://github.com/y3owk1n/nvs/issues/166)) ([72f5115](https://github.com/y3owk1n/nvs/commit/72f511513cb006db6f3505013125f9d08ec91440))
* add Windows PowerShell installer support ([#164](https://github.com/y3owk1n/nvs/issues/164)) ([c22ab5f](https://github.com/y3owk1n/nvs/commit/c22ab5f58fbde69def3a43f864713e1f3aff2ec9))
* **cli:** add --json flag for machine-readable output ([#176](https://github.com/y3owk1n/nvs/issues/176)) ([4725540](https://github.com/y3owk1n/nvs/commit/472554041ffdfec086a27fd22978d3a14ff99b35))
* enhance Neovim build-from-source dependencies and UI progress utilities ([#173](https://github.com/y3owk1n/nvs/issues/173)) ([195c99b](https://github.com/y3owk1n/nvs/commit/195c99bf6a87714cc8527d1e0a42e781984dae57))
* **home-manager:** add comprehensive Home Manager support ([#167](https://github.com/y3owk1n/nvs/issues/167)) ([d9bfd10](https://github.com/y3owk1n/nvs/commit/d9bfd101f739f21d5b8b56a79d167d9385b3c323))
* implement version pinning, nightly rollback, and changelog features ([#165](https://github.com/y3owk1n/nvs/issues/165)) ([f790182](https://github.com/y3owk1n/nvs/commit/f790182b10541cd1a6a76e3a02b9eee1b1d26354))
* make list-remote command use --force flag consistently ([#175](https://github.com/y3owk1n/nvs/issues/175)) ([45d2bf9](https://github.com/y3owk1n/nvs/commit/45d2bf9ee75aa90887aa830648c4d6edaa12b274))
* massive refactoring of the whole repo ([#157](https://github.com/y3owk1n/nvs/issues/157)) ([f63be45](https://github.com/y3owk1n/nvs/commit/f63be45da26b7926a15404a0757677fd3f6cfb7f))
* **test:** enhance testing infrastructure with race detection and CI updates ([#171](https://github.com/y3owk1n/nvs/issues/171)) ([a69b0e2](https://github.com/y3owk1n/nvs/commit/a69b0e2e995c8c2babc2da1f9e2ba1dc75ff58be))
* **ui:** add progress utilities and refactor spinner usage in install/upgrade ([#170](https://github.com/y3owk1n/nvs/issues/170)) ([44909d4](https://github.com/y3owk1n/nvs/commit/44909d4711921682011e798575c0661f3d0d6a90))


### Bug Fixes

* add missing build steps to CI workflow ([#162](https://github.com/y3owk1n/nvs/issues/162)) ([eef47a6](https://github.com/y3owk1n/nvs/commit/eef47a6d2185ee5104cb2156694ee1fc380a256e))
* correct nightly alias case in upgrade command ([#153](https://github.com/y3owk1n/nvs/issues/153)) ([a937c62](https://github.com/y3owk1n/nvs/commit/a937c627b5c764da9f5bd46aa333c8f7b6f99521))
* demote cache read failure log level to debug ([#158](https://github.com/y3owk1n/nvs/issues/158)) ([a30f381](https://github.com/y3owk1n/nvs/commit/a30f381528390bb322ee4b097c27f1c9539dfcdb))
* enhance stable release handling in list-remote command ([#160](https://github.com/y3owk1n/nvs/issues/160)) ([b3a0c42](https://github.com/y3owk1n/nvs/commit/b3a0c42ac84ec4a60f1d2c1a8fe4a9f3a7a28f5a))
* separate unit and integration tests, fix TestRunPath stdin mocking, and add development guide ([#155](https://github.com/y3owk1n/nvs/issues/155)) ([3713b9d](https://github.com/y3owk1n/nvs/commit/3713b9d3fac9665f026f37a5bce6a40d31fc5e65))
* **upgrade:** add initial prefix to progress spinner ([#159](https://github.com/y3owk1n/nvs/issues/159)) ([d5556fe](https://github.com/y3owk1n/nvs/commit/d5556fe8e977568338b8b013a219862ceae629f6))

## [1.10.7](https://github.com/y3owk1n/nvs/compare/v1.10.6...v1.10.7) (2025-08-28)


### Bug Fixes

* **cmd.config:** ensure getting the right standard path based on different OS ([#140](https://github.com/y3owk1n/nvs/issues/140)) ([d5f0c97](https://github.com/y3owk1n/nvs/commit/d5f0c974103daf63b9143da7d2a9f80ba540e0be))
* **cmd.config:** ignore *-data as nvim will create them automatically ([#147](https://github.com/y3owk1n/nvs/issues/147)) ([9824c17](https://github.com/y3owk1n/nvs/commit/9824c17595f2cc21fd0782c08463403d6edba413))
* ensure hardlinking logic works with windows ([#144](https://github.com/y3owk1n/nvs/issues/144)) ([110fafe](https://github.com/y3owk1n/nvs/commit/110fafe9a97b2044894812950f3d1dda22d6ed56))
* ensure to get the right nvim config directories on windows ([#145](https://github.com/y3owk1n/nvs/issues/145)) ([6423127](https://github.com/y3owk1n/nvs/commit/6423127fd77b4d6cafe7b0a74d37fc48e21bc33a))
* exclude `nvim-data` folder for windows ([#146](https://github.com/y3owk1n/nvs/issues/146)) ([f59ee9b](https://github.com/y3owk1n/nvs/commit/f59ee9bc51ef751afdebbb550fb63cc3ef8dfda9))
* respect window default paths for bin ([#143](https://github.com/y3owk1n/nvs/issues/143)) ([b4ec9e7](https://github.com/y3owk1n/nvs/commit/b4ec9e74b53d8ad80dc8fd2047d38401816e192e))
* use junction instead of symlink on windows ([#142](https://github.com/y3owk1n/nvs/issues/142)) ([c464b17](https://github.com/y3owk1n/nvs/commit/c464b17605da2ace2becdeef6f027a9ad7cd82ba))

## [1.10.6](https://github.com/y3owk1n/nvs/compare/v1.10.5...v1.10.6) (2025-08-09)


### Bug Fixes

* **cmd.env:** improve auto ENV sourcing ([#137](https://github.com/y3owk1n/nvs/issues/137)) ([fdec934](https://github.com/y3owk1n/nvs/commit/fdec934eea3ad66b1e27f230b3541b45a790fdc1))

## [1.10.5](https://github.com/y3owk1n/nvs/compare/v1.10.4...v1.10.5) (2025-04-15)


### Bug Fixes

* **installer:** bump timeout to 5 minutes for downloads ([#131](https://github.com/y3owk1n/nvs/issues/131)) ([d0d3893](https://github.com/y3owk1n/nvs/commit/d0d38934527e60d82be94b5a83276db11b6e09bc))

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
