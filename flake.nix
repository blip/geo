{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
  };

  outputs =
    inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } (
      { lib, ... }:
      {
        systems = lib.systems.flakeExposed;
        perSystem =
          { config, pkgs, ... }:
          {
            packages = {
              gen-domains = pkgs.callPackage ./pkgs/gen-domains {
                convert-domains = pkgs.callPackage ./pkgs/convert-domains { };
              };
            };
            devShells.default = pkgs.mkShellNoCC {
              name = "shell";
              packages = [
                config.packages.gen-domains
              ];
            };
          };
      }
    );
}
