{
  description = "nono-go: build the bundled nono C library (libnono_ffi.a and nono.h) with Nix";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/d3b13ad75cc492690728f9307e443dec9ef9a20e";
  # candacelabs/nono main: upstream nono v0.81.0 plus optional sigstore
  # (nolabs-ai/nono#2030) and nono_sandbox_restrict_execute in the C binding.
  # flake.lock pins the commit; `nix flake update nono` moves it.
  inputs.nono = {
    url = "github:candacelabs/nono";
    flake = false;
  };

  outputs = { self, nixpkgs, nono }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};

      # nono-ffi as a static library and its cbindgen header, for x86_64 Linux.
      # The FFI crate builds nono without default features, so no sigstore,
      # TLS or aws-lc, no keyring, no D-Bus: pure Rust, which links against
      # any glibc from 2.28 on.
      nonoFfi = pkgs.stdenv.mkDerivation {
        pname = "nono-ffi";
        version = "0.81.0";
        src = nono;
        cargoDeps = pkgs.rustPlatform.importCargoLock { lockFile = "${nono}/Cargo.lock"; };
        nativeBuildInputs = [ pkgs.rustPlatform.cargoSetupHook pkgs.cargo pkgs.rustc ];
        dontConfigure = true;
        buildPhase = ''
          runHook preBuild
          export HOME=$TMPDIR
          cargo build --release --offline -p nono-ffi
          runHook postBuild
        '';
        installPhase = ''
          runHook preInstall
          mkdir -p $out/lib $out/include
          cp target/release/libnono_ffi.a $out/lib/
          cp bindings/c/include/nono.h $out/include/nono.h
          echo ${nono.rev} > $out/VERSION
          runHook postInstall
        '';
      };
    in
    {
      packages.${system} = {
        nono-ffi = nonoFfi;
        default = nonoFfi;
      };
    };
}
