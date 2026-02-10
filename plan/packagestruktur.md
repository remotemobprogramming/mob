# Plan: Packagestruktur fuer mob.sh

## 1. Analyse des Ist-Zustands

### Was ist mob.sh?

mob.sh ist ein CLI-Tool fuer **Remote Mob Programming**. Es ist ein duenner Wrapper um Git, der den schnellen Wechsel (Handover) zwischen Mob-Programming-Teilnehmern ermoeglicht. Die Kern-Fachlichkeit umfasst:

- **Session-Management**: `start` (Session beginnen), `next` (Uebergabe an naechste Person), `done` (Session beenden)
- **WIP-Branch-Verwaltung**: Automatische Verwaltung temporaerer WIP-Branches (`mob/<base-branch>`)
- **Commit-Handling**: Automatische WIP-Commits, Squash-Strategien, Co-Author-Tracking
- **Timer-Integration**: Lokale Timer und Remote-Timer via timer.mob.sh
- **Konfiguration**: Mehrstufig (Defaults, Env-Vars, User-Config, Projekt-Config)

### Aktuelle Code-Struktur

```
mob/
├── mob.go                    # 1314 Zeilen - Kern-Logik: CLI-Entry, Git-Ops, Branch-Logik, Commands
├── timer.go                  # 192 Zeilen  - Timer-Logik (lokal + remote)
├── status.go                 # 33 Zeilen   - Status-Anzeige
├── coauthors.go              # 135 Zeilen  - Co-Author-Tracking
├── squash_wip.go             # 236 Zeilen  - Squash-Commit-Handling via interaktives Rebase
├── find_next.go              # 82 Zeilen   - Algorithmus: Wer ist als naechstes dran?
├── configuration/            # Bereits eigenes Package (564 Zeilen)
├── say/                      # Bereits eigenes Package (85 Zeilen) - Logging/Output
├── help/                     # Bereits eigenes Package (72 Zeilen)
├── open/                     # Bereits eigenes Package - Plattformuebergreifendes Browser-Oeffnen
├── goal/                     # Bereits eigenes Package (138 Zeilen) - Timer-Room-Goals
├── httpclient/               # Bereits eigenes Package (82 Zeilen) - HTTP-Client
└── test/                     # Bereits eigenes Package - Test-Utilities
```

### Probleme der aktuellen Struktur

1. **Ueberladenes `main`-Package**: ~1990 Zeilen ueber 6 Dateien, mit vermischten Verantwortlichkeiten
2. **Globaler Zustand**: `workingDir`, `args`, `GitPassthroughStderrStdout` sind globale Variablen
3. **Vermischte Concerns in mob.go**:
   - Git-Kommando-Ausfuehrung (`git()`, `silentgit()`, `runCommand()`)
   - Branch-Domaenen-Modell (`Branch` struct + Methoden)
   - Session-Kommandos (`start()`, `next()`, `done()`, `reset()`, `clean()`)
   - CLI-Routing (`execute()`)
   - Hilfsfunktionen (`stringContains()`, `ReverseSlice()`, etc.)
4. **Enge Kopplung**: Branch-Methoden rufen direkt Git-Funktionen auf (z.B. `hasRemoteBranch()` ruft `gitRemoteBranches()`)
5. **Exit-Aufrufe mitten in der Logik**: `git()` und `silentgit()` rufen bei Fehler `Exit(1)` auf, statt Fehler zurueckzugeben

---

## 2. Geteilte Infrastruktur-Funktionen: Wo gehoeren sie hin?

Das Hauptproblem bei der Restructurierung ist, dass viele Funktionen im `main`-Package
von mehreren Bereichen gleichzeitig genutzt werden. Hier die Analyse:

### Schicht 1: Rohe Kommando-Ausfuehrung

| Funktion | Genutzt von | Ziel-Package |
|----------|-------------|--------------|
| `runCommandSilent(name, args...)` | `silentgit()`, `doBranchesDiverge()`, `gitVersion()`, `isGit()` | `git/` |
| `runCommand(name, args...)` | `git()`, `gitIgnoreFailure()` | `git/` |
| `startCommand(name, args...)` | `executeCommandsInBackgroundProcess()`, `openLastModifiedFileIfPresent()` | bleibt in `main` |
| `executeCommandsInBackgroundProcess(cmds...)` | `timer.go` (2x), `moo()` | bleibt in `main` |

**Entscheidung**: `runCommand` und `runCommandSilent` gehen ins `git/`-Package, weil sie
*ausschliesslich* fuer Git-Kommandos verwendet werden. `startCommand` und
`executeCommandsInBackgroundProcess` starten Nicht-Git-Prozesse (Timer-Sleep, Voice-Commands,
IDE-Open) und bleiben daher in `main`. Wenn `timer/` spaeter ein eigenes Package wird,
bekommen diese Funktionen einen eigenen Platz (z.B. `process/` oder als Parameter uebergeben).

### Schicht 2: Git-Wrapper

| Funktion | Genutzt von | Ziel-Package |
|----------|-------------|--------------|
| `git(args...)` | mob.go (30x), squash_wip.go (4x), Tests (60x+) | `git/` |
| `silentgit(args...)` | mob.go (20x), status.go, squash_wip.go (3x), Tests | `git/` |
| `gitIgnoreFailure(args...)` | mob.go:done() | `git/` |
| `gitWithoutEmptyStrings(args...)` | mob.go (4x) | `git/` |
| `silentgitignorefailure(args...)` | mob.go (3x) | `git/` |
| `gitHooksOption(c)` | mob.go (5x), squash_wip.go (2x) | `git/` |

**Design**: Diese Funktionen werden Methoden auf einem `git.Context`-Struct, das den
globalen Zustand kapselt:

```go
package git

type Context struct {
    WorkingDir              string
    PassthroughStderrStdout bool  // fuer Git-Hooks
}

func (g *Context) Run(args ...string)                    { ... }  // vorher: git()
func (g *Context) Silent(args ...string) string          { ... }  // vorher: silentgit()
func (g *Context) IgnoreFailure(args ...string) error    { ... }  // vorher: gitIgnoreFailure()
```

### Schicht 3: Git-Info-Funktionen

| Funktion | Genutzt von | Ziel-Package |
|----------|-------------|--------------|
| `gitCurrentBranch()` | mob.go (8x), timer.go, status.go | `git/` |
| `gitBranches()` | mob.go (8x), timer.go, status.go, Branch-Methoden | `git/` |
| `gitRemoteBranches()` | mob.go, Branch-Methoden | `git/` |
| `gitUserName()` | mob.go:showNext(), timer.go | `git/` |
| `gitUserEmail()` | coauthors.go | `git/` |
| `isGit()` | mob.go:run(), timer.go, Fehlerbehandlung | `git/` |
| `gitRootDir()` | mob.go (3x) | `git/` |
| `gitDir()` | mob.go:done() | `git/` |
| `hasCommits()` | mob.go:run() | `git/` |
| `doBranchesDiverge(a, b)` | mob.go:startJoinMobSession() | `git/` |

**Design**: Werden ebenfalls Methoden auf `git.Context`:

```go
func (g *Context) CurrentBranch() string       { ... }
func (g *Context) Branches() []string          { ... }
func (g *Context) RemoteBranches() []string    { ... }
func (g *Context) UserName() string            { ... }
func (g *Context) UserEmail() string           { ... }
func (g *Context) IsGitRepo() bool             { ... }
func (g *Context) RootDir() string             { ... }
```

### Querschnitt: Utility-Funktionen

| Funktion | Genutzt von | Ziel-Package |
|----------|-------------|--------------|
| `injectCommandWithMessage(cmd, msg)` | mob.go:openCommandFor(), timer.go (2x) | bleibt in `main` (spaeter eigenes Utility-Package oder wird Parameter) |
| `stringContains(list, element)` | mob.go, Branch-Methoden | wird durch `slices.Contains()` ersetzt (Go 1.21+) oder geht mit Branch nach `branch/` |
| `deleteEmptyStrings(s)` | mob.go:gitWithoutEmptyStrings() | geht mit nach `git/` |
| `ReverseSlice(s)` | mob.go:sayLastCommitsList() | bleibt in `main` |

### Zusammenfassung: Globaler Zustand

| Variable | Aktuell | Ziel |
|----------|---------|------|
| `workingDir` | Globale Variable in main | Feld in `git.Context.WorkingDir` |
| `GitPassthroughStderrStdout` | Globale Variable in main | Feld in `git.Context.PassthroughStderrStdout` |
| `args` | Globale Variable in main | Lokale Variable in `run()`, nur noch fuer CLI-Parsing |
| `Exit` | Globale `var` in main | Bleibt als globale var oder wird Parameter im `git.Context` |

### Uebergangsphase

Wichtig: Beim Extrahieren von `git/` koennen Funktionen wie `startCommand` und
`executeCommandsInBackgroundProcess` zunaechst in `main` bleiben. Sie werden erst
beim Extrahieren von `timer/` relevant. Die Strategie ist:

1. `git/`-Package nimmt alles Git-spezifische auf
2. `main` behaelt vorerst die Nicht-Git-Prozesse (`startCommand`, `executeCommandsInBackgroundProcess`)
3. `timer/` bekommt spaeter `executeCommandsInBackgroundProcess` als Dependency injected
4. `injectCommandWithMessage` wandert entweder nach `timer/` oder wird inline aufgeloest

---

## 3. Ziel-Packagestruktur (Vision)

```
mob/
├── main.go                     # Nur noch Entry-Point + CLI-Routing (~100 Zeilen)
│
├── git/                        # NEU: Git-Kommando-Ausfuehrung (Infrastruktur-Layer)
│   └── git.go                  #   git(), silentgit(), runCommand(), gitVersion(), isGit()
│
├── branch/                     # NEU: Branch-Domaenen-Modell
│   └── branch.go               #   Branch struct, determineBranches(), WIP-Branch-Logik
│
├── session/                    # NEU: Session-Kommandos (Applikations-Layer)
│   └── session.go              #   start(), next(), done(), reset(), clean(), status()
│
├── timer/                      # NEU: Timer-Logik (umbenannt aus main)
│   └── timer.go                #   StartTimer(), StartBreakTimer(), lokaler/remote Timer
│
├── squash/                     # NEU: Squash-WIP-Logik
│   └── squash.go               #   squashWip(), Git-Editor-Callbacks
│
├── coauthor/                   # NEU: Co-Author-Tracking
│   └── coauthor.go             #   parseCoauthors(), appendCoauthorsToSquashMsg()
│
├── findnext/                   # NEU: Naechster-Typist-Algorithmus
│   └── findnext.go             #   findNextTypist() - reiner Algorithmus
│
├── configuration/              # Bereits vorhanden (ungeaendert)
├── say/                        # Bereits vorhanden (ungeaendert)
├── help/                       # Bereits vorhanden (ungeaendert)
├── open/                       # Bereits vorhanden (ungeaendert)
├── goal/                       # Bereits vorhanden (ungeaendert)
├── httpclient/                 # Bereits vorhanden (ungeaendert)
└── test/                       # Bereits vorhanden (ungeaendert)
```

### Abhaengigkeits-Hierarchie (von unten nach oben)

```
say, configuration, httpclient, open       (Basis-Infrastruktur, existiert bereits)
         |
    findnext                                (reiner Algorithmus, keine Abhaengigkeiten)
         |
       git/                                 (Git-Kommando-Ausfuehrung, kapselt workingDir)
       (git.Context struct)                 (abhaengig von: say)
         |
      branch/                               (Domaenen-Modell)
                                            (abhaengig von: git/, configuration)
         |
  coauthor/, squash/, timer/                (Feature-Module)
                                            (abhaengig von: git/, branch/, configuration)
         |
     session/                               (Applikations-Logik, orchestriert alles)
                                            (abhaengig von: allen obigen Packages)
         |
      main.go                               (Entry-Point, CLI-Routing)
                                            (behaelt: startCommand, executeCommandsInBackground,
                                             injectCommandWithMessage bis spaetere Extraktion)
```

---

## 4. Reihenfolge der Extraktion

| Schritt | Package      | Komplexitaet | Was passiert mit geteilten Funktionen? |
|---------|-------------|-------------|----------------------------------------|
| **1**   | `findnext/` | Sehr niedrig | Keine geteilten Funktionen betroffen |
| 2       | `coauthor/` | Niedrig      | `gitUserEmail()` wird als Parameter uebergeben |
| 3       | `git/`      | Mittel-Hoch  | `runCommand*`, `git()`, `silentgit()`, alle Git-Info-Fns wandern hierher. `git.Context` kapselt `workingDir` + `GitPassthroughStderrStdout`. `startCommand` + `executeCommandsInBackgroundProcess` + `injectCommandWithMessage` bleiben vorerst in main |
| 4       | `branch/`   | Mittel       | `Branch` struct + Methoden. Bekommt `git.Context` als Abhaengigkeit |
| 5       | `squash/`   | Mittel       | Bekommt `git.Context` als Abhaengigkeit |
| 6       | `timer/`    | Niedrig      | Bekommt `executeCommandsInBackgroundProcess` + `injectCommandWithMessage` als Dependency injected oder diese wandern in ein kleines `process/`-Package |
| 7       | `session/`  | Hoch         | Orchestriert alles. `main.go` wird zum reinen Entry-Point |

---

## 5. Erster Schritt: Package `findnext/` extrahieren

### Warum `findnext/` als erstes?

1. **Null Abhaengigkeiten**: `findNextTypist()` ist ein reiner Algorithmus - er nimmt `[]string` und `string` und gibt Ergebnisse zurueck. Kein Git, kein Config, kein IO.
2. **Eigene Test-Datei**: `find_next_test.go` testet den Algorithmus isoliert.
3. **Klar definierte Schnittstelle**: Eine einzige exportierte Funktion.
4. **Geringes Risiko**: Kein globaler Zustand, keine Seiteneffekte.
5. **Muster-Etablierung**: Zeigt das Extraktionsmuster fuer alle folgenden Schritte.

### Konkrete Schritte

#### Schritt 1.1: Package erstellen

Neues Verzeichnis `findnext/` mit Datei `findnext.go` erstellen.

#### Schritt 1.2: Code verschieben

Aus `find_next.go` (aktuell `package main`) in `findnext/findnext.go` verschieben:

```go
package findnext

// FindNextTypist bestimmt anhand der Commit-Historie, wer als naechstes tippen sollte.
// lastCommitters ist die Liste der letzten Committer (neuester zuerst).
// gitUserName ist der Name des aktuellen Git-Users.
func FindNextTypist(lastCommitters []string, gitUserName string) (nextTypist string, previousCommitters []string) {
    // ... bestehende Implementierung von findNextTypist() ...
}

// Hilfsfunktionen (unexportiert, bleiben im Package)
func reverse(list []string) []string { ... }
func lookahead(processedCommitters []string, previousCommitters []string) string { ... }
func contains(list []string, element string) bool { ... }
func min(a int, b int) int { ... }
func prepend(list []string, element string) []string { ... }
```

Aenderungen:
- `findNextTypist` -> `FindNextTypist` (exportiert)
- Alle Hilfsfunktionen bleiben unexportiert
- Package-Deklaration: `package findnext`

#### Schritt 1.3: Tests verschieben

Aus `find_next_test.go` in `findnext/findnext_test.go` verschieben:

```go
package findnext

// Alle bestehenden Tests, mit Anpassung:
// findNextTypist -> FindNextTypist
```

#### Schritt 1.4: Aufrufer anpassen

In `mob.go` den Import hinzufuegen und die Aufrufe anpassen:

```go
import (
    "github.com/remotemobprogramming/mob/v5/findnext"
)

// In showNext():
// ALT:  nextTypist, previousCommitters := findNextTypist(lines, gitUserName)
// NEU:  nextTypist, previousCommitters := findnext.FindNextTypist(lines, gitUserName)
```

#### Schritt 1.5: Alte Dateien loeschen

`find_next.go` und `find_next_test.go` aus dem Root-Verzeichnis loeschen.

#### Schritt 1.6: Tests ausfuehren

```bash
go test ./...
```

Alle Tests muessen gruen sein, bevor der Schritt als abgeschlossen gilt.

### Erwartetes Ergebnis nach Schritt 1

```
mob/
├── mob.go                    # Import von findnext, Aufruf angepasst
├── findnext/                 # NEU
│   ├── findnext.go           # FindNextTypist() + Hilfsfunktionen
│   └── findnext_test.go      # Tests
├── find_next.go              # GELOESCHT
├── find_next_test.go         # GELOESCHT
└── ... (Rest unveraendert)
```

### Risikobewertung

- **Risiko**: Sehr gering
- **Abwaertskompatibilitaet**: Keine oeffentliche API betroffen (alles intern)
- **Testabdeckung**: Bestehende Tests decken die Funktionalitaet ab
- **Rollback**: Einfach rueckgaengig zu machen

---

## 6. Ausblick: Zweiter Schritt (`coauthor/`)

Nach erfolgreichem Abschluss von Schritt 1 waere die Extraktion von `coauthor/` der logische naechste Schritt:

- `coauthors.go` ist weitgehend eigenstaendig
- Einzige externe Abhaengigkeit: `gitUserEmail()` - kann als Parameter uebergeben werden
- Hat eigene Tests (`coauthors_test.go`)
- Aehnliches Vorgehen wie bei `findnext/`

Die Signatur von `collectCoauthorsFromWipCommits` wuerde sich aendern:

```go
// ALT (ruft intern gitUserEmail() auf):
func collectCoauthorsFromWipCommits(file *os.File) []Author

// NEU (bekommt die Email als Parameter):
func CollectCoauthorsFromWipCommits(file *os.File, currentUserEmail string) []Author
```

---

## 7. Prinzipien fuer die gesamte Umstrukturierung

1. **Bottom-Up**: Immer zuerst die Teile extrahieren, die keine Abhaengigkeiten nach "oben" haben
2. **Ein Package pro Schritt**: Jeder Schritt ist ein eigener, testbarer Commit
3. **Tests zuerst gruen**: Vor und nach jedem Schritt muessen alle Tests bestehen
4. **Abhaengigkeiten als Parameter**: Statt globale Funktionen aufzurufen, Abhaengigkeiten explizit uebergeben
5. **Kein Verhalten aendern**: Reine Struktur-Aenderung, keine funktionalen Aenderungen
6. **Globalen Zustand schrittweise eliminieren**: `workingDir` und andere Globals werden spaeter durch ein `git.Context`-Objekt ersetzt
