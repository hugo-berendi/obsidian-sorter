{
  description = "Go-based Obsidian sorter with Ollama";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs";
  };

  outputs = {
    self,
    nixpkgs,
    ...
  }: let
    system = "x86_64-linux";
    pkgs = nixpkgs.legacyPackages.${system};
  in {
    packages.${system}.default = pkgs.buildGoModule {
      pname = "obsidian-sorter";
      version = "0.1.0";
      src = ./.;
      vendorHash = "sha256-lAMeiE450pdE7aVpMWHcQq48egI3zr6t04iklA+s0Ps=";
      nativeBuildInputs = [pkgs.go];
    };

    nixosModules.obsidian-sorter = {
      config,
      lib,
      pkgs,
      ...
    }:
      with lib; let
        cfg = config.services.obsidian-sorter;
      in {
        options = {
          services.obsidian-sorter = {
            enable = mkEnableOption "Enable the Obsidian PARA sorter service";

            vaultDir = mkOption {
              type = types.str;
              description = "Path to the Obsidian vault directory.";
            };

            ollamaURL = mkOption {
              type = types.str;
              default = "http://localhost:11434";
              description = "URL of the Ollama server.";
            };

            interval = mkOption {
              type = types.str;
              default = "30m";
              description = "How often the sorter runs (e.g., '30m', '1h').";
            };
          };
        };

        config = mkIf cfg.enable {
          environment.variables = {
            VAULT_PATH = cfg.vaultDir;
            OLLAMA_URL = cfg.ollamaURL;
          };

          systemd.services.obsidian-sorter = {
            description = "Obsidian PARA Sorter Service";
            after = ["network.target"];
            wants = ["network.target"];
            serviceConfig = {
              ExecStart = "${pkgs.obsidian-sorter}/bin/obsidian-sorter";
              Restart = "always";
              Environment = [
                "VAULT_PATH=${cfg.vaultDir}"
                "OLLAMA_URL=${cfg.ollamaURL}"
              ];
            };
          };

          systemd.timers.obsidian-sorter = {
            wantedBy = ["timers.target"];
            timerConfig = {
              OnBootSec = "5m";
              OnUnitActiveSec = cfg.interval;
              Unit = "obsidian-sorter.service";
            };
          };
        };
      };
  };
}
