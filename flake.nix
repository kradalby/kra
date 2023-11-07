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
        krapage = pkgs.buildGo121Module {
          pname = "krapage";
          version = kraVersion;
          src = pkgs.nix-gitignore.gitignoreSource [] ./.;

          subPackages = ["cmd/krapage"];

          patchPhase = ''
            ${pkgs.nodePackages.tailwindcss}/bin/tailwind --input ./input.css --output ./cmd/krapage/static/tailwind.css
          '';

          vendorSha256 = "sha256-7AsE8J/1vkX0gklVATlCCp99AQ5O2EyIgO5b6Z3Zl7s=";
        };
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
        go_1_21
      ];
      devDeps = with pkgs;
        buildDeps
        ++ [
          golangci-lint
          entr
          nodePackages.tailwindcss
        ];
    in rec {
      # `nix develop`
      devShell = pkgs.mkShell {
        buildInputs = with pkgs;
          [
            (writeShellScriptBin
              "krarun"
              ''
                # if [ ! -f ./static/tailwind.css ]
                # then
                    # echo "static/tailwind.css does not exist, creating..."
                    tailwind --input ./input.css --output ./static/tailwind.css
                # fi
                go run ./cmd/krapage --verbose
              '')
            (writeShellScriptBin
              "kradev"
              ''
                ls *.go | entr -r krarun
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
          services.krapagepage = {
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
