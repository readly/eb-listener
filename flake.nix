{
  description = "EventBridge listener";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { nixpkgs, flake-utils, ... }: flake-utils.lib.eachDefaultSystem (system:
    let
      pkgs = import nixpkgs { inherit system; config.allowUnfree = true; };
    in
    {
      devShells.default = pkgs.mkShell {
        buildInputs = (with pkgs; [
          delve
          go_1_22
          goreleaser
        ]);
      };
    }
  );
}

