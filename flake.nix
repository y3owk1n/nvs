{
  description = "Easily install, switch, and manage multiple versions (including commit hashes) and config of Neovim like a boss";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/b976292fb39a449bcf410219e4cf0aa05a8b4d04?narHash=sha256-NmiCO/7hKv3TVIXXtEAkpGHiJzQc/5z8PT8tO+SKPZA=";
  };
  outputs =
    { self, nixpkgs, ... }:
    let
      eachSystem = nixpkgs.lib.genAttrs [
        "aarch64-darwin"
        "x86_64-darwin"
        "aarch64-linux"
        "x86_64-linux"
      ];
      # Update this to your latest release version
      latestVersion = "1.10.7";
      # Function to build package with specific version
      makeNvsPackage =
        pkgs: version: usePrebuilt: commitHash:
        pkgs.callPackage ./package.nix {
          inherit version usePrebuilt commitHash;
        };
    in
    {
      overlays.default = final: prev: {
        nvs = makeNvsPackage final latestVersion true null;
        nvs-source = makeNvsPackage final "main" false (self.rev or self.dirtyRev or "unknown");
      };
      # Packages output using the overlay
      packages = eachSystem (
        system:
        let
          pkgs = import nixpkgs {
            inherit system;
            overlays = [ self.overlays.default ];
          };
        in
        {
          # Default: latest version from prebuilt binary
          default = makeNvsPackage pkgs latestVersion true null;
          # Build from source
          source = makeNvsPackage pkgs "main" false (self.rev or self.dirtyRev or "unknown");
        }
      );

      homeManagerModules.default = import ./home-module.nix;
    };
}
