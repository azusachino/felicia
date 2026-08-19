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
          # Browsers for the admin-GUI E2E pass come from nix so they run in
          # non-FHS containers too; @playwright/test in apps/felicia-web must
          # stay on the same version as this playwright-driver (1.60.0).
          PLAYWRIGHT_BROWSERS_PATH = pkgs.playwright-driver.browsers.override {
            withFirefox = false;
            withWebkit = false;
          };
          PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS = "true";
          # Headless chromium needs at least one font to lay text out
          # (containers ship none; zero-size text reads as "hidden" to
          # Playwright's visibility checks).
          FONTCONFIG_FILE = pkgs.makeFontsConf {
            fontDirectories = [ pkgs.dejavu_fonts ];
          };
        };
      });
    };
}
