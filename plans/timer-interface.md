# Plan: Timer Interface Refactoring

## Ziel

Die bestehende Timer-Implementierung in `mob` soll hinter einem klar definierten Go-Interface abstrahiert werden. Dadurch wird es möglich, neben der aktuellen lokalen Timer-Implementierung (Shell-Befehle im Hintergrund) zukünftig weitere Implementierungen (z. B. ein UI-basierter Timer, ein HTTP-basierter Timer, etc.) hinzuzufügen, ohne den bestehenden Code zu verändern.

## Ist-Zustand

### Betroffene Dateien

| Datei | Relevanz |
|---|---|
| `timer.go` | Kernlogik: `startTimer()`, `startBreakTimer()`, lokale + remote Timer-Logik, Hilfsfunktionen |
| `mob.go:484-501` | `executeCommandsInBackgroundProcess()` – führt Shell-Befehle im Hintergrund aus |
| `mob.go:385-399` | `openTimerInBrowser()` – öffnet Remote-Timer im Browser (wandert ins `timer`-Package, aber nicht Teil des Interfaces) |
| `mob.go:303-365` | Command-Routing: `case "s","start"`, `case "t","timer"`, `case "break"` rufen `StartTimer`/`StartBreakTimer` auf |
| `mob.go:460-481` | `injectCommandWithMessage()` – Hilfs-Funktion für Command-Templates |
| `mob.go:503-505` | `currentTime()` – Hilfsfunktion |
| `mob.go:1134-1309` | Git-Funktionen: `silentgit()`, `git()`, `runCommandSilent()`, `runCommand()`, `startCommand()` etc. |
| `mob.go:29-32` | Globale Variablen: `workingDir`, `GitPassthroughStderrStdout` |
| `configuration/configuration.go:177-219` | Default-Konfiguration: `VoiceCommand`, `NotifyCommand`, `TimerLocal`, `TimerUrl`, `TimerRoom`, etc. |
| `timer_test.go` | Tests für Timer-Funktionen |

### Aktuelles Verhalten

`startTimer()` und `startBreakTimer()` in `timer.go` machen intern **beides**:

1. **Remote Timer** (wenn `TimerRoom` gesetzt): HTTP PUT an `timer.mob.sh`
2. **Lokaler Timer** (wenn `TimerLocal=true`): Hintergrund-Prozess mit `sleep` + `say` + `notify-send`

Die beiden Funktionen sind nahezu identisch – sie unterscheiden sich nur in:
- der Nachricht (`"mob next"` vs. `"mob start"`)
- dem HTTP-Endpunkt (`timer` vs. `breaktimer` im JSON-Body)
- der User-Feedback-Nachricht

## Soll-Zustand: Interface Design

### Vorgeschlagenes Interface

```go
// timer/timer.go (neues Package)

// TimerType unterscheidet zwischen normalem Timer und Break-Timer
type TimerType int

const (
    TimerTypeNormal TimerType = iota
    TimerTypeBreak
)

// Timer definiert das Interface für Timer-Implementierungen.
// Die Configuration wird übergeben, damit jede Implementierung auf die
// für sie relevanten Felder zugreifen kann (z.B. TimerRoom, VoiceCommand, etc.)
type Timer interface {
    Start(durationMinutes int, timerType TimerType, configuration config.Configuration) error
}
```

### Warum dieses Design?

- **Ein Interface, eine Methode**: `Start()` ist die einzige Operation, die beide Timer-Typen (lokal und remote) gemeinsam haben. Ein minimales Interface ist leichter zu implementieren.
- **`TimerType` als Parameter**: Statt zwei Methoden (`StartTimer`/`StartBreakTimer`) wird der Typ als Parameter übergeben. Das vermeidet die aktuelle Code-Duplizierung und hält das Interface schlank.
- **`Configuration` als Parameter**: Die gesamte Configuration wird übergeben statt einzelner Felder. So hat jede Implementierung Zugriff auf alle relevanten Config-Werte (der `RemoteTimer` braucht z.B. `TimerUrl`, `TimerInsecure`; der `LocalTimer` braucht `VoiceCommand`, `VoiceMessage`, `NotifyCommand`, `NotifyMessage`). Neue Implementierungen können auf weitere Config-Felder zugreifen, ohne dass das Interface angepasst werden muss.
- **Kein `Stop()`**: Der aktuelle Timer hat keine Stop-Funktionalität (der Hintergrund-Prozess läuft einfach aus). Falls zukünftig benötigt, kann das Interface erweitert werden.
- **`openTimerInBrowser()` ist NICHT Teil des Interfaces**: Die Funktion wandert ins `timer`-Package (gehört thematisch dazu), wird aber nicht im Interface abgebildet, da sie nur für den Remote-Timer relevant ist.
- **`moo()` wandert ins `timer`-Package**: Auch `moo()` ist Timer-Funktionalität (Voice-Ausgabe). Sie wandert als exportierte Funktion `Moo()` ins `timer`-Package, ist aber ebenfalls nicht Teil des Interfaces.

### Implementierungen

#### 1. `LocalTimer` (bestehende Logik)

```go
// timer/local.go

type LocalTimer struct{}

func (t *LocalTimer) Start(durationMinutes int, timerType TimerType, configuration config.Configuration) error {
    timeoutInSeconds := durationMinutes * 60

    message := configuration.VoiceMessage   // "mob next"
    if timerType == TimerTypeBreak {
        message = "mob start"
    }

    return executeCommandsInBackgroundProcess(
        getSleepCommand(timeoutInSeconds),
        getVoiceCommand(message, configuration.VoiceCommand),
        getNotifyCommand(message, configuration.NotifyCommand),
        "echo \"mobTimer\"",
    )
}
```

Der `LocalTimer` benötigt keine eigenen Felder – alles kommt aus der `Configuration`.

#### 2. `RemoteTimer` (bestehende Logik)

```go
// timer/remote.go

type RemoteTimer struct {
    Room string   // vorab aufgelöst in main (via getMobTimerRoom)
    User string   // vorab aufgelöst in main (via getUserForMobTimer)
}

func (t *RemoteTimer) Start(durationMinutes int, timerType TimerType, configuration config.Configuration) error {
    // JSON-Body je nach TimerType: "timer" oder "breaktimer"
    timerKey := "timer"
    if timerType == TimerTypeBreak {
        timerKey = "breaktimer"
    }

    putBody, _ := json.Marshal(map[string]interface{}{
        timerKey: durationMinutes,
        "user":   t.User,
    })
    client := httpclient.CreateHttpClient(configuration.TimerInsecure)
    _, err := client.SendRequest(putBody, "PUT", configuration.TimerUrl+t.Room)
    return err
}
```

Der `RemoteTimer` bekommt `Room` und `User` als Struct-Felder. Diese werden **in `main` aufgelöst** (vor dem Timer-Start), da die Auflösung Git-Funktionen und Mob-Domain-Logik (`determineBranches`, `Branch.IsWipBranch`) benötigt. So muss das `timer`-Package weder `Branch` noch das `git`-Package kennen.

### Nutzung in `timer.go` (nach Refactoring)

```go
func startTimer(timerInMinutes string, configuration config.Configuration) error {
    err, timeoutInMinutes := toMinutes(timerInMinutes)
    if err != nil {
        return err
    }

    timers := buildTimers(configuration)
    if len(timers) == 0 {
        say.Error("No timer configured, not starting timer")
        return errors.New("no timer configured")
    }

    for _, t := range timers {
        if err := t.Start(timeoutInMinutes, timer.TimerTypeNormal, configuration); err != nil {
            return err
        }
    }

    timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")
    say.Info("It's now " + currentTime() + ". " + fmt.Sprintf("%d min timer ends at approx. %s", timeoutInMinutes, timeOfTimeout) + ". Happy collaborating! :)")
    return nil
}
```

Die Funktion `buildTimers(configuration)` erstellt basierend auf der Konfiguration die passenden `Timer`-Implementierungen (0..n). `startBreakTimer` nutzt dieselbe Logik mit `timer.TimerTypeBreak`.

### Entschiedene Design-Fragen

1. **Neues Package `timer/`** – Ja, das Interface und die Implementierungen kommen in ein eigenes `timer/`-Package.
2. **`openTimerInBrowser()` wandert ins `timer`-Package** – Gehört thematisch zum Timer, wird aber nicht im Interface abgebildet. Wird als exportierte Funktion `OpenTimerInBrowser()` bereitgestellt.
3. **`moo()` wandert ins `timer`-Package** – Nutzt `executeCommandsInBackgroundProcess` und `getVoiceCommand`, die beide im `timer`-Package leben. Wird als exportierte Funktion `Moo()` bereitgestellt.
4. **`executeCommandsInBackgroundProcess()` wird im `timer`-Package neu implementiert** (ohne `workingDir`/`startCommand`) – Die Original-Funktion in `mob.go` nutzt `startCommand()`, das an ein globales `workingDir` gebunden ist (relevant für Git-Befehle). Der lokale Timer braucht kein `workingDir` – daher bekommt das `timer`-Package eine eigene, schlanke Version, die direkt `exec.Command` nutzt. Das Original in `mob.go` bleibt für `openLastModifiedFileOfLastCommit()` etc. bestehen.
5. **`getMobTimerRoom()` bleibt in `main`** – Die Funktion nutzt `isGit()`, `gitCurrentBranch()`, `determineBranches()` und `Branch.IsWipBranch()` – alles Mob-Domain-Logik, die Git-Zugriff benötigt. Der aufgelöste Room-String wird dem `RemoteTimer` als Struct-Feld übergeben. Ebenso wird `getUserForMobTimer()` in `main` aufgelöst und als `User`-Feld übergeben.
6. **`Branch` bleibt in `main`** (Variante 1) – Branch ist ein Mob-Domain-Typ. Das `timer`-Package braucht ihn nicht (Room/User werden vorher aufgelöst). Die 4 git-abhängigen Branch-Methoden (`hasRemoteBranch`, `hasLocalBranch`, `hasUnpushedCommits`, `hasLocalCommits`) bekommen einen `*git.GitClient` Parameter. Aufwand minimal (~11 Aufrufstellen).
7. **Neues `git`-Package** – Alle Git-Befehle werden in einem `git`-Package gekapselt. Statt globalem `workingDir` wird ein `GitClient`-Struct verwendet. Das `git`-Package ist ein Leaf-Package ohne Abhängigkeit auf `config` oder Mob-Domain-Logik.

## Architektur: `git`-Package

### Design

```go
package git

// GitClient kapselt alle Git-Operationen. workingDir bestimmt,
// in welchem Verzeichnis die Git-Befehle ausgeführt werden.
type GitClient struct {
    workingDir              string
    passthroughStderrStdout bool
}

func NewGitClient(workingDir string) *GitClient {
    return &GitClient{workingDir: workingDir}
}

func (g *GitClient) SetPassthroughStderrStdout(enabled bool) {
    g.passthroughStderrStdout = enabled
}
```

### Was wandert ins `git`-Package?

**Execution Layer** (unexportiert – nur intern genutzt):
- `runCommandSilent(name string, args ...string) (string, string, error)`
- `runCommand(name string, args ...string) (string, string, error)`
- `startCommand(name string, args ...string) (string, error)`

**Git-Executors** (exportiert):
- `SilentGit(args ...string) string`
- `SilentGitIgnoreFailure(args ...string) (string, error)`
- `Git(args ...string)`
- `GitWithoutEmptyStrings(args ...string)`
- `GitIgnoreFailure(args ...string) error`

**Git-Queries** (exportiert):
- `CurrentBranch() string` (gibt Branch-Name als String zurück, `main` erstellt daraus ein `Branch`-Objekt)
- `Branches() []string`
- `RemoteBranches() []string`
- `UserName() string`
- `UserEmail() string`
- `RootDir() string`
- `Dir() string`
- `Version() string`
- `CommitHash() string`

**Git-State-Checks** (exportiert):
- `IsGit() bool`
- `HasCommits() bool`
- `IsNothingToCommit() bool`
- `HasUncommittedChanges() bool`
- `DoBranchesDiverge(ancestor, successor string) bool`
- `HasRemoteBranch(branchName, remoteName string) bool` (bisher `Branch.hasRemoteBranch`)
- `HasLocalBranch(branchName string) bool` (bisher `Branch.hasLocalBranch`)
- `HasUnpushedCommits(branchName, remoteBranchName string) bool` (bisher `Branch.hasUnpushedCommits`)

**Git-Helpers** (exportiert):
- `GetUntrackedFiles() string`
- `GetUnstagedChanges() string`
- `GetChangesOfLastCommit() string`
- `GetCachedChanges() string`
- `GetModifiedFiles(rootDir string) []string`

### Was bleibt in `main`?

- `Branch` struct und alle seine Methoden (reine String/Config-Logik)
- `determineBranches()` (Mob-Domain-Logik)
- `getMobTimerRoom()` (nutzt Git + Branch + determineBranches)
- `getUserForMobTimer()` (nutzt GitClient.UserName)
- `showNext()`, `sayLastCommitsList()` etc. (Mob-Domain-Logik die Git nutzt)
- `makeWipCommit()`, `deleteRemoteWipBranch()` etc. (Mob-Workflow-Funktionen)

### Migration in `main`

In `mob.go` wird ein package-level `gitClient` erstellt und in `run()` initialisiert:

```go
var gitClient *git.GitClient

func run(args) {
    // ...
    gitClient = git.NewGitClient(workingDir)
    // ...
}
```

Die bisherigen Aufrufe ändern sich minimal:
- `silentgit("status", "--porcelain")` → `gitClient.SilentGit("status", "--porcelain")`
- `gitCurrentBranch()` → `newBranch(gitClient.CurrentBranch())`
- `branch.hasRemoteBranch(config)` → `gitClient.HasRemoteBranch(branch.remote(config).Name, branch.remote(config).Name)` (TODO: Signatur vereinfachen)

## Umsetzungsschritte

- [ ] **Schritt 1: `git/`-Package anlegen und Git-Infrastruktur verschieben**
  - Neues Package `git/` erstellen
  - `git/git.go`: `GitClient` struct mit `NewGitClient(workingDir string)`
  - Execution Layer verschieben: `runCommandSilent`, `runCommand`, `startCommand` als unexportierte Methoden auf `GitClient`
  - Git-Executors verschieben: `SilentGit`, `SilentGitIgnoreFailure`, `Git`, `GitWithoutEmptyStrings`, `GitIgnoreFailure`
  - Git-Queries verschieben: `CurrentBranch` (→ gibt `string` zurück), `Branches`, `RemoteBranches`, `UserName`, `UserEmail`, `RootDir`, `Dir`, `Version`, `CommitHash`
  - Git-State-Checks verschieben: `IsGit`, `HasCommits`, `IsNothingToCommit`, `HasUncommittedChanges`, `DoBranchesDiverge`
  - Branch-bezogene Git-Checks als GitClient-Methoden: `HasRemoteBranch(branchName, remoteName string)`, `HasLocalBranch(branchName string)`, `HasUnpushedCommits(branchName, remoteBranchName string)`
  - Git-Helpers verschieben: `GetUntrackedFiles`, `GetUnstagedChanges`, `GetChangesOfLastCommit`, `GetCachedChanges`, `GetModifiedFiles`
  - In `mob.go`: package-level `var gitClient *git.GitClient`, Initialisierung in `run()`
  - Alle Aufrufstellen in `mob.go` und `timer.go` auf `gitClient.XYZ()` umstellen
  - `Branch.hasRemoteBranch/hasLocalBranch/hasUnpushedCommits/hasLocalCommits` entfernen und durch `gitClient.HasRemoteBranch(...)` etc. ersetzen
  - Globale Variable `workingDir` entfernen (lebt jetzt im GitClient)
  - Globale Variable `GitPassthroughStderrStdout` durch `gitClient.SetPassthroughStderrStdout()` ersetzen
  - Tests anpassen / sicherstellen dass alles kompiliert und grün ist

- [ ] **Schritt 2: `timer/`-Package anlegen, Interface definieren und Hilfsfunktionen verschieben**
  - Neues Package `timer/` erstellen
  - `timer/timer.go`: Interface `Timer` mit `Start(durationMinutes int, timerType TimerType, configuration config.Configuration) error`
  - `TimerType`-Enum definieren (`TimerTypeNormal`, `TimerTypeBreak`)
  - `openTimerInBrowser()` als exportierte Funktion `OpenTimerInBrowser()` ins `timer`-Package verschieben
  - `moo()` als exportierte Funktion `Moo()` ins `timer`-Package verschieben
  - `executeCommandsInBackgroundProcess()` im `timer`-Package neu implementieren: eigene schlanke Version ohne `workingDir`/`startCommand()`, nutzt direkt `exec.Command` (Original in `mob.go` kann entfernt werden, da `moo()` der letzte Nutzer war – prüfen ob `startCommand` in mob.go noch anderweitig gebraucht wird)
  - `getSleepCommand()`, `getVoiceCommand()`, `getNotifyCommand()` ins `timer`-Package verschieben
  - `injectCommandWithMessage()` ins `timer`-Package verschieben
  - Aufrufe in `mob.go` anpassen: `timer.OpenTimerInBrowser(configuration)` / `timer.Moo(configuration)`

- [ ] **Schritt 3: `LocalTimer`-Struct im `timer`-Package erstellen**
  - `timer/local.go`: `LocalTimer` struct (ohne eigene Felder)
  - `Start()`-Methode implementieren: bestehende Logik aus `startTimer()` / `startBreakTimer()` extrahieren (sleep + voice + notify + echo)
  - Nutzt die bereits im `timer`-Package vorhandenen Hilfsfunktionen (`executeCommandsInBackgroundProcess`, `getSleepCommand`, etc.)

- [ ] **Schritt 4: `RemoteTimer`-Struct im `timer`-Package erstellen**
  - `timer/remote.go`: `RemoteTimer` struct mit `Room string` und `User string`
  - `Start()`-Methode implementieren: bestehende Logik aus `httpPutTimer()` / `httpPutBreakTimer()` extrahieren
  - `httpPutTimer()` und `httpPutBreakTimer()` zu einer Methode zusammenführen (Unterscheidung über `TimerType`)
  - `getMobTimerRoom()` und `getUserForMobTimer()` bleiben in `main` – sie lösen Room/User auf und übergeben die Strings an `RemoteTimer{Room: room, User: user}`

- [ ] **Schritt 5: Builder/Factory-Funktion erstellen**
  - `buildTimers(configuration) []timer.Timer` implementieren (in `timer.go`, bleibt in `main`)
  - Löst Room/User vorab auf via `getMobTimerRoom()` / `getUserForMobTimer()`
  - Erstellt `&timer.RemoteTimer{Room: room, User: user}` wenn Remote-Timer aktiv
  - Erstellt `&timer.LocalTimer{}` wenn `TimerLocal=true`
  - Ersetzt die bisherige `if startRemoteTimer` / `if startLocalTimer` Logik

- [ ] **Schritt 6: `startTimer()` und `startBreakTimer()` refactoren**
  - Gemeinsame Logik zusammenführen (die Funktionen sind zu ~90% identisch)
  - Interface-Aufrufe statt direkte Implementierung nutzen
  - Die Unterscheidung Normal/Break über `TimerType` abbilden
  - `StartTimer()` und `StartBreakTimer()` (exportierte Funktionen) beibehalten als öffentliche API

- [ ] **Schritt 7: Tests anpassen**
  - Bestehende Tests in `timer_test.go` anpassen
  - Mock-Implementation des `Timer`-Interfaces für Unit-Tests erstellen
  - Sicherstellen, dass alle bestehenden Tests weiterhin grün sind
  - Neue Tests für `LocalTimer` und `RemoteTimer` separat schreiben

- [ ] **Schritt 8: Aufräumen**
  - Verwaiste Funktionen in `timer.go` und `mob.go` entfernen, deren Logik nun in `RemoteTimer.Start()` bzw. `LocalTimer.Start()` lebt (z.B. `httpPutTimer`, `httpPutBreakTimer` → zusammengeführt in `RemoteTimer.Start()`)
  - `hasLocalCommits` auf Branch entfernen (unused)
  - `gitUserEmail` entfernen falls weiterhin unused
  - Alle Tests ausführen und sicherstellen dass nichts kaputt ist
