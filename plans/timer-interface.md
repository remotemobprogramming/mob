# Plan: Timer Interface Refactoring

## Ziel

Die bestehende Timer-Implementierung in `mob` soll hinter einem klar definierten Go-Interface abstrahiert werden. Dadurch wird es möglich, neben der aktuellen lokalen Timer-Implementierung (Shell-Befehle im Hintergrund) zukünftig weitere Implementierungen (z. B. ein UI-basierter Timer, ein HTTP-basierter Timer, etc.) hinzuzufügen, ohne den bestehenden Code zu verändern.

## Ist-Zustand

### Betroffene Dateien

| Datei | Relevanz |
|---|---|
| `timer.go` | Kernlogik: `startTimer()`, `startBreakTimer()`, lokale + remote Timer-Logik, Hilfsfunktionen |
| `mob.go:484-501` | `executeCommandsInBackgroundProcess()` – führt Shell-Befehle im Hintergrund aus |
| `mob.go:385-399` | `openTimerInBrowser()` – öffnet Remote-Timer im Browser |
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
type Timer interface {
    // Start startet einen Timer mit der gegebenen Dauer und dem gegebenen Typ.
    // Die message wird bei Ablauf des Timers angezeigt/gesprochen.
    Start(durationMinutes int, timerType TimerType, message string) error
}
```

### Warum dieses Design?

- **Ein Interface, eine Methode**: `Start()` ist die einzige Operation, die beide Timer-Typen (lokal und remote) gemeinsam haben. Ein minimales Interface ist leichter zu implementieren.
- **`TimerType` als Parameter**: Statt zwei Methoden (`StartTimer`/`StartBreakTimer`) wird der Typ als Parameter übergeben. Das vermeidet die aktuelle Code-Duplizierung und hält das Interface schlank.
- **`message` als Parameter**: Die Nachricht (`"mob next"` / `"mob start"`) wird von außen übergeben, nicht hardcoded.
- **Kein `Stop()`**: Der aktuelle Timer hat keine Stop-Funktionalität (der Hintergrund-Prozess läuft einfach aus). Falls zukünftig benötigt, kann das Interface erweitert werden.

### Implementierungen

#### 1. `LocalTimer` (bestehende Logik)

```go
// timer/local.go

type LocalTimer struct {
    VoiceCommand  string
    NotifyCommand string
}

func (t *LocalTimer) Start(durationMinutes int, timerType TimerType, message string) error {
    // Bisherige Logik aus startTimer()/startBreakTimer():
    // executeCommandsInBackgroundProcess(sleep, voice, notify, echo)
}
```

#### 2. `RemoteTimer` (bestehende Logik)

```go
// timer/remote.go

type RemoteTimer struct {
    Room                   string
    User                   string
    TimerService           string
    DisableSSLVerification bool
}

func (t *RemoteTimer) Start(durationMinutes int, timerType TimerType, message string) error {
    // Bisherige Logik: httpPutTimer() / httpPutBreakTimer()
}
```

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
        if err := t.Start(timeoutInMinutes, timer.TimerTypeNormal, configuration.VoiceMessage); err != nil {
            return err
        }
    }

    // User-Feedback wie bisher
}
```

Die Funktion `buildTimers(configuration)` erstellt basierend auf der Konfiguration die passenden `Timer`-Implementierungen (0..n).

## Offene Design-Fragen

1. **Neues Package `timer/` oder im `main`-Package belassen?** – Ein eigenes Package fördert Entkopplung, erfordert aber ggf. dass `executeCommandsInBackgroundProcess` verschoben oder exportiert wird.
2. **Soll `openTimerInBrowser()` Teil des Interfaces sein?** – Aktuell ist es nur für den Remote-Timer relevant. Es könnte als optionale Methode (zweites Interface) oder separat bleiben.
3. **Soll `executeCommandsInBackgroundProcess()` ins `timer`-Package verschoben werden?** – Diese Funktion ist aktuell in `mob.go` und wird nur vom lokalen Timer genutzt. Logisch gehört sie zum lokalen Timer.

## Umsetzungsschritte

- [ ] **Schritt 1: `Timer`-Interface definieren**
  - Neues Package `timer/` anlegen (oder im `main`-Package definieren, je nach Entscheidung zu Frage 1)
  - Interface `Timer` mit `Start(durationMinutes int, timerType TimerType, message string) error` erstellen
  - `TimerType`-Enum definieren (`TimerTypeNormal`, `TimerTypeBreak`)

- [ ] **Schritt 2: `LocalTimer`-Struct erstellen**
  - Struct mit den benötigten Feldern: `VoiceCommand`, `NotifyCommand`, `VoiceMessage`, `NotifyMessage`
  - `Start()`-Methode implementieren: bestehende Logik aus `startTimer()` / `startBreakTimer()` extrahieren (sleep + voice + notify + echo)
  - `executeCommandsInBackgroundProcess()`, `getSleepCommand()`, `getVoiceCommand()`, `getNotifyCommand()` in den Kontext des LocalTimer verschieben
  - `injectCommandWithMessage()` mitnehmen (wird von Voice/Notify gebraucht)

- [ ] **Schritt 3: `RemoteTimer`-Struct erstellen**
  - Struct mit Feldern: `Room`, `User`, `TimerService`, `DisableSSLVerification`
  - `Start()`-Methode implementieren: bestehende Logik aus `httpPutTimer()` / `httpPutBreakTimer()` extrahieren
  - `httpPutTimer()` und `httpPutBreakTimer()` zu einer Methode zusammenführen (Unterscheidung über `TimerType`)

- [ ] **Schritt 4: Builder/Factory-Funktion erstellen**
  - `buildTimers(configuration) []Timer` implementieren
  - Entscheidet basierend auf `TimerLocal` und `TimerRoom` welche Timer-Implementierungen erstellt werden
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

- [ ] **Schritt 7: Aufräumen**
  - Nicht mehr benötigte Hilfsfunktionen in `mob.go` entfernen oder verschieben
  - `openTimerInBrowser()` ggf. dem `RemoteTimer` zuordnen
  - Sicherstellen, dass die Konfiguration sauber an die Timer-Structs übergeben wird
  - Alle Tests ausführen und sicherstellen dass nichts kaputt ist
