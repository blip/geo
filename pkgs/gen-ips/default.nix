{
  writeShellApplication,
  coreutils,
  geoipupdate,
  convert-ips,
}:
writeShellApplication {
  name = "gen-ips";
  runtimeInputs = [
    coreutils
    geoipupdate
    convert-ips
  ];
  text = builtins.readFile ./gen-ips.sh;
}
