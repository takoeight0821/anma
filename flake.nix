{
  description = "A simple Go package";

  # Nixpkgs / NixOS version to use.
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
  }:
    flake-utils.lib.eachDefaultSystem (
      system: let
        # to work with older version of flakes
        lastModifiedDate = self.lastModifiedDate or self.lastModified or "19700101";
        # Generate a user-friendly version number.
        version = builtins.substring 0 8 lastModifiedDate;
        pkgs = import nixpkgs {inherit system;};
        anma = pkgs.buildGoModule {
          pname = "anma";
          inherit version;
          # In 'nix develop', we don't need a copy of the source tree
          # in the Nix store.
          src = ./.;

          # This hash locks the dependencies of this package. It is
          # necessary because of how Go requires network access to resolve
          # VCS.  See https://www.tweag.io/blog/2021-03-04-gomod2nix/ for
          # details. Normally one can build with a fake hash and rely on native Go
          # mechanisms to tell you what the hash should be or determine what
          # it should be "out-of-band" with other tooling (eg. gomod2nix).
          # To begin with it is recommended to set this, but one must
          # remember to bump this hash when your dependencies change.
          # vendorHash = pkgs.lib.fakeHash;

          vendorHash = "sha256-SJ71q2OCDxbsDOneSZFe6CrKnmFHbOB4QRYwlBaU1dU=";
        };
      in {
        packages.default = anma;
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [go gopls gotools go-tools];
        };
        formatter = pkgs.alejandra;
      }
    );
}
