{
  writeShellApplication,
  coreutils,
  gnused,
  gawk,
  curl,
  convert-domains,
}:
writeShellApplication {
  name = "gen-domains";
  runtimeInputs = [
    coreutils
    gnused
    gawk
    curl
    convert-domains
  ];
  text = builtins.readFile ./gen-domains.sh;
}
