# DeadKeys guide

Everything about installing, configuring and building DeadKeys. For a short
overview, see the [README](README.md).

## How it works

Some laptop keyboard faults kill a whole column of keys at once: one wire in
the keyboard matrix serves several keys, so when it breaks they all go dead
together. No driver or setting can bring them back. DeadKeys lets other keys
type the missing characters instead:

- **Layer rules:** hold CapsLock and press a working key. CapsLock + Q types `W`.
- **Direct remaps:** a key you never use always types a dead one. Insert types `S`.

Shift and Ctrl still work as normal, because DeadKeys only swaps the key and
leaves your real modifier keys alone:

| You press                  | You get |
| -------------------------- | ------- |
| CapsLock + Q               | `w`     |
| Shift + CapsLock + Q       | `W`     |
| Shift + CapsLock + 1       | `@`     |
| Shift + CapsLock + 8       | `(`     |
| Ctrl + CapsLock + A        | Ctrl+S (save) |

A quick tap of CapsLock on its own still toggles caps lock.

## Default rules

The defaults fit a ThinkPad E15 whose `2 W S X` and `O 9` keys are dead.

| Hold CapsLock + | Types |
| --------------- | ----- |
| Q               | W     |
| A               | S     |
| Z               | X     |
| I               | O     |
| 1               | 2     |
| 8               | 9     |

Each replacement is the key just left of (or below) the dead one, so it's easy to
remember. Change them in the settings page or in `config.json`.

## Install

### With npx

If you have [Node.js](https://nodejs.org) 18 or newer, run:

```
npx deadkeys
```

This downloads `deadkeys.exe` from the latest release, checks it against
`SHA256SUMS.txt`, and starts it. The exe and `config.json` go in
`%LOCALAPPDATA%\DeadKeys`, so your rules survive npm clearing its cache. Run
the same command again to open the settings page. `npx deadkeys --debug`
opens the debug console.

### From the release

1. Download `DeadKeys-<version>-windows-amd64.zip` from the
   [Releases](../../releases) page.
2. Unzip it to a folder you can write to, for example
   `%LOCALAPPDATA%\DeadKeys` (paste that into the Explorer address bar).
   Don't use `Program Files`: DeadKeys saves `config.json` next to the exe.
3. Double-click `deadkeys.exe`.

A blue keycap icon appears in the tray (you may need to click the `^` arrow
next to the clock and drag it out so it's always visible).

### Programs that run as admin

Windows doesn't let a normal program change keys typed into programs running
as admin, such as Task Manager, Registry Editor or an admin terminal. In those
windows the dead keys stay dead. Everywhere else DeadKeys works without admin.

If you need the rules inside admin programs too, right-click `deadkeys.exe`
and choose **Run as administrator**, or use Option B below to start it as
admin at login.

### "Windows protected your PC"

The exe isn't code-signed, so SmartScreen warns about it the first time. Click
**More info**, then **Run anyway**.

### Windows Defender

Any program that watches the keyboard can look like a keylogger, so Defender
may quarantine `deadkeys.exe`. DeadKeys never stores or sends your keystrokes:
the only network use is the settings page, which listens on `127.0.0.1` (your
own PC) and needs a secret token that changes on every start.

If Defender removes it:

1. Open **Windows Security**, then **Virus & threat protection**, then
   **Protection history**.
2. Find the DeadKeys entry, click **Actions**, then **Allow** (or **Restore**).
3. To stop it happening again, go to **Virus & threat protection settings**,
   **Exclusions**, **Add an exclusion**, **Folder**, and pick your DeadKeys
   folder.

## Using it

- **Left-click the tray icon** to open the settings page in your browser.
- **Right-click it** for **Settings**, **Pause** (turns all rules off and
  greys out the icon) and **Quit**.
- **Running `deadkeys.exe` again** also opens the settings page, so a desktop
  shortcut works as a "settings" button.

### Settings page

The page draws the ThinkPad E15 keyboard.

1. **Which keys don't work?** Click each dead key on the keyboard. Click it
   again to undo.
2. **How to type them.** Each dead key gets a row. If it has no shortcut yet,
   the row suggests a nearby working key: click **Use Caps Lock + R** to accept
   it, or **Choose another** and click or press the key you want. **Use a
   spare key** makes a key you never use always type the dead one.
3. Click **Save** (or press Ctrl+S). The new rules work straight away; no
   restart needed. Type in the **Try it** box to check them.

Every change can be undone from the message at the bottom of the page, or
with Ctrl+Z. **More settings** holds the hold key, the quick-tap time and
**Reset to defaults**. For the hold key, pick one you rarely type on its own
(the list puts those first): a letter or Space only types when tapped, so fast
typing can lose it.

If the page is open in two tabs, saving in one updates the other when you
switch back to it.

On the keyboard, red keys are dead, a green letter shows what a key types
while you hold Caps Lock, and a purple letter marks a spare key.

## Start automatically at login

### Option A: Startup folder (recommended)

Press `Win + R`, type `shell:startup`, press Enter, and put a shortcut to
`deadkeys.exe` in that folder. DeadKeys then starts at login without admin
rights, so it won't work inside admin programs (see above).

If you installed with npx, make the shortcut run `npx -y deadkeys` instead,
and set it to start minimized. A shortcut to the exe itself breaks on the
next update, because each version lives in its own folder.

### Option B: Task Scheduler, as admin with no UAC prompt

Open **PowerShell as administrator** and run (change the path if you put
DeadKeys somewhere else):

```powershell
$exe       = "$env:LOCALAPPDATA\DeadKeys\deadkeys.exe"
$action    = New-ScheduledTaskAction -Execute $exe
$trigger   = New-ScheduledTaskTrigger -AtLogOn -User $env:USERNAME
$principal = New-ScheduledTaskPrincipal -UserId $env:USERNAME -LogonType Interactive -RunLevel Highest
$settings  = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -ExecutionTimeLimit ([TimeSpan]::Zero)
Register-ScheduledTask -TaskName "DeadKeys" -Action $action -Trigger $trigger -Principal $principal -Settings $settings -Force
```

To start it now without logging out: `Start-ScheduledTask DeadKeys`.
To remove it: `Unregister-ScheduledTask DeadKeys`.

If you'd rather click through Task Scheduler yourself, create a task that
runs `deadkeys.exe` **At log on**, tick **Run with highest privileges**, and
on the **Conditions** and **Settings** tabs untick **Start the task only if
the computer is on AC power** and **Stop the task if it runs longer than 3
days**. Both are on by default and would stop DeadKeys on a laptop.

## config.json

The settings page edits this file for you, but you can also edit it by hand.
It lives next to `deadkeys.exe` (or pass `--config path\to\file.json`). If it's
missing, DeadKeys writes the defaults. After editing by hand, quit and restart
DeadKeys (right-click the tray icon, **Quit**). If the file has a mistake,
DeadKeys tells you what's wrong and which rule it's on. Saving from the
settings page rewrites key names in their standard spelling (`caps` becomes
`CapsLock`).

```json
{
  "version": 1,
  "layerKey": "CapsLock",
  "tapTimeoutMs": 300,
  "layer": [
    { "from": "Q", "to": "W" },
    { "from": "1", "to": "2" }
  ],
  "remap": [
    { "from": "Insert", "to": "S" }
  ],
  "deadKeys": ["2", "W", "S", "X", "O", "9"]
}
```

| Field          | Meaning |
| -------------- | ------- |
| `version`      | Format version. Leave it at `1`. |
| `layerKey`     | The key you hold to use `layer` rules. Tapping it still sends it as normal. |
| `tapTimeoutMs` | A press shorter than this, with no other key, counts as a tap (toggles caps lock). `0` turns tapping off, so CapsLock never toggles. |
| `layer`        | Rules that work while `layerKey` is held. |
| `remap`        | Rules that are always on. The `from` key stops typing itself. |
| `deadKeys`     | Keys that don't work. A rule can't start from a dead key. The settings page uses this to colour the keyboard. |

### Key names

Names are not case-sensitive.

- Letters and digits: `A` to `Z`, `0` to `9`
- Punctuation: `` ` `` `-` `=` `[` `]` `\` `;` `'` `,` `.` `/`
  (or `Backquote`, `Minus`, `Equals`, `LBracket`, `RBracket`, `Backslash`,
  `Semicolon`, `Quote`, `Comma`, `Period`, `Slash`)
- `Esc` `Tab` `CapsLock` `Enter` `Backspace` `Space` `Menu`
- `LShift` `RShift` `LCtrl` `RCtrl` `LAlt` `RAlt` `LWin` `RWin`
- `Insert` `Delete` `Home` `End` `PageUp` `PageDown`
- `Up` `Down` `Left` `Right`
- `PrintScreen` `ScrollLock` `Pause` `NumLock`
- `F1` to `F24`
- Number pad: `Num0` to `Num9`, `NumAdd`, `NumSubtract`, `NumMultiply`,
  `NumDivide`, `NumDecimal`

Any other key can be given by its
[virtual-key code](https://learn.microsoft.com/windows/win32/inputdev/virtual-key-codes),
like `"0xAD"` (volume mute). To find a key's name or code, run
`deadkeys.exe --debug`: a console window opens and prints every key you press.

## Command-line options

| Option            | Meaning |
| ----------------- | ------- |
| `--debug`         | Open a console window and log every key event and what DeadKeys did with it. Close the window or press Ctrl+C to quit. |
| `--config <path>` | Use a different config file. |
| `--version`       | Show the version. |

## Good to know

- **Sign-in screen and UAC prompts:** Windows doesn't let any program change
  keys on the sign-in screen, the lock screen or the UAC (admin) prompt. If
  your password uses a dead key, use a PIN, or the on-screen keyboard (the
  accessibility button in the bottom-right corner of the sign-in screen).
- **Fn** never reaches Windows (the keyboard handles it itself), so it can't be
  remapped.
- **Enter and number-pad Enter** are the same key to Windows, so a rule on
  one applies to both.
- **Long idle or heavy load:** Windows silently turns off a keyboard hook that
  answers too slowly. DeadKeys keeps its work tiny and also re-installs its
  hook every 5 minutes, so if that ever happens, the rules come back on their
  own within 5 minutes.
- **Stuck key?** Press and release it once. Quitting DeadKeys also releases
  every key it was holding.

## Building from source

You need [Go](https://go.dev/dl/) 1.24 or newer. No C compiler is needed.

```sh
go test ./...
GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui" -o deadkeys.exe .
```

On Windows in PowerShell:

```powershell
$env:GOOS = "windows"; $env:GOARCH = "amd64"
go build -ldflags "-H=windowsgui" -o deadkeys.exe .
```

`-H=windowsgui` stops a console window from opening when DeadKeys starts.
`--debug` opens one when you want it.

### Releases

Pushing a tag like `v0.1.0` builds `deadkeys.exe` on GitHub Actions and
publishes it, zipped with `config.json`, the README and this guide, on the
Releases page.
Every push also uploads a test build under the **Actions** tab.

```sh
git tag v0.1.0
git push origin v0.1.0
```

### Publishing to npm

The npm package only holds a launcher that downloads `deadkeys.exe` from the
release whose tag matches the package version. Publish it after the GitHub
release is up, with `version` in `npm/package.json` set to the tag without
the `v`:

```sh
cd npm
npm publish
```

Check it with `npx deadkeys@0.1.0` on Windows afterwards.

### Code layout

| File               | What it does |
| ------------------ | ------------ |
| `layer.go`         | The state machine: takes key events, decides what to swallow and what to send. No Windows code, fully unit tested. |
| `config.go`        | Reads, checks and saves `config.json`. |
| `keys.go`          | Key names and Windows virtual-key codes. |
| `settings.go`      | The settings page server and its JSON API. |
| `web/settings.html`| The settings page (built into the exe). |
| `icon.go`          | Draws the tray icon. |
| `hook_windows.go`  | The low-level keyboard hook (`WH_KEYBOARD_LL`) and `SendInput`. |
| `tray_windows.go`  | The tray icon and menu. |
| `app_windows.go`   | Startup, single instance, debug console. |
| `win32_windows.go` | The raw Win32 functions and structs used. |

The hook runs on its own thread with its own message loop, separate from the
tray, so nothing the tray or settings page does can slow down typing. Keys sent
by DeadKeys are marked as "injected" by Windows, and the hook lets those
through untouched, so it never remaps its own output.

## Roadmap

- A key test screen: press every key, and DeadKeys works out which ones are
  dead and suggests rules using the nearest working keys.
- Export the rules as a [kanata](https://github.com/jtroo/kanata) or
  [AutoHotkey](https://www.autohotkey.com/) config.
