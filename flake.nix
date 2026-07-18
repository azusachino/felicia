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
        # Keep the repository toolchain in one reproducible shell for local
        # development and CI. PostgreSQL/PostGIS is the only database stack.
        default = pkgs.mkShell {
          packages = with pkgs; [
            go_1_26
            bun
            golangci-lint
            goose
            # Keep the local client and extension toolchain aligned with the
            # postgis/postgis:18-3.6 service used by Compose and CI.
            postgresql_18
            postgresql18Packages.postgis
            gnumake
            sqlc
            prettier
            uv
            # Provide the interpreter pinned by .python-version so uv can run
            # with a system Python everywhere; uv-managed manylinux builds
            # cannot exec inside non-FHS (nix) containers.
            python314
          ];
        };
      });
    };
}
