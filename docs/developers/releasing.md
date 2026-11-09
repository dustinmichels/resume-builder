# Releasing

```sh
mise run release              # patch bump
mise run release minor        # or major, or an exact X.Y.Z
```

`mise run release` cuts a release from a clean `main` that's level with `origin/main`, on a Mac. It:

1. Checks signing and notarization will work, and that `gh` can push to the Homebrew tap.
2. Runs the server tests.
3. Builds the archives and the Mac app.
4. Signs and notarizes the app.
5. Tags `vX.Y.Z`, pushes the tag, and publishes a GitHub release with `Resume-Builder.dmg`, `Resume-Builder.zip`, the
   archives and `checksums.txt`.
6. Points the Homebrew cask at the new release.

Nothing is tagged until notarization has passed. Versions come from the git tags; the first release was `v0.1.0`. It
needs the [GitHub CLI](https://cli.github.com) logged in (`gh auth login`) and able to push to
[dustinmichels/homebrew-tap](https://github.com/dustinmichels/homebrew-tap).

`mise run package [version]` builds everything into `build/release/` without publishing. Elsewhere than macOS it skips
the Mac app.

## Homebrew cask

The last step runs `scripts/update-cask.sh <version>`: it downloads the published `Resume-Builder.dmg`, hashes it, and
commits the new `version` and `sha256` to `Casks/resume-builder.rb` in the tap through the GitHub API. If that step
fails, the release is already out; rerun `mise run cask` (latest release) or `scripts/update-cask.sh <version>`.
Rerunning is a no-op when the cask already matches.

## What's in a release

| Asset                                  | Built by                   | Holds                                                                     |
| -------------------------------------- | -------------------------- | ------------------------------------------------------------------------- |
| `resume-builder-<os>-<arch>.tar.gz/.zip` | `scripts/package.sh`     | `resume-builder` (`-tags release`, editor and templates embedded), the Typst binary pinned in `mise.toml`, Typst's license |
| `Resume-Builder.dmg`                   | `scripts/package-macos.sh` | `Resume Builder.app`: universal (arm64 + x86_64) `resume-builder` (`-tags "release desktop production"`) and Typst |
| `Resume-Builder.zip`                   | `scripts/package-macos.sh` | The same app alone, for the in-app updater                                |
| `checksums.txt`                        | `scripts/package.sh`       | SHA-256 of every asset                                                    |

The in-app update (`server/desktop_update.go`, Wails' `pkg/updater`) downloads `Resume-Builder.zip` from the latest
release, checks it against `checksums.txt` and swaps it in for the running app. Keep those asset names and the
`vX.Y.Z` tags.

The icon is `macos/AppIcon.icns`, made from `macos/icon.svg` by `scripts/icon.sh`, which also writes `server/icon.png`
(the Dock icon the app sets at run time, so `mise run app` shows it too).

## Signing setup

Once per Mac:

1. A **Developer ID Application** certificate in the Keychain (developer.apple.com → Certificates, or Xcode → Settings →
   Accounts → Manage Certificates). `security find-identity -v -p codesigning` lists it. Back up its private key as a
   `.p12`.
2. A notarytool profile named `resume-builder`, with an app-specific password for the Apple ID:

   ```sh
   xcrun notarytool store-credentials resume-builder --apple-id <apple id> --team-id LLM5F36KYW
   ```

`CODESIGN_IDENTITY` and `NOTARY_PROFILE` override either. Without the certificate, `mise run package` signs the app ad
hoc and skips notarization: it runs on the Mac that built it, but Gatekeeper blocks it elsewhere.
