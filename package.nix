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
          # run `nix hash convert --hash-algo sha256 (nix-prefetch-url https://github.com/y3owk1n/nvs/releases/download/v1.10.7/nvs-darwin-arm64)`
          sha256 = "sha256-qGv0DzXDVu/UAhHEFrKLNzVRxB4hucdYJkOQOefoJ50=";
        };
        "x86_64-darwin" = {
          url = "https://github.com/y3owk1n/nvs/releases/download/v${version}/nvs-darwin-amd64";
          # run `nix hash convert --hash-algo sha256 (nix-prefetch-url https://github.com/y3owk1n/nvs/releases/download/v1.10.7/nvs-darwin-amd64)`
          sha256 = "sha256-jGEllbTI69Uvqw2fix9381RnXs7xlUVTi8o6+cAbHXg=";
        };
        "aarch64-linux" = {
          url = "https://github.com/y3owk1n/nvs/releases/download/v${version}/nvs-linux-arm64";
          # run `nix hash convert --hash-algo sha256 (nix-prefetch-url https://github.com/y3owk1n/nvs/releases/download/v1.10.7/nvs-linux-arm64)`
          sha256 = "sha256-z9GBFC/h2EyimEDeJ1TdaL48R/myppp2wDus2vTpSt0=";
        };
        "x86_64-linux" = {
          url = "https://github.com/y3owk1n/nvs/releases/download/v${version}/nvs-linux-amd64";
          # run `nix hash convert --hash-algo sha256 (nix-prefetch-url https://github.com/y3owk1n/nvs/releases/download/v1.10.7/nvs-linux-amd64)`
          sha256 = "sha256-oxR8eVtCr8RGyDy8w63xxrpsvEQnk78ll3N0I7eAgsA=";
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
    vendorHash = "sha256-l2FdnXA+vKVRekcIKt1R+MxppraTsmo0b/B7RNqnxjA=";

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
