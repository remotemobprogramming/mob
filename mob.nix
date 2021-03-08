{buildGoPackage, fetchFromGitHub, lib, ...}:
buildGoPackage rec {
    pname = "mob.sh";
    version = "1.3.0";
    owner = "remotemobprogramming";
    repo = "mob";
    src = fetchFromGitHub {
      owner = owner;
      repo = repo;
      rev = "v${version}";
      sha256 = "04x6cl2r4ja41cmy82p5apyavmdvak6jsclzf2l7islf0pmsnddv";
    };

    goPackagePath = "github.com/${owner}/${repo}/";

    subPackages = [ "." ];
    
    meta = {
      description = "Remote mob programming tool";
      homepage = "https://mob.sh";
      license = lib.licenses.mit;
    };
}
