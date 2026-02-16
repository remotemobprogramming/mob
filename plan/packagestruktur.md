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

**Design**: Diese Funktionen werden Methoden auf einem `git.Client`-Struct, das den
globalen Zustand kapselt:

```go
package git

type Client struct {
    WorkingDir              string
    PassthroughStderrStdout bool  // fuer Git-Hooks
}

func (g *Client) Run(args ...string)                    { ... }  // vorher: git()
func (g *Client) Silent(args ...string) string          { ... }  // vorher: silentgit()
func (g *Client) IgnoreFailure(args ...string) error    { ... }  // vorher: gitIgnoreFailure()
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

**Design**: Werden ebenfalls Methoden auf `git.Client`:

```go
func (g *Client) CurrentBranch() string       { ... }
func (g *Client) Branches() []string          { ... }
func (g *Client) RemoteBranches() []string    { ... }
func (g *Client) UserName() string            { ... }
func (g *Client) UserEmail() string           { ... }
func (g *Client) IsGitRepo() bool             { ... }
func (g *Client) RootDir() string             { ... }
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
| `workingDir` | Globale Variable in main | Feld in `git.Client.WorkingDir` |
| `GitPassthroughStderrStdout` | Globale Variable in main | Feld in `git.Client.PassthroughStderrStdout` |
| `args` | Globale Variable in main | Lokale Variable in `run()`, nur noch fuer CLI-Parsing |
| `Exit` | Globale `var` in main | Bleibt als globale var oder wird Parameter im `git.Client` |

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
       (git.Client struct)                 (abhaengig von: say)
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
| 3       | `git/`      | Mittel-Hoch  | `runCommand*`, `git()`, `silentgit()`, alle Git-Info-Fns wandern hierher. `git.Client` kapselt `workingDir` + `GitPassthroughStderrStdout`. `startCommand` + `executeCommandsInBackgroundProcess` + `injectCommandWithMessage` bleiben vorerst in main |
| 4       | `branch/`   | Mittel       | `Branch` struct + Methoden. Bekommt `git.Client` als Abhaengigkeit |
| 5       | `squash/`   | Mittel       | Bekommt `git.Client` als Abhaengigkeit |
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

## 6. Zweiter Schritt: Package `coauthors/` extrahieren

### Warum `coauthors/` als zweites?

1. **Minimale Abhaengigkeiten**: Die einzige externe Abhaengigkeit ist `gitUserEmail()` aus `mob.go` und `say.Debug()` aus dem bereits bestehenden `say/`-Package. `gitUserEmail()` kann als Parameter injiziert werden.
2. **Eigene Test-Datei**: `coauthors_test.go` enthaelt bereits Unit-Tests fuer die Kernfunktionen (`createCommitMessage`, `sortByLength`, `removeDuplicateValues`) sowie einen Integrationstest (`TestStartDoneCoAuthors`).
3. **Klar abgegrenzter Fachbereich**: Co-Author-Tracking ist ein eigenstaendiges Feature - es parst WIP-Commit-Nachrichten und erzeugt `Co-authored-by:`-Trailer.
4. **Aehnliches Muster wie `findnext/`**: Weitgehend reiner Algorithmus mit einer klar definierten Schnittstelle nach aussen.

### Analyse der aktuellen Datei `coauthors.go`

#### Funktionen und ihre Rollen

| Funktion | Sichtbarkeit | Rolle | Abhaengigkeiten |
|----------|-------------|-------|-----------------|
| `collectCoauthorsFromWipCommits(file)` | intern | Orchestrierung: parst, filtert, dedupliziert, sortiert | `parseCoauthors`, `removeElementsContaining`, `removeDuplicateValues`, `sortByLength`, `gitUserEmail()`, `say.Debug` |
| `parseCoauthors(file)` | intern | Parst `Co-authored-by:`/`Author:`-Zeilen aus einer Datei | `stripToAuthor` |
| `stripToAuthor(line)` | intern | Extrahiert `Name <email>` aus einer Zeile | keine |
| `sortByLength(slice)` | intern | Sortiert Strings nach Laenge | keine |
| `removeElementsContaining(slice, filter)` | intern | Filtert Strings die `filter` enthalten | keine |
| `removeDuplicateValues(slice)` | intern | Entfernt Duplikate (erhaelt Reihenfolge) | keine |
| `appendCoauthorsToSquashMsg(gitDir)` | intern | Liest SQUASH_MSG, haengt Co-Author-Zeilen an | `collectCoauthorsFromWipCommits`, `createCommitMessage`, `say.Debug` |
| `createCommitMessage(coauthors)` | intern | Erzeugt den Co-Author-Block als String | keine |

#### Typ-Definition

```go
type Author = string
```

`Author` ist ein Type-Alias fuer `string` im Format `"Full Name <email>"`. Er wird im neuen Package exportiert.

#### Externe Abhaengigkeiten (die aufgeloest werden muessen)

| Abhaengigkeit | Herkunft | Aufloesung |
|---------------|----------|------------|
| `gitUserEmail()` | `mob.go:1098` - ruft `silentgit("config", "--get", "user.email")` auf | Wird als Parameter an `CollectCoauthorsFromWipCommits` uebergeben |
| `say.Debug()` | `say/`-Package (bereits eigenes Package) | Import bleibt, kein Problem |
| `gitDir()` | `mob.go:1015` - ruft `silentgit("rev-parse", "--absolute-git-dir")` auf | Bleibt in `mob.go`; `AppendCoauthorsToSquashMsg` bekommt `gitDir` bereits als Parameter |

#### Aufrufstelle in `mob.go`

Es gibt genau **eine** Aufrufstelle in `mob.go:996`:

```go
err := appendCoauthorsToSquashMsg(gitDir())
```

Diese befindet sich in der `done()`-Funktion und wird aufgerufen, nachdem der WIP-Branch in den Base-Branch gesquashed wurde. `gitDir()` wird bereits als Parameter uebergeben - das ist ideal.

### Konkrete Schritte

#### Schritt 2.1: Package erstellen

Neues Verzeichnis `coauthors/` mit Datei `coauthors.go` erstellen.

#### Schritt 2.2: Code verschieben

Aus `coauthors.go` (aktuell `package main`) in `coauthors/coauthors.go` verschieben:

```go
package coauthors

import (
	"bufio"
	"fmt"
	"github.com/remotemobprogramming/mob/v5/say"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
)

// Author repraesentiert einen Co-Author im Format "Full Name <email>"
type Author = string

// AppendCoauthorsToSquashMsg liest die SQUASH_MSG-Datei im angegebenen gitDir,
// parst die Co-Authors aus den WIP-Commits und haengt sie als Co-authored-by-Trailer an.
func AppendCoauthorsToSquashMsg(gitDir string, currentUserEmail string) error {
	squashMsgPath := path.Join(gitDir, "SQUASH_MSG")
	say.Debug("opening " + squashMsgPath)
	file, err := os.OpenFile(squashMsgPath, os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			say.Debug(squashMsgPath + " does not exist")
			return nil
		}
		return err
	}

	defer file.Close()

	coauthors := CollectCoauthorsFromWipCommits(file, currentUserEmail)

	if len(coauthors) > 0 {
		coauthorSuffix := CreateCommitMessage(coauthors)

		writer := bufio.NewWriter(file)
		writer.WriteString(coauthorSuffix)
		err = writer.Flush()
	}

	return err
}

// CollectCoauthorsFromWipCommits parst Co-Authors aus einer Datei (typischerweise SQUASH_MSG),
// filtert den aktuellen User heraus, entfernt Duplikate und sortiert nach Namenlaenge.
func CollectCoauthorsFromWipCommits(file *os.File, currentUserEmail string) []Author {
	coauthors := parseCoauthors(file)
	say.Debug("Parsed coauthors")
	say.Debug(strings.Join(coauthors, ","))

	coauthors = removeElementsContaining(coauthors, currentUserEmail)
	say.Debug("Parsed coauthors without committer")
	say.Debug(strings.Join(coauthors, ","))

	coauthors = removeDuplicateValues(coauthors)
	say.Debug("Unique coauthors without committer")
	say.Debug(strings.Join(coauthors, ","))

	sortByLength(coauthors)
	say.Debug("Sorted unique coauthors without committer")
	say.Debug(strings.Join(coauthors, ","))

	return coauthors
}

// CreateCommitMessage erzeugt den Co-authored-by-Block fuer die Commit-Nachricht.
func CreateCommitMessage(coauthors []Author) string {
	commitMessage := "\n\n"
	commitMessage += "# automatically added all co-authors from WIP commits\n"
	commitMessage += "# add missing co-authors manually\n"
	for _, coauthor := range coauthors {
		commitMessage += fmt.Sprintf("Co-authored-by: %s\n", coauthor)
	}
	return commitMessage
}

// --- Unexportierte Hilfsfunktionen (bleiben im Package) ---

func parseCoauthors(file *os.File) []Author {
	var coauthors []Author
	authorOrCoauthorMatcher := regexp.MustCompile("(?i).*(author)+.+<+.*>+")
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if authorOrCoauthorMatcher.MatchString(line) {
			author := stripToAuthor(line)
			coauthors = append(coauthors, author)
		}
	}
	return coauthors
}

func stripToAuthor(line string) Author {
	return strings.TrimSpace(strings.Join(strings.Split(line, ":")[1:], ""))
}

func sortByLength(slice []string) {
	sort.Slice(slice, func(i, j int) bool {
		return len(slice[i]) < len(slice[j])
	})
}

func removeElementsContaining(slice []string, containsFilter string) []string {
	var result []string
	for _, entry := range slice {
		if !strings.Contains(entry, containsFilter) {
			result = append(result, entry)
		}
	}
	return result
}

func removeDuplicateValues(slice []string) []string {
	var result []string
	keys := make(map[string]bool)
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			result = append(result, entry)
		}
	}
	return result
}
```

Aenderungen gegenueber dem Original:
- `package main` -> `package coauthors`
- `appendCoauthorsToSquashMsg` -> `AppendCoauthorsToSquashMsg` (exportiert)
- `collectCoauthorsFromWipCommits` -> `CollectCoauthorsFromWipCommits` (exportiert)
- `createCommitMessage` -> `CreateCommitMessage` (exportiert)
- **Neue Parameter**: `CollectCoauthorsFromWipCommits` und `AppendCoauthorsToSquashMsg` bekommen `currentUserEmail string` als Parameter statt intern `gitUserEmail()` aufzurufen
- Alle Hilfsfunktionen (`parseCoauthors`, `stripToAuthor`, `sortByLength`, `removeElementsContaining`, `removeDuplicateValues`) bleiben unexportiert

#### Schritt 2.3: Tests verschieben und anpassen

Die Unit-Tests aus `coauthors_test.go` werden in `coauthors/coauthors_test.go` verschoben:

```go
package coauthors

import "testing"

func TestCreateCommitMessage(t *testing.T) {
	// Nutzt jetzt exportierte Funktion CreateCommitMessage
	expected := "\n\n# automatically added all co-authors from WIP commits\n# add missing co-authors manually\nCo-authored-by: Alice <alice@example.com>\nCo-authored-by: Bob <bob@example.com>\n"
	actual := CreateCommitMessage([]Author{"Alice <alice@example.com>", "Bob <bob@example.com>"})
	if actual != expected {
		t.Errorf("expected %q, got %q", expected, actual)
	}
}

func TestSortByLength(t *testing.T) {
	slice := []string{"aa", "b"}
	sortByLength(slice)
	// sortByLength bleibt unexportiert, Test ist im gleichen Package moeglich
	if slice[0] != "b" || slice[1] != "aa" {
		t.Errorf("expected [b, aa], got %v", slice)
	}
}

func TestRemoveDuplicateValues(t *testing.T) {
	slice := []string{"aa", "b", "c", "b"}
	actual := removeDuplicateValues(slice)
	if len(actual) != 3 || actual[0] != "aa" || actual[1] != "b" || actual[2] != "c" {
		t.Errorf("expected [aa, b, c], got %v", actual)
	}
}
```

**Hinweis zu den Tests**: Die Unit-Tests (`TestCreateCommitMessage`, `TestSortByLength`, `TestRemoveDuplicateValues`) koennen direkt ins neue Package verschoben werden, da sie keine externen Abhaengigkeiten haben. Sie nutzen aber aktuell die Hilfsfunktion `equals()` aus `mob_test.go` - diese muss entweder:
- a) durch Standard-`testing`-Vergleiche ersetzt werden (empfohlen, da einfacher), oder
- b) als Hilfsfunktion im neuen Test-File dupliziert werden.

Der **Integrationstest** `TestStartDoneCoAuthors` bleibt in `mob_test.go` (bzw. `coauthors_test.go` im Root), da er die gesamte Session-Maschinerie (`start()`, `next()`, `done()`, `setWorkingDir()`, `createFile()`) benoetigt. Er testet das Zusammenspiel und wird nach Anpassung der Aufrufe in `mob.go` weiterhin funktionieren.

#### Schritt 2.4: Aufrufer in `mob.go` anpassen

In `mob.go` den Import hinzufuegen und die Aufrufstelle anpassen:

```go
import (
	"github.com/remotemobprogramming/mob/v5/coauthors"
)

// In done(), mob.go:996
// ALT:  err := appendCoauthorsToSquashMsg(gitDir())
// NEU:  err := coauthors.AppendCoauthorsToSquashMsg(gitDir(), gitUserEmail())
```

Die Funktion `gitUserEmail()` bleibt in `mob.go` (sie wird spaeter ins `git/`-Package wandern).

#### Schritt 2.5: Integrationstest anpassen

`TestStartDoneCoAuthors` bleibt im Root-Package (da er `start()`, `next()`, `done()` benoetigt). Folgende Aenderungen sind noetig:
- Keine Code-Aenderungen am Test selbst, da er `appendCoauthorsToSquashMsg` nicht direkt aufruft, sondern indirekt ueber `done()`. Die Aenderung in `done()` (Schritt 2.4) sorgt dafuer, dass der Test automatisch das neue Package nutzt.
- Die Test-Datei im Root bleibt als `coauthors_test.go` bestehen, wird aber nur noch den Integrationstest enthalten:

```go
package main

import (
	"path/filepath"
	"testing"
)

func TestStartDoneCoAuthors(t *testing.T) {
	// ... unveraendert, da der Test indirekt ueber done() laeuft ...
}
```

Die Unit-Tests (`TestCreateCommitMessage`, `TestSortByLength`, `TestRemoveDuplicateValues`) werden aus der Root-Datei entfernt, da sie ins neue Package gewandert sind.

#### Schritt 2.6: Alte Datei bereinigen

`coauthors.go` im Root-Verzeichnis wird geloescht. `coauthors_test.go` im Root-Verzeichnis wird auf den verbleibenden Integrationstest reduziert.

#### Schritt 2.7: Tests ausfuehren

```bash
go test ./...
```

Alle Tests muessen gruen sein, bevor der Schritt als abgeschlossen gilt. Insbesondere:
- `go test ./coauthors/` - Unit-Tests im neuen Package
- `go test .` - Integrationstest `TestStartDoneCoAuthors` im Root-Package

### Erwartetes Ergebnis nach Schritt 2

```
mob/
├── mob.go                    # Import von coauthors, Aufruf angepasst (1 Zeile geaendert)
├── coauthors_test.go         # NUR NOCH Integrationstest TestStartDoneCoAuthors
├── coauthors.go              # GELOESCHT
├── coauthors/                # NEU
│   ├── coauthors.go          # AppendCoauthorsToSquashMsg(), CollectCoauthorsFromWipCommits(),
│   │                         # CreateCommitMessage() + unexportierte Hilfsfunktionen
│   └── coauthors_test.go     # Unit-Tests (CreateCommitMessage, sortByLength, removeDuplicateValues)
├── findnext/                 # Bereits extrahiert (Schritt 1)
│   ├── findnext.go
│   └── findnext_test.go
└── ... (Rest unveraendert)
```

### Signatur-Aenderungen im Ueberblick

| Funktion (alt) | Funktion (neu) | Aenderung |
|----------------|---------------|-----------|
| `appendCoauthorsToSquashMsg(gitDir string) error` | `coauthors.AppendCoauthorsToSquashMsg(gitDir string, currentUserEmail string) error` | Exportiert + neuer Parameter `currentUserEmail` |
| `collectCoauthorsFromWipCommits(file *os.File) []Author` | `coauthors.CollectCoauthorsFromWipCommits(file *os.File, currentUserEmail string) []Author` | Exportiert + neuer Parameter `currentUserEmail` |
| `createCommitMessage(coauthors []Author) string` | `coauthors.CreateCommitMessage(coauthors []Author) string` | Exportiert |
| `Author = string` (Typ-Alias) | `coauthors.Author = string` | Exportiert |

### Risikobewertung

- **Risiko**: Gering
- **Abwaertskompatibilitaet**: Keine oeffentliche API betroffen (alles intern)
- **Testabdeckung**: Bestehende Unit-Tests decken Kernlogik ab, Integrationstest deckt Zusammenspiel ab
- **Einzige Stolperfalle**: Der Integrationstest `TestStartDoneCoAuthors` muss weiterhin im Root-Package laufen, da er `start()`, `next()`, `done()` etc. benoetigt. Das ist kein Problem, solange die Root-`coauthors_test.go` korrekt aufgeraeumt wird.
- **Rollback**: Einfach rueckgaengig zu machen (eine Datei wiederherstellen, neues Verzeichnis loeschen)

---

## 8. Dritter Schritt: Package `git/` extrahieren

### Warum `git/` als drittes?

1. **Fundamentale Infrastruktur**: Alle verbleibenden Extraktionen (`branch/`, `squash/`, `timer/`, `session/`) haengen von Git-Funktionen ab. Ohne `git/` als eigenstaendiges Package kann keiner dieser Schritte sauber umgesetzt werden.
2. **Eliminiert globalen Zustand**: Die globalen Variablen `workingDir` und `GitPassthroughStderrStdout` werden in einem `git.Client`-Struct gekapselt - der wichtigste Schritt zur Entflechtung des Codes.
3. **Klare Schichtentrennung**: Trennt die "wie fuehre ich Git-Kommandos aus?"-Infrastruktur von der "was mache ich mit Git?"-Geschaeftslogik.
4. **Exit-Handling wird testbar**: `Exit` wird von einer globalen Variable zum injizierbaren Feld auf `git.Client`, was die Testbarkeit verbessert.

### Analyse der aktuellen Funktionen

#### Kommando-Ausfuehrung (Schicht 1: Basis)

| Funktion | Zeile | Zeilen | Beschreibung |
|----------|-------|--------|--------------|
| `runCommandSilent(name, args...)` | 1245-1256 | 12 | Fuehrt Kommando aus, gibt stdout+stderr als String zurueck |
| `runCommand(name, args...)` | 1258-1300 | 42 | Fuehrt Kommando aus, streamt Output an Konsole |

#### Git-Wrapper (Schicht 2: Komfort-Layer)

| Funktion | Zeile | Zeilen | Beschreibung |
|----------|-------|--------|--------------|
| `git(args...)` | 1176-1200 | 25 | Hauptfunktion: Fuehrt Git-Kommando aus, bei Fehler `Exit(1)` |
| `silentgit(args...)` | 1136-1150 | 15 | Wie `git()`, aber silent, gibt Output als String zurueck |
| `silentgitignorefailure(args...)` | 1152-1159 | 8 | Wie `silentgit()`, aber gibt Error statt Exit zurueck |
| `gitWithoutEmptyStrings(args...)` | 1171-1174 | 4 | Wie `git()`, filtert leere Strings aus Args |
| `gitIgnoreFailure(args...)` | 1202-1224 | 23 | Wie `git()`, aber Warning statt Exit bei Fehler |
| `gitHooksOption(c)` | 940-946 | 7 | Gibt `"--no-verify"` oder `""` zurueck je nach Config |

#### Git-Info-Funktionen (Schicht 3: Abfragen)

| Funktion | Zeile | Zeilen | Rueckgabe | Beschreibung |
|----------|-------|--------|-----------|--------------|
| `gitCurrentBranch()` | 1081-1084 | 4 | `Branch` | Gibt aktuellen Branch zurueck (nutzt `rev-parse`) |
| `gitBranches()` | 1073-1075 | 3 | `[]string` | Alle lokalen Branches |
| `gitRemoteBranches()` | 1077-1079 | 3 | `[]string` | Alle Remote-Branches |
| `gitUserName()` | 1094-1097 | 4 | `string` | Git config user.name |
| `gitUserEmail()` | 1099-1101 | 3 | `string` | Git config user.email |
| `isGit()` | 1240-1243 | 4 | `bool` | Prueft ob aktuelles Verzeichnis ein Git-Repo ist |
| `gitRootDir()` | 1020-1022 | 3 | `string` | Git-Root-Verzeichnis |
| `gitDir()` | 1016-1018 | 3 | `string` | Absoluter Pfad zum `.git`-Verzeichnis |
| `hasCommits()` | 302-305 | 4 | `bool` | Prueft ob das Repo Commits hat |
| `doBranchesDiverge(a, b)` | 1086-1092 | 7 | `bool` | Pruefen ob zwei Branches divergieren |
| `gitVersion()` | 1231-1238 | 8 | `string` | Git-Versionsstring |
| `isNothingToCommit()` | 1057-1060 | 4 | `bool` | Prueft ob Working Tree clean ist |
| `hasUncommittedChanges()` | 1062-1064 | 3 | `bool` | Negation von `isNothingToCommit()` |

#### Typen und Hilfsfunktionen

| Element | Zeile | Zeilen | Beschreibung |
|---------|-------|--------|--------------|
| `GitVersion` struct | 49-53 | 5 | Major/Minor/Patch Versionsfelder |
| `parseGitVersion(string)` | 55-83 | 29 | Parst Git-Versionsstring |
| `GitVersion.Less(rhs)` | 85-89 | 5 | Versionsvergleich |
| `deleteEmptyStrings(s)` | 1161-1169 | 9 | Filtert leere Strings (fuer `gitWithoutEmptyStrings`) |

#### Globale Variablen die gekapselt werden

| Variable | Zeile | Aktuell | Ziel |
|----------|-------|---------|------|
| `workingDir` | 32 | `var workingDir = ""` | `Client.WorkingDir` |
| `GitPassthroughStderrStdout` | 34 | `var GitPassthroughStderrStdout = false` | `Client.PassthroughStderrStdout` |
| `Exit` | 1313-1315 | `var Exit = func(code int) { os.Exit(code) }` | `Client.Exit` |

#### Was NICHT in `git/` wandert

| Element | Grund |
|---------|-------|
| `Branch` struct + Methoden | Domaenen-Modell, geht spaeter nach `branch/` (Schritt 4) |
| `startCommand()` | Startet Nicht-Git-Prozesse (Timer, IDE) |
| `executeCommandsInBackgroundProcess()` | Hintergrundprozesse fuer Timer/Voice |
| `injectCommandWithMessage()` | Kommando-Injection fuer Timer/IDE |
| `stringContains()` | Von Branch-Methoden genutzt, geht mit Branch nach `branch/` |
| `addSuffix()`, `removePrefix()` | Von Branch-Methoden genutzt, gehen mit Branch nach `branch/` |
| `ReverseSlice()` | Allgemeine Utility, nicht git-spezifisch |

### Uebergangsstrategie: Duenne Wrapper in `main`

**Kernentscheidung**: Um den Diff minimal und die Aenderung sicher zu halten, bleiben in `main` *duenne Wrapper-Funktionen* bestehen, die an `git.Client` delegieren.

**Warum Wrapper statt sofortiger Umbenennung?**

1. **89 Aufrufe** von `git()` und `silentgit()` allein in `mob_test.go` - eine Massen-Umbenennung waere fehleranfaellig und schwer zu reviewen.
2. **Branch-Methoden** (`hasRemoteBranch()`, `hasLocalBranch()`, `hasLocalCommits()` etc.) rufen `silentgit()` und `gitBranches()` direkt auf. Diese muessen nicht geaendert werden, wenn die Wrapper in main bleiben.
3. **squash_wip.go** ruft `git()`, `silentgit()`, `gitHooksOption()` auf - aendert sich nicht.
4. **Jeder nachfolgende Extraktionsschritt** (branch/, squash/, timer/) entfernt natuerlich die Wrapper, da die extrahierten Packages `git/` direkt importieren.
5. Am Ende von Schritt 7 (session/) sind alle Wrapper verschwunden und koennen geloescht werden.

**Ausnahme `gitCurrentBranch()`**: Diese Funktion gibt aktuell `Branch` zurueck. Da `Branch` in main bleibt und `git/` nicht von main importieren kann (zirkulaere Abhaengigkeit), gibt die `git.Client`-Methode `CurrentBranch()` einen `string` zurueck. Der Wrapper in main wickelt das in `newBranch()` ein:

```go
// In git/ package
func (g *Client) CurrentBranch() string {
    return g.Silent("rev-parse", "--abbrev-ref", "HEAD")
}

// In main package (duenner Wrapper)
func gitCurrentBranch() Branch {
    return newBranch(gitClient.CurrentBranch())
}
```

### Design des `git.Client`

```go
package git

import (
    "bufio"
    config "github.com/remotemobprogramming/mob/v5/configuration"
    "github.com/remotemobprogramming/mob/v5/say"
    "os/exec"
    "regexp"
    "strconv"
    "strings"
)

// Client kapselt den Zustand fuer Git-Operationen.
// Ersetzt die globalen Variablen workingDir, GitPassthroughStderrStdout und Exit.
type Client struct {
    WorkingDir              string
    PassthroughStderrStdout bool
    Exit                    func(int)
}

// --- Schicht 1: Rohe Kommando-Ausfuehrung ---

func (g *Client) runCommandSilent(name string, args ...string) (string, string, error) { ... }
func (g *Client) runCommand(name string, args ...string) (string, string, error) { ... }

// --- Schicht 2: Git-Wrapper ---

func (g *Client) Run(args ...string)                       { ... }  // vorher: git()
func (g *Client) Silent(args ...string) string             { ... }  // vorher: silentgit()
func (g *Client) SilentIgnoreFailure(args ...string) (string, error) { ... }
func (g *Client) RunWithoutEmptyStrings(args ...string)    { ... }  // vorher: gitWithoutEmptyStrings()
func (g *Client) RunIgnoreFailure(args ...string) error    { ... }  // vorher: gitIgnoreFailure()
func HooksOption(c config.Configuration) string             { ... }  // vorher: gitHooksOption()

// --- Schicht 3: Git-Info-Abfragen ---

func (g *Client) CurrentBranch() string                    { ... }  // gibt string zurueck, nicht Branch
func (g *Client) Branches() []string                       { ... }
func (g *Client) RemoteBranches() []string                 { ... }
func (g *Client) UserName() string                         { ... }
func (g *Client) UserEmail() string                        { ... }
func (g *Client) IsRepo() bool                             { ... }  // vorher: isGit()
func (g *Client) RootDir() string                          { ... }
func (g *Client) Dir() string                              { ... }  // vorher: gitDir()
func (g *Client) HasCommits() bool                         { ... }
func (g *Client) DoBranchesDiverge(a, b string) bool       { ... }
func (g *Client) Version() string                          { ... }  // vorher: gitVersion()
func (g *Client) IsNothingToCommit() bool                  { ... }
func (g *Client) HasUncommittedChanges() bool              { ... }
```

**Hinweis zu `HooksOption`**: Diese Funktion ist eine *reine Funktion* (keine Seiteneffekte, kein Client-Zugriff). Sie wird daher als Package-Level-Funktion exportiert, nicht als Methode auf Client.

### Umgang mit `Exit`

`Exit` wird ein Feld auf `git.Client`:

```go
type Client struct {
    // ...
    Exit func(int)
}
```

**In Produktion** wird `Exit` mit `os.Exit` initialisiert:

```go
gitClient = &git.Client{
    Exit: func(code int) { os.Exit(code) },
}
```

**In Tests** kann `Exit` ueberschrieben werden, genau wie bisher:

```go
// vorher: Exit = func(code int) { panic(code) }
// nachher: gitClient.Exit = func(code int) { panic(code) }
```

Die bestehenden Hilfsfunktionen `mockExit()` und `resetExit()` in `mob_test.go` werden minimal angepasst:

```go
func mockExit() {
    originalExitFunction = gitClient.Exit
    gitClient.Exit = func(code int) {
        defer func() {
            if r := recover(); r != nil {
                fmt.Printf("exit(%d)\n", code)
            }
        }()
        panic(code)
    }
}

func resetExit() {
    gitClient.Exit = originalExitFunction
}
```

### Umgang mit Tests

Die **Test-Aenderungen sind minimal** dank der Wrapper-Strategie:

1. **`setWorkingDir(dir)`** wird angepasst, um `gitClient.WorkingDir` zu setzen:
   ```go
   func setWorkingDir(dir string) {
       gitClient.WorkingDir = dir
       say.Say("\n===== cd " + dir)
   }
   ```

2. **`createTestbed()`** setzt `gitClient.WorkingDir = ""` statt `workingDir = ""`.

3. **`mockExit()` / `resetExit()`** verwenden `gitClient.Exit` statt der globalen `Exit`-Variable.

4. **Alle 76 `git(...)`-Aufrufe und 13 `silentgit(...)`-Aufrufe** in Tests bleiben unveraendert, da sie die Wrapper in main nutzen.

### Konkrete Schritte

#### Schritt 3.1: Package erstellen

Neues Verzeichnis `git/` mit Datei `git.go` erstellen.

#### Schritt 3.2: Client-Struct und Kommando-Ausfuehrung verschieben

`runCommand()` und `runCommandSilent()` aus `mob.go` in `git/git.go` verschieben und als unexportierte Methoden auf `Client` implementieren. Die Methoden verwenden `g.WorkingDir` statt der globalen `workingDir`-Variable.

```go
func (g *Client) runCommandSilent(name string, args ...string) (string, string, error) {
    command := exec.Command(name, args...)
    if len(g.WorkingDir) > 0 {
        command.Dir = g.WorkingDir
    }
    // ... Rest wie bisher
}
```

#### Schritt 3.3: Git-Wrapper-Methoden verschieben

`git()`, `silentgit()`, `silentgitignorefailure()`, `gitWithoutEmptyStrings()`, `gitIgnoreFailure()` als Methoden auf `Client` implementieren. Sie verwenden `g.PassthroughStderrStdout` statt der globalen Variable und `g.Exit` statt der globalen `Exit`-Variable.

```go
func (g *Client) Run(args ...string) {
    say.Indented("git " + strings.Join(args, " "))
    commandString, output, err := "", "", error(nil)
    if g.PassthroughStderrStdout {
        commandString, output, err = g.runCommand("git", args...)
    } else {
        commandString, output, err = g.runCommandSilent("git", args...)
    }
    if err != nil {
        if !g.IsRepo() {
            say.Error("expecting the current working directory to be a git repository.")
        } else {
            // ... Fehlerbehandlung wie bisher
        }
        g.Exit(1)
    }
}
```

#### Schritt 3.4: Git-Info-Methoden verschieben

Alle Git-Info-Funktionen als Methoden auf `Client` implementieren. `CurrentBranch()` gibt `string` statt `Branch` zurueck (da `Branch` in main bleibt):

```go
func (g *Client) CurrentBranch() string {
    return g.Silent("rev-parse", "--abbrev-ref", "HEAD")
}

func (g *Client) Branches() []string {
    return strings.Split(g.Silent("branch", "--format=%(refname:short)"), "\n")
}

func (g *Client) IsNothingToCommit() bool {
    output := g.Silent("status", "--porcelain")
    return len(output) == 0
}

func (g *Client) HasUncommittedChanges() bool {
    return !g.IsNothingToCommit()
}
```

#### Schritt 3.5: GitVersion-Struct und gitHooksOption verschieben

`GitVersion` struct, `parseGitVersion()` und `Less()` in `git/git.go` verschieben. Exportierung:
- `GitVersion` -> bleibt exportiert
- `parseGitVersion` -> `ParseVersion` (exportiert, da von main genutzt)
- `Less()` -> bleibt exportiert
- `gitHooksOption()` -> `HooksOption()` (Package-Level-Funktion)

#### Schritt 3.6: Hilfsfunktion verschieben

`deleteEmptyStrings()` wandert als unexportierte Funktion nach `git/git.go`, da sie ausschliesslich von `RunWithoutEmptyStrings()` verwendet wird.

#### Schritt 3.7: Package-Level-Variable und Wrapper in main erstellen

In `mob.go` eine Package-Level-Variable `gitClient` anlegen und duenne Wrapper erstellen:

```go
// Package-Level Git-Client (ersetzt globale Variablen workingDir, GitPassthroughStderrStdout, Exit)
var gitClient = &git.Client{
    Exit: func(code int) { os.Exit(code) },
}

// --- Duenne Wrapper (werden in Schritten 4-7 schrittweise entfernt) ---

func git(args ...string)                         { gitClient.Run(args...) }
func silentgit(args ...string) string            { return gitClient.Silent(args...) }
func silentgitignorefailure(args ...string) (string, error) { return gitClient.SilentIgnoreFailure(args...) }
func gitWithoutEmptyStrings(args ...string)      { gitClient.RunWithoutEmptyStrings(args...) }
func gitIgnoreFailure(args ...string) error      { return gitClient.RunIgnoreFailure(args...) }
func gitHooksOption(c config.Configuration) string { return git.HooksOption(c) }

func gitCurrentBranch() Branch                   { return newBranch(gitClient.CurrentBranch()) }
func gitBranches() []string                      { return gitClient.Branches() }
func gitRemoteBranches() []string                { return gitClient.RemoteBranches() }
func gitUserName() string                        { return gitClient.UserName() }
func gitUserEmail() string                       { return gitClient.UserEmail() }
func isGit() bool                                { return gitClient.IsRepo() }
func gitRootDir() string                         { return gitClient.RootDir() }
func gitDir() string                             { return gitClient.Dir() }
func hasCommits() bool                           { return gitClient.HasCommits() }
func doBranchesDiverge(a, b string) bool         { return gitClient.DoBranchesDiverge(a, b) }
func gitVersion() string                         { return gitClient.Version() }
func isNothingToCommit() bool                    { return gitClient.IsNothingToCommit() }
func hasUncommittedChanges() bool                { return gitClient.HasUncommittedChanges() }
```

In der `run()`-Funktion in `mob.go` wird die Initialisierung von `GitPassthroughStderrStdout` angepasst:

```go
// vorher: GitPassthroughStderrStdout = true
// nachher: gitClient.PassthroughStderrStdout = true
```

#### Schritt 3.8: Tests anpassen

Minimale Aenderungen in `mob_test.go`:

1. `setWorkingDir()`: `workingDir = dir` -> `gitClient.WorkingDir = dir`
2. `createTestbed()`: `workingDir = ""` -> `gitClient.WorkingDir = ""`
3. `createTestbedIn()`: `workingDir = localDirectory` -> `gitClient.WorkingDir = localDirectory`
4. `mockExit()`: `Exit = func(...)` -> `gitClient.Exit = func(...)`
5. `resetExit()`: `Exit = originalExitFunction` -> `gitClient.Exit = originalExitFunction`
6. `originalExitFunction`-Variable: Typ aendern auf `func(int)`, Initialisierung anpassen

Alle anderen Test-Aufrufe (`git(...)`, `silentgit(...)`, etc.) bleiben unveraendert.

#### Schritt 3.9: Alte Funktionen und Variablen loeschen

Aus `mob.go` entfernen:
- Die globalen Variablen `workingDir`, `GitPassthroughStderrStdout` (durch `gitClient`-Felder ersetzt)
- Die globale Variable `Exit` (durch `gitClient.Exit` ersetzt)
- Die Implementierungen von `runCommand()`, `runCommandSilent()` (jetzt in git/)
- Die Implementierung von `deleteEmptyStrings()` (jetzt in git/)
- `GitVersion` struct, `parseGitVersion()`, `Less()` (jetzt in git/)
- Die Variable `args` bleibt als lokale Variable in `run()` (wird nur fuer CLI-Parsing genutzt)

**Hinweis**: Die duennen Wrapper-Funktionen (Schritt 3.7) bleiben in main - sie ersetzen die alten Implementierungen.

#### Schritt 3.10: Tests ausfuehren

```bash
go test ./...
```

Alle Tests muessen gruen sein. Insbesondere:
- `go test ./git/` - Falls Unit-Tests fuer git/ angelegt werden
- `go test .` - Alle bestehenden Tests im Root-Package (nutzen die Wrapper)

### Erwartetes Ergebnis nach Schritt 3

```
mob/
├── mob.go                    # gitClient Variable, duenne Wrapper, kein globaler Zustand mehr
│                             # (~20 Zeilen Wrapper + gitClient-Initialisierung ersetzen ~230 Zeilen Implementierung)
├── git/                      # NEU
│   └── git.go                # Client struct, Run(), Silent(), alle Git-Info-Methoden,
│                             # GitVersion, HooksOption(), ~230 Zeilen
├── squash_wip.go             # Unveraendert (nutzt Wrapper in main)
├── timer.go                  # Unveraendert (nutzt Wrapper in main)
├── status.go                 # Unveraendert (nutzt Wrapper in main)
├── coauthors/                # Bereits extrahiert (Schritt 2)
├── findnext/                 # Bereits extrahiert (Schritt 1)
└── ... (Rest unveraendert)
```

### Signatur-Aenderungen im Ueberblick

| Funktion (alt, in main) | Methode (neu, in git/) | Aenderung |
|--------------------------|------------------------|-----------|
| `git(args...)` | `Client.Run(args...)` | Methode auf Client |
| `silentgit(args...)` | `Client.Silent(args...)` | Methode auf Client |
| `silentgitignorefailure(args...)` | `Client.SilentIgnoreFailure(args...)` | Methode auf Client |
| `gitWithoutEmptyStrings(args...)` | `Client.RunWithoutEmptyStrings(args...)` | Methode auf Client |
| `gitIgnoreFailure(args...)` | `Client.RunIgnoreFailure(args...)` | Methode auf Client |
| `gitHooksOption(c)` | `HooksOption(c)` | Package-Level-Funktion |
| `gitCurrentBranch() Branch` | `Client.CurrentBranch() string` | Rueckgabe `string` statt `Branch` |
| `gitBranches()` | `Client.Branches()` | Methode auf Client |
| `gitRemoteBranches()` | `Client.RemoteBranches()` | Methode auf Client |
| `gitUserName()` | `Client.UserName()` | Methode auf Client |
| `gitUserEmail()` | `Client.UserEmail()` | Methode auf Client |
| `isGit()` | `Client.IsRepo()` | Methode auf Client, umbenannt |
| `gitRootDir()` | `Client.RootDir()` | Methode auf Client |
| `gitDir()` | `Client.Dir()` | Methode auf Client |
| `hasCommits()` | `Client.HasCommits()` | Methode auf Client |
| `doBranchesDiverge(a, b)` | `Client.DoBranchesDiverge(a, b)` | Methode auf Client |
| `gitVersion()` | `Client.Version()` | Methode auf Client |
| `isNothingToCommit()` | `Client.IsNothingToCommit()` | Methode auf Client |
| `hasUncommittedChanges()` | `Client.HasUncommittedChanges()` | Methode auf Client |
| `var workingDir` | `Client.WorkingDir` | Feld auf Client |
| `var GitPassthroughStderrStdout` | `Client.PassthroughStderrStdout` | Feld auf Client |
| `var Exit` | `Client.Exit` | Feld auf Client |
| `GitVersion` struct | `GitVersion` struct | Unveraendert, nur Package gewechselt |
| `parseGitVersion()` | `ParseVersion()` | Exportiert |

### Auswirkungen auf nachfolgende Schritte

Nach der `git/`-Extraktion aendern sich die Abhaengigkeiten fuer die Folge-Schritte:

| Schritt | Package | Abhaengigkeit von git/ | Wrapper-Entfernung |
|---------|---------|----------------------|-------------------|
| 4 | `branch/` | `Branch`-Methoden importieren `git/` direkt, erhalten `*git.Client` als Feld oder Parameter | `gitCurrentBranch()`, `gitBranches()`, `gitRemoteBranches()`, `stringContains()` Wrapper werden entfernt |
| 5 | `squash/` | Importiert `git/` direkt | `git()`, `silentgit()`, `gitHooksOption()` Wrapper in squash-relevanten Aufrufen werden entfernt |
| 6 | `timer/` | Importiert `git/` direkt | `isGit()`, `gitCurrentBranch()`, `gitBranches()`, `gitUserName()` Wrapper werden entfernt |
| 7 | `session/` | Importiert `git/` direkt | Alle verbleibenden Wrapper werden entfernt |

### Risikobewertung

- **Risiko**: Mittel (groesster Schritt bisher, aber mechanische Aenderung)
- **Abwaertskompatibilitaet**: Keine oeffentliche API betroffen (alles intern)
- **Testabdeckung**: Alle bestehenden Tests bleiben funktional durch Wrapper-Strategie
- **Groesse**: ~230 Zeilen wandern in git/, ~30 Zeilen Wrapper + Variable in main, ~10 Zeilen Test-Anpassung
- **Hauptrisiko**: Vergessene Stellen wo `workingDir` direkt gelesen wird (statt ueber `gitClient.WorkingDir`)
- **Mitigation**: `grep -r "workingDir" .` nach der Aenderung - darf nur noch in Wrapper/setWorkingDir vorkommen
- **Rollback**: Maessig aufwendig (eine neue Datei loeschen, alte Funktionen wiederherstellen), aber durch Git trivial

---

## 7. Prinzipien fuer die gesamte Umstrukturierung

1. **Bottom-Up**: Immer zuerst die Teile extrahieren, die keine Abhaengigkeiten nach "oben" haben
2. **Ein Package pro Schritt**: Jeder Schritt ist ein eigener, testbarer Commit
3. **Tests zuerst gruen**: Vor und nach jedem Schritt muessen alle Tests bestehen
4. **Abhaengigkeiten als Parameter**: Statt globale Funktionen aufzurufen, Abhaengigkeiten explizit uebergeben
5. **Kein Verhalten aendern**: Reine Struktur-Aenderung, keine funktionalen Aenderungen
6. **Globalen Zustand schrittweise eliminieren**: `workingDir` und andere Globals werden spaeter durch ein `git.Client`-Objekt ersetzt
