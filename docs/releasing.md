# Releasing Cue Space

Releases are signed with a Developer ID certificate and notarized by Apple,
so people can download `CueSpace.dmg` from GitHub and open it without
security warnings. The disk image works on Apple silicon and Intel Macs.

The latest release is always at:

```
https://github.com/ShehabEddin64/Cue_Space/releases/latest/download/CueSpace.dmg
```

## One-time setup

You need to be in the Apple Developer Program, as the Account Holder (only
they can create Developer ID certificates).

### 1. Create a Developer ID Application certificate

1. Open **Keychain Access › Certificate Assistant › Request a Certificate
   From a Certificate Authority…**, enter your email, choose **Saved to
   disk**, and save the `.certSigningRequest` file.
2. On [developer.apple.com › Certificates](https://developer.apple.com/account/resources/certificates/list),
   click **+**, choose **Developer ID Application**, upload the request and
   download the certificate.
3. Double-click the downloaded `.cer` to add it to your login keychain.
4. Check it's there:

   ```bash
   security find-identity -v -p codesigning
   ```

   You should see `Developer ID Application: Your Name (TEAMID)`.

### 2. Create an App Store Connect API key for notarization

1. In [App Store Connect › Users and Access › Integrations › Team Keys](https://appstoreconnect.apple.com/access/integrations/api),
   create a key with the **Developer** role.
2. Download the `AuthKey_XXXXXXXXXX.p8` file (you can only download it once)
   and note the **Key ID** and the **Issuer ID** shown on that page.
3. Save it for local releases:

   ```bash
   xcrun notarytool store-credentials cuespace-notary --key AuthKey_XXXXXXXXXX.p8 --key-id XXXXXXXXXX --issuer YOUR-ISSUER-ID
   ```

### 3. Add the secrets to GitHub (for automatic releases)

Export the certificate with its private key: in Keychain Access, find
**Developer ID Application: …** under **My Certificates**, right-click it,
choose **Export…**, save it as `certificate.p12` and set a password.

Then add five repository secrets (Settings › Secrets and variables ›
Actions), or with the GitHub CLI:

```bash
base64 -i certificate.p12 | gh secret set MACOS_CERTIFICATE_P12
gh secret set MACOS_CERTIFICATE_PASSWORD
base64 -i AuthKey_XXXXXXXXXX.p8 | gh secret set NOTARY_KEY_P8
gh secret set NOTARY_KEY_ID
gh secret set NOTARY_ISSUER_ID
```

Delete `certificate.p12` afterwards; the certificate stays in your keychain.

## Making a release

Push a version tag. The **Release** workflow builds the universal app, signs
it, notarizes it, makes the disk image, notarizes that too and publishes a
GitHub release with `CueSpace.dmg` and generated notes:

```bash
git tag v0.1.0
git push origin v0.1.0
```

To build a release on your Mac instead (it ends up in `dist/CueSpace.dmg`):

```bash
./scripts/release.sh 0.1.0
```

To try the packaging without a certificate, sign locally and skip
notarization (the result only opens on this Mac):

```bash
SIGN_IDENTITY=- ./scripts/release.sh 0.1.0 --no-notarize
```

## Notes

- The disk image background is drawn by `scripts/make-dmg-background.py`;
  run it again after changing the design.
- macOS keeps permissions such as Accessibility across updates because every
  release is signed by the same team.
- If notarization fails, the script prints Apple's response. See the full
  log with `xcrun notarytool log <submission-id> --keychain-profile cuespace-notary`.
