# DeadKeys

**Keep typing on a laptop with dead keys.**

When a laptop keyboard fails, it often takes a whole group of keys with it.
On a ThinkPad E15 that meant `2`, `W`, `S`, `X`, `O` and `9` all stopped at
once. A new keyboard costs money and a repair visit takes days. DeadKeys gets
you typing again in a minute.

Hold **Caps Lock** and press the key next to the dead one:

| Hold Caps Lock + | Types |
| ---------------- | ----- |
| Q                | W     |
| A                | S     |
| 1                | 2     |
| 8                | 9     |

Shift, Ctrl and the rest still work as normal, so Caps Lock + Shift + Q gives
a capital `W` and Ctrl + Caps Lock + A still saves. A quick tap of Caps Lock
still toggles caps lock.

## Features

- **Set up by clicking.** Click your dead keys on a picture of the keyboard,
  accept the suggested shortcut, and save. No config files.
- **Works in every app.** Browsers, Office, chat, code editors. Admin windows
  need DeadKeys to run as admin too.
- **Spare keys.** Turn a key you never use, like Insert, into one you need.
- **Stays out of the way.** Lives in the tray, can start with Windows, and
  pauses from the tray menu.
- **Private.** Nothing you type is stored or sent anywhere.
- **Free.** One exe, no installer, MIT licensed.

## Get it

With [Node.js](https://nodejs.org) installed:

```
npx deadkeys
```

Or download `deadkeys.exe` from [Releases](../../releases) and run it.

The settings page opens on first run. Click the keys that don't work, accept
the suggested shortcuts and save. Later, click the keycap icon in the tray to
open it again.

Windows only. The settings page draws the ThinkPad E15 (US) keyboard, but the
shortcuts work on any laptop.

## Learn more

Install details, starting at login, Windows Defender warnings, the config
file, building from source and how it works inside: see the
[full guide](DOCS.md).

## License

[MIT](LICENSE)
