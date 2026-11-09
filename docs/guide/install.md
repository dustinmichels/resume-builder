# Install

Resume Builder runs on your own computer. There's no account and nothing to sign up for: your resumes are files in a
folder you choose.

## Mac

**Download:**

1. Download [**Resume-Builder.dmg**](https://github.com/dustinmichels/resume-builder/releases/latest/download/Resume-Builder.dmg).
2. Open it and drag **Resume Builder** to **Applications**.
3. Open it from Applications or Launchpad like any other app.

**Or with [Homebrew](https://brew.sh)**, which also installs Poppler (see below):

```sh
brew install --cask dustinmichels/tap/resume-builder
```

It runs on macOS 13 (Ventura) or later, on Apple Silicon and Intel Macs. The app is signed and notarized by Apple, so it
opens without warnings.

On first run, choose where to keep your resumes (`~/resume-builder/` unless you pick another folder). If the folder is
empty you start from a sample resume.

::: tip Turn on the machine view
The **What machines read** tab and the screen-reader and PDF-link checks use [Poppler](https://poppler.freedesktop.org),
which the app doesn't include yet. The Homebrew cask installs it for you. If you installed from the disk image and have
Homebrew, run:

```sh
brew install poppler
```

Everything else works without it.
:::

### Updates

The app checks GitHub for a newer release when it opens and once a day while it's open, and stays quiet unless there is
one. **Resume Builder → Check for Updates…** checks right away.

Nothing is downloaded without asking. A new release brings up **Update**, **Later** and **Release Notes**. **Update**
downloads the new version and checks it against the release's checksums; **Restart & Apply** then replaces the app and
reopens it. Your resumes aren't touched.

A copy that can't replace itself (opened straight from the disk image, or in a folder you can't change) offers
**Download** instead, which gets the new `Resume-Builder.dmg` to drag over the old app.

### Logs

The app logs to `~/Library/Logs/Resume Builder/`. **Help → Show Log File** opens it.

## Command line

Every [release](https://github.com/dustinmichels/resume-builder/releases/latest) also has an archive per platform with a
`resume-builder` program to run from a terminal. It opens the editor in your browser at `http://127.0.0.1:8787`. Leave
the terminal open while you use it, and press <kbd>Ctrl</kbd>+<kbd>C</kbd> to stop it.

Each archive carries the editor, the templates and Typst, so nothing else is needed (Poppler is optional, as above).

| Platform              | Archive                                |
| --------------------- | -------------------------------------- |
| macOS (Apple Silicon) | `resume-builder-darwin-arm64.tar.gz`   |
| macOS (Intel)         | `resume-builder-darwin-amd64.tar.gz`   |
| Linux (x86-64)        | `resume-builder-linux-amd64.tar.gz`    |
| Linux (ARM64)         | `resume-builder-linux-arm64.tar.gz`    |
| Windows (x86-64)      | `resume-builder-windows-amd64.zip`     |
| Windows (ARM64)       | `resume-builder-windows-arm64.zip`     |

::: code-group

```sh [macOS]
# Download with curl, not a browser: the command-line build is unsigned,
# and Gatekeeper refuses to run files a browser downloaded.
curl -fsSL https://github.com/dustinmichels/resume-builder/releases/latest/download/resume-builder-darwin-arm64.tar.gz | tar -xz
./resume-builder-darwin-arm64/resume-builder

# Optional, for the machine view and screen-reader checks
brew install poppler
```

```sh [Linux]
curl -fsSL https://github.com/dustinmichels/resume-builder/releases/latest/download/resume-builder-linux-amd64.tar.gz | tar -xz
./resume-builder-linux-amd64/resume-builder

# Optional, for the machine view and screen-reader checks
sudo apt install poppler-utils
```

```powershell [Windows]
# Download and unzip resume-builder-windows-amd64.zip from the releases page, then:
.\resume-builder-windows-amd64\resume-builder.exe
```

:::

When a newer release is out, `resume-builder` says so in its log on start, with a link. Update by downloading the new
archive the same way.

### Options

| Flag              | Default                                      | What it does                                      |
| ----------------- | -------------------------------------------- | ------------------------------------------------- |
| `-addr`           | `127.0.0.1:8787`                             | Address to serve the editor on                    |
| `-open`           | `true`                                       | Open the editor in your browser once serving      |
| `-config`         | `<user config dir>/resume-builder/config.json` | Settings file (remembers your data folder)      |
| `-template`       | bundled                                      | Use your own copy of the templates                |

The bundled templates are unpacked on every start into `<user cache dir>/resume-builder/template/` (on macOS
`~/Library/Caches/resume-builder/template/`), undoing edits there. To change a template, copy that folder and run with
`-template <copy>/resume.typ`. See [Writing templates](/developers/templates).

## Next

Take the [tour of the editor](./tour).
