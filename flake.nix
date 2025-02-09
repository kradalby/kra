{
  description = "kra - webpage and helper modules";

  inputs = {
    nixpkgs.url = "nixpkgs/nixpkgs-unstable";
    utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    utils,
    ...
  }: let
    kraVersion =
      if (self ? shortRev)
      then self.shortRev
      else "dev";
  in
    {
      overlay = _: prev: let
        pkgs = nixpkgs.legacyPackages.${prev.system};
      in {
        krapage = pkgs.callPackage ({buildGoModule}:
          buildGoModule {
            pname = "krapage";
            version = kraVersion;
            src = pkgs.nix-gitignore.gitignoreSource [] ./.;

            subPackages = ["cmd/krapage"];

            vendorHash = "sha256-n9hPKuMC3b+Prd8AokBU2ZqtfdqolmP83ESKfB3B2vQ=";
          }) {};
      };
    }
    // utils.lib.eachDefaultSystem
    (system: let
      pkgs = import nixpkgs {
        overlays = [self.overlay];
        inherit system;
      };
      buildDeps = with pkgs; [
        git
        gnumake
        go
      ];
      devDeps = with pkgs;
        buildDeps
        ++ [
          golangci-lint
          entr
        ];
    in rec {
      # `nix develop`
      devShell = pkgs.mkShell {
        buildInputs = with pkgs;
          [
            (writeShellScriptBin
              "krarun"
              ''
                go run ./cmd/krapage --verbose
              '')
            (writeShellScriptBin
              "kradev"
              ''
                fd .go | entr -r krarun
              '')

            (writeShellScriptBin
              "nix-vendor-sri"
              ''
                set -eu
                OUT=$(mktemp -d -t nar-hash-XXXXXX)
                rm -rf "$OUT"
                go mod vendor -o "$OUT"
                go run tailscale.com/cmd/nardump --sri "$OUT"
                rm -rf "$OUT"
              '')

            (writeShellScriptBin
              "go-mod-update-all"
              ''
                cat go.mod | ${pkgs.silver-searcher}/bin/ag "\t" | ${pkgs.silver-searcher}/bin/ag -v indirect | ${pkgs.gawk}/bin/awk '{print $1}' | ${pkgs.findutils}/bin/xargs go get -u
                go mod tidy
              '')
          ]
          ++ devDeps;
      };

      # `nix build`
      packages = with pkgs; {
        inherit krapage;
      };

      defaultPackage = pkgs.krapage;

      # `nix run`
      apps = {
        krapage = utils.lib.mkApp {
          drv = packages.krapage;
        };
      };

      defaultApp = apps.krapage;

      overlays.default = self.overlay;
    })
    // {
      nixosModules.default = {
        pkgs,
        lib,
        config,
        ...
      }: let
        cfg = config.services.krapage;
      in {
        options = with lib; {
          services.krapage = {
            enable = mkEnableOption "Enable krapage";

            package = mkOption {
              type = types.package;
              description = ''
                krapage package to use
              '';
              default = pkgs.krapage;
            };

            user = mkOption {
              type = types.str;
              default = "krapage";
              description = "User account under which krapage runs.";
            };

            group = mkOption {
              type = types.str;
              default = "krapage";
              description = "Group account under which krapage runs.";
            };

            dataDir = mkOption {
              type = types.path;
              default = "/var/lib/krapage";
              description = "Path to data dir";
            };

            tailscaleKeyPath = mkOption {
              type = types.path;
            };

            verbose = mkOption {
              type = types.bool;
              default = false;
            };

            localhostPort = mkOption {
              type = types.port;
              default = 56661;
            };

            environmentFile = mkOption {
              type = types.nullOr types.path;
              default = null;
              example = "/var/lib/secrets/krapageSecrets";
            };
          };
        };
        config = lib.mkIf cfg.enable {
          systemd.services.krapage = {
            enable = true;
            script = let
              args =
                [
                  "--ts-key-path ${cfg.tailscaleKeyPath}"
                  "--listen-addr localhost:${toString cfg.localhostPort}"
                ]
                ++ lib.optionals cfg.verbose ["--verbose"];
            in ''
              ${cfg.package}/bin/krapage ${builtins.concatStringsSep " " args}
            '';
            wantedBy = ["multi-user.target"];
            after = ["network-online.target"];
            serviceConfig = {
              User = cfg.user;
              Group = cfg.group;
              Restart = "always";
              RestartSec = "15";
              WorkingDirectory = cfg.dataDir;
              EnvironmentFile = lib.optional (cfg.environmentFile != null) cfg.environmentFile;
            };
            path = [cfg.package];
          };
        };
      };
    };
}
