{
  fetchurl,
  gitUpdater,
  installShellFiles,
  stdenv,
  versionCheckHook,
  lib,
  buildGoModule,
  version ? "main",
  usePrebuilt ? false,
  commitHash ? null,
  writableTmpDirAsHomeHook,
  nix-update-script,
}:
if usePrebuilt then
  let
    # Determine architecture-specific details
    archInfo =
      {
        "aarch64-darwin" = {
          url = "https://github.com/y3owk1n/nvs/releases/download/v${version}/nvs-darwin-arm64";
          # run `nix hash convert --hash-algo sha256 (nix-prefetch-url https://github.com/y3owk1n/nvs/releases/download/v1.12.1/nvs-darwin-arm64)`
          sha256 = "sha256-UZHFKNPf+DC3k+D9VdRyD+JpjSVEXjuqfSIjjjTwj9k=";
        };
        "x86_64-darwin" = {
          url = "https://github.com/y3owk1n/nvs/releases/download/v${version}/nvs-darwin-amd64";
          # run `nix hash convert --hash-algo sha256 (nix-prefetch-url https://github.com/y3owk1n/nvs/releases/download/v1.12.1/nvs-darwin-amd64)`
          sha256 = "sha256-lPf1VeMSe+WCRVuJg9/SeRsveVmCAc3kcp0CeiBqEsQ=";
        };
        "aarch64-linux" = {
          url = "https://github.com/y3owk1n/nvs/releases/download/v${version}/nvs-linux-arm64";
          # run `nix hash convert --hash-algo sha256 (nix-prefetch-url https://github.com/y3owk1n/nvs/releases/download/v1.12.1/nvs-linux-arm64)`
          sha256 = "sha256-Y0JMCVelBeQLrAMh/SN+fG1emq//ncoVo7rojTkFx48=";
        };
        "x86_64-linux" = {
          url = "https://github.com/y3owk1n/nvs/releases/download/v${version}/nvs-linux-amd64";
          # run `nix hash convert --hash-algo sha256 (nix-prefetch-url https://github.com/y3owk1n/nvs/releases/download/v1.12.1/nvs-linux-amd64)`
          sha256 = "sha256-eADoWxdmz0ENAItPH4NgZdJJnIj93CYV3Ijnngh1yHg=";
        };
      }
      .${stdenv.hostPlatform.system} or (throw "Unsupported system: ${stdenv.hostPlatform.system}");

  in
  stdenv.mkDerivation {
    pname = "nvs";

    inherit version;

    src = fetchurl {
      url = archInfo.url;
      sha256 = archInfo.sha256;
    };

    nativeBuildInputs = [
      installShellFiles
      writableTmpDirAsHomeHook
    ];

    dontUnpack = true;
    dontBuild = true;

    installPhase = ''
      runHook preInstall
      mkdir -p $out/bin
      install -m755 $src $out/bin/nvs
      runHook postInstall
    '';

    postInstall = ''
      if [[ "${lib.boolToString (stdenv.buildPlatform.canExecute stdenv.hostPlatform)}" == "true" ]]; then
        installShellCompletion --cmd nvs \
          --bash <($out/bin/nvs completion bash) \
          --fish <($out/bin/nvs completion fish) \
          --zsh <($out/bin/nvs completion zsh)
      fi
    '';

    doInstallCheck = true;
    nativeInstallCheckInputs = [
      versionCheckHook
    ];

    passthru.updateScript = gitUpdater {
      url = "https://github.com/y3owk1n/nvs.git";
      rev-prefix = "v";
    };

    meta = with lib; {
      description = "Easily install, switch, and manage multiple versions (including commit hashes) and config of Neovim like a boss";
      homepage = "https://github.com/y3owk1n/nvs";
      license = licenses.mit;
      mainProgram = "nvs";
    };
  }
else
  let
    shortHash = if commitHash != null then lib.substring 0 7 commitHash else null;

    pversion = "${version}${if shortHash != null then "-${shortHash}" else ""}";
  in
  # Build from source
  buildGoModule (finalAttrs: {
    pname = "nvs";
    version = pversion;

    src = lib.cleanSource ./.;

    # run the following command to get the sha256 hash
    # `nix-shell -p go --run 'go mod vendor'`
    # `nix hash path vendor`
    # `rm -rf vendor`
    vendorHash = "sha256-VUoWtM74fWVwdnWYa4tdKGWkkf+Hr/M8Wvkq/sRULhs=";

    ldflags = [
      "-s"
      "-w"
      "-X github.com/y3owk1n/nvs/cmd.Version=${finalAttrs.version}"
    ];

    # Completions
    nativeBuildInputs = [
      installShellFiles
      writableTmpDirAsHomeHook
    ];

    # Allow Go to use any available toolchain
    preBuild = ''
      export GOTOOLCHAIN=auto
    '';

    postInstall = ''
      # install shell completions
      if [[ "${lib.boolToString (stdenv.buildPlatform.canExecute stdenv.hostPlatform)}" == "true" ]]; then
        installShellCompletion --cmd nvs \
          --bash <($out/bin/nvs completion bash) \
          --fish <($out/bin/nvs completion fish) \
          --zsh <($out/bin/nvs completion zsh)
      fi
    '';

    passthru = {
      updateScript = nix-update-script { };
    };

    meta = with lib; {
      description = "Easily install, switch, and manage multiple versions (including commit hashes) and config of Neovim like a boss";
      homepage = "https://github.com/y3owk1n/nvs";
      license = licenses.mit;
      mainProgram = "nvs";
    };
  })
