mobsh = buildGoPackage rec {
    pname = "mob.sh";
    version = "1.3.0";
    src = fetchFromGitHub {
      owner = "remotemobprogramming";
      repo = "mob";
      rev = "v${version}";
      sha256 = "04x6cl2r4ja41cmy82p5apyavmdvak6jsclzf2l7islf0pmsnddv";
    };

    goPackagePath = "github.com/remotemobprogramming/mob/";

    subPackages = [ "." ];
    
    meta = {
      description = "Remote mob programming tool";
      homepage = "https://mob.sh";
      license = lib.licenses.mit;
    };

};

