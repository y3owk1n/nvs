{
  config,
  lib,
  pkgs,
  ...
}:
let
  cfg = config.programs.nvs;
in
{
  options = {
    programs.nvs = {
      enable = lib.mkEnableOption "nvs (Neovim Version Switcher)";

      package = lib.mkPackageOption pkgs "nvs" { };

      configDir = lib.mkOption {
        type = lib.types.str;
        default = "${config.xdg.configHome}/nvs";
        description = "Directory for nvs configuration and versions";
      };

      cacheDir = lib.mkOption {
        type = lib.types.str;
        default = "${config.xdg.cacheHome}/nvs";
        description = "Directory for nvs cache files";
      };

      binDir = lib.mkOption {
        type = lib.types.str;
        default = "${config.home.homeDirectory}/.local/bin";
        description = "Directory for nvs binary symlinks";
      };

      enableShellIntegration = lib.mkOption {
        type = lib.types.bool;
        default = true;
        description = "Enable shell integration for automatic environment setup";
      };

      enableAutoSwitch = lib.mkOption {
        type = lib.types.bool;
        default = true;
        description = "Enable automatic version switching when entering directories with .nvs-version files";
      };

      shellIntegration = {
        bash = lib.mkOption {
          type = lib.types.bool;
          default = true;
          description = "Enable bash shell integration";
        };

        zsh = lib.mkOption {
          type = lib.types.bool;
          default = true;
          description = "Enable zsh shell integration";
        };

        fish = lib.mkOption {
          type = lib.types.bool;
          default = true;
          description = "Enable fish shell integration";
        };
      };
    };
  };

  config = lib.mkIf cfg.enable {
    home.packages = [ cfg.package ];

    home.sessionVariables = {
      NVS_CONFIG_DIR = cfg.configDir;
      NVS_CACHE_DIR = cfg.cacheDir;
      NVS_BIN_DIR = cfg.binDir;
    };

    home.sessionPath = [
      cfg.binDir
    ];

    # Ensure directories exist
    home.activation.createNvsDirectories = lib.hm.dag.entryAfter [ "writeBoundary" ] ''
      run mkdir -p "${cfg.configDir}" "${cfg.cacheDir}" "${cfg.binDir}"
    '';

    programs.bash.initExtra = lib.mkIf (cfg.enableShellIntegration && cfg.shellIntegration.bash) ''
      eval "$(${cfg.package}/bin/nvs env --source --shell bash)"
      ${lib.optionalString cfg.enableAutoSwitch ''
        eval "$(${cfg.package}/bin/nvs hook bash)"
      ''}
    '';

    programs.zsh.initExtra = lib.mkIf (cfg.enableShellIntegration && cfg.shellIntegration.zsh) ''
      eval "$(${cfg.package}/bin/nvs env --source --shell zsh)"
      ${lib.optionalString cfg.enableAutoSwitch ''
        eval "$(${cfg.package}/bin/nvs hook zsh)"
      ''}
    '';

    programs.fish.shellInit = lib.mkIf (cfg.enableShellIntegration && cfg.shellIntegration.fish) ''
      ${cfg.package}/bin/nvs env --source --shell fish | source
      ${lib.optionalString cfg.enableAutoSwitch ''
        ${cfg.package}/bin/nvs hook fish | source
      ''}
    '';
  };
}
