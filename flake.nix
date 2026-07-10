{
  description = "Reverse proxy for SNU sugang syllabus pages";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      packages = forAllSystems (pkgs: rec {
        snutt-proxy = pkgs.buildGoModule {
          pname = "snutt-proxy";
          version = "0.1.0";
          src = ./.;
          vendorHash = null;
        };
        default = snutt-proxy;
      });

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gotools
          ];
        };
      });
    };
}
