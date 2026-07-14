{
  description = "felicia — map-based travel journal (system tooling devShell)";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "aarch64-darwin"
        "x86_64-darwin"
        "aarch64-linux"
        "x86_64-linux"
      ];
      forAll = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      devShells = forAll (pkgs: {
        # Runtimes (go, bun) come from mise — intentionally NOT here.
        # This shell provides the system tools the Makefile wraps via NIX_RUN.
        default = pkgs.mkShell {
          packages = with pkgs; [
            golangci-lint
            goose
            postgresql_16
            postgresqlPackages.postgis
            gnumake
            sqlc
            uv
          ];
        };
      });
    };
}
