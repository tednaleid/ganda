ABOUTME: Step-by-step instructions for setting up AUR (Arch User Repository) publishing
ABOUTME: for the ganda project via GoReleaser.

# AUR Setup

These are the manual steps needed to enable automated AUR publishing when a new
ganda release is tagged.

## 1. Generate an SSH keypair for AUR

This key is tied to your AUR account, not to a specific package. You can reuse
it for any AUR packages you maintain.

```bash
ssh-keygen -t ed25519 -f ~/.ssh/aur -C "AUR"
```

Copy the public key to your clipboard:
```bash
cat ~/.ssh/aur.pub | pbcopy
```

## 2. Create an AUR account

Go to https://aur.archlinux.org/register and fill out the registration form.
Paste the contents of `~/.ssh/aur.pub` into the **SSH Public Key** field.

## 3. Add the private key as a GitHub secret

This allows the release workflow to push PKGBUILD updates to AUR on your behalf.

```bash
gh secret set AUR_SSH_KEY < ~/.ssh/aur
```

## 4. Verify the setup

Run a local dry run to confirm the goreleaser config is valid:

```bash
goreleaser check
goreleaser release --snapshot --clean
```

The snapshot build will produce `.deb` and `.rpm` files in `dist/` but will not
push to AUR (that only happens on a real tagged release).

## 5. First real release

The first tagged release after this setup will automatically create the
`ganda-bin` package on AUR. Subsequent releases update it in place.

```bash
just bump <version>
```
