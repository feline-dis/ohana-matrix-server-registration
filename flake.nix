{
  description = "Invite-code-gated registration proxy for Matrix homeservers";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" "aarch64-linux" ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
    in
    {
      packages = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          registration-proxy = pkgs.buildGoModule {
            pname = "registration-proxy";
            version = "0.1.0";
            src = ./registration;
            vendorHash = null;
            postInstall = ''
              mv $out/bin/registration $out/bin/registration-proxy
            '';
          };

          default = self.packages.${system}.registration-proxy;
        });
    };
}
