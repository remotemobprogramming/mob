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
- **`Configuration` als Parameter**: Die gesamte Configuration wird übergeben statt einzelner Felder. So hat jede Implementierung Zugriff auf alle relevanten Config-Werte (der `RemoteTimer` braucht z.B. `TimerRoom`, `TimerUser`, `TimerUrl`, `TimerInsecure`; der `LocalTimer` braucht `VoiceCommand`, `VoiceMessage`, `NotifyCommand`, `NotifyMessage`). Neue Implementierungen können auf weitere Config-Felder zugreifen, ohne dass das Interface angepasst werden muss.
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

type RemoteTimer struct{}

func (t *RemoteTimer) Start(durationMinutes int, timerType TimerType, configuration config.Configuration) error {
    room := getMobTimerRoom(configuration)
    user := getUserForMobTimer(configuration.TimerUser)

    // JSON-Body je nach TimerType: "timer" oder "breaktimer"
    timerKey := "timer"
    if timerType == TimerTypeBreak {
        timerKey = "breaktimer"
    }

    putBody, _ := json.Marshal(map[string]interface{}{
        timerKey: durationMinutes,
        "user":   user,
    })
    client := httpclient.CreateHttpClient(configuration.TimerInsecure)
    _, err := client.SendRequest(putBody, "PUT", configuration.TimerUrl+room)
    return err
}
```

Der `RemoteTimer` liest `TimerRoom`, `TimerUser`, `TimerUrl` und `TimerInsecure` direkt aus der Configuration.

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
4. **`executeCommandsInBackgroundProcess()` wird ins `timer`-Package verschoben** (nicht kopiert) – Da `moo()` ebenfalls ins `timer`-Package wandert, gibt es keinen Nutzer mehr in `mob.go`. Die Funktion wird verschoben.

## Umsetzungsschritte

- [ ] **Schritt 1: `timer/`-Package anlegen und Interface definieren**
  - Neues Package `timer/` erstellen
  - `timer/timer.go`: Interface `Timer` mit `Start(durationMinutes int, timerType TimerType, configuration config.Configuration) error`
  - `TimerType`-Enum definieren (`TimerTypeNormal`, `TimerTypeBreak`)

- [ ] **Schritt 2: `LocalTimer`-Struct im `timer`-Package erstellen**
  - `timer/local.go`: `LocalTimer` struct (ohne eigene Felder)
  - `Start()`-Methode implementieren: bestehende Logik aus `startTimer()` / `startBreakTimer()` extrahieren (sleep + voice + notify + echo)
  - `executeCommandsInBackgroundProcess()` aus `mob.go` ins `timer`-Package **verschieben** (nicht kopieren – `moo()` wandert ebenfalls hierher)
  - `getSleepCommand()`, `getVoiceCommand()`, `getNotifyCommand()` ins `timer`-Package verschieben
  - `injectCommandWithMessage()` ins `timer`-Package verschieben

- [ ] **Schritt 3: `RemoteTimer`-Struct im `timer`-Package erstellen**
  - `timer/remote.go`: `RemoteTimer` struct (ohne eigene Felder)
  - `Start()`-Methode implementieren: bestehende Logik aus `httpPutTimer()` / `httpPutBreakTimer()` extrahieren
  - `httpPutTimer()` und `httpPutBreakTimer()` zu einer Methode zusammenführen (Unterscheidung über `TimerType`)
  - `getMobTimerRoom()` und `getUserForMobTimer()` ins `timer`-Package verschieben

- [ ] **Schritt 4: Builder/Factory-Funktion erstellen**
  - `buildTimers(configuration) []timer.Timer` implementieren (in `timer.go` oder im `timer`-Package)
  - Entscheidet basierend auf `TimerLocal` und `TimerRoom`/`getMobTimerRoom()` welche Timer-Implementierungen erstellt werden
  - Ersetzt die bisherige `if startRemoteTimer` / `if startLocalTimer` Logik

- [ ] **Schritt 5: `startTimer()` und `startBreakTimer()` refactoren**
  - Gemeinsame Logik zusammenführen (die Funktionen sind zu ~90% identisch)
  - Interface-Aufrufe statt direkte Implementierung nutzen
  - Die Unterscheidung Normal/Break über `TimerType` abbilden
  - `StartTimer()` und `StartBreakTimer()` (exportierte Funktionen) beibehalten als öffentliche API

- [ ] **Schritt 6: Tests anpassen**
  - Bestehende Tests in `timer_test.go` anpassen
  - Mock-Implementation des `Timer`-Interfaces für Unit-Tests erstellen
  - Sicherstellen, dass alle bestehenden Tests weiterhin grün sind
  - Neue Tests für `LocalTimer` und `RemoteTimer` separat schreiben

- [ ] **Schritt 7: `openTimerInBrowser()` und `moo()` ins `timer`-Package verschieben**
  - `openTimerInBrowser()` als exportierte Funktion `OpenTimerInBrowser()` ins `timer`-Package verschieben
  - `moo()` als exportierte Funktion `Moo()` ins `timer`-Package verschieben
  - Aufrufe in `mob.go` anpassen: `timer.OpenTimerInBrowser(configuration)` / `timer.Moo(configuration)`

- [ ] **Schritt 8: Aufräumen**
  - Nicht mehr benötigte Hilfsfunktionen in `timer.go` und `mob.go` entfernen (z.B. `httpPutTimer`, `httpPutBreakTimer`, alte `executeCommandsInBackgroundProcess`)
  - Alle Tests ausführen und sicherstellen dass nichts kaputt ist
