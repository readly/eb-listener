{
  description = "EventBridge listener";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, utils }:
    utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config.allowUnfree = true;
        };
        eb-listener = pkgs.buildGoModule rec {
          pname = "eb-listener";
          version = "0.1.0";
          src = pkgs.fetchFromGitHub {
            owner = "readly";
            repo = "eb-listener";
            rev = "v${version}";
            #hash = pkgs.lib.fakeHash;
            hash = "sha256-/OjKARWuTJPS8RTcOgXFFRO6Gdo/CvOEE1UvlWCcw8g=";
          };
          #vendorHash = pkgs.lib.fakeHash;
          vendorHash = "sha256-W18cjpeJIFVWfq0lmVkaxGw0VVDxwrMG3Hxg2sTS7eE=";
          meta = {
            description = "";
            homepage = "https://github.com/readly/eb-listener";
            maintainers = with pkgs.maintainers; [ smgt ];
          };
        };
      in {
        packages = { default = eb-listener; };
        devShells.default =
          pkgs.mkShell { buildInputs = with pkgs; [ delve go goreleaser ]; };
      });
}
