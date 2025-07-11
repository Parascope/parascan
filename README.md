# Sitedog CLI

Universal CLI for sites/apps config preview & management that renders them as cards:

![sitedog-cards](https://i.imgur.com/JKjTgqY.png)

## Examples

Below are source configs for two of the above cards.

```yaml
painlessrails.com:
  dns: route 53
  hosting: hetzner
  ip: 121.143.73.32
  ssh access given to: inem, kirill

  repository: https://gitlab.com/rbbr_io/painlessrails
  registry: https://gitlab.com/rbbr_io/painlessrails/cloud/container_registry
  ci: https://gitlab.com/rbbr_io/painlessrails/ci
  issues: https://gitlab.com/rbbr_io/painlessrails/issues
  monitoring: https://appsignal.com/rbbr
  metabase: https://metabase.com
  i18n: https://poeditor.com
  analythics: https://plausible.io/
```

```yaml
inem.at:
  registrar: gandi
  dns: Route 53
  hosting: https://carrd.com
  mail: zoho
```

---

## 🛠️ How-to

1. Install the CLI (see below).
2. Run `sitedog sniff` to detect your stack and create sitedog.yml.
3. Use `sitedog serve` to preview your card locally while editing.
4. Generate an HTML card with `sitedog render`.

---

## 🚀 Installation

```sh
curl -sL https://get.sitedog.io | sh
```


## 📦 Usage

### 1. Detect your stack and create sitedog.yml

```sh
sitedog sniff                          # detect stack and create sitedog.yml
sitedog sniff ./my-project             # detect stack in directory and create config
```

### 2. Live card preview on localhost

Useful when editing config.
```sh
sitedog serve                    # serve current directory config
sitedog serve ./my-project       # serve specific directory
sitedog serve --port 3030        # serve on custom port
```

### 3. Send configuration to cloud

```sh
sitedog push                     # push current config (interactive auth)
SITEDOG_TOKEN=xxx sitedog push   # push with token (for CI/CD)
```

**For CI/CD:** Set `SITEDOG_TOKEN` environment variable to avoid interactive authentication. Get your token at [app.sitedog.io/tokens](https://app.sitedog.io/tokens).

### 4. Render html file with card locally
```sh
sitedog render                   # render to sitedog.html
sitedog render --output my-card.html  # render to custom file
```

### 5. CLI help

```sh
sitedog help
Usage: sitedog <command> <path(optional)>

Commands:
  sniff   Detect your stack and create sitedog.yml
  serve   Start live server with preview
  push    Send configuration to cloud
  render  Render HTML card

  logout  Remove authentication token
  version Show version
  help    Show this help message

Options for serve:
  --port PORT      Port to run server on (default: 8081)

Options for render:
  --output PATH    Path to output HTML file (default: sitedog.html)

Examples:
  sitedog sniff                          # detect stack and create sitedog.yml
  sitedog sniff ./my-project             # detect stack in directory and create config

  sitedog serve --port 3030
```

## 🚀 Uninstallation

```sh
curl -sL https://get.sitedog.io/uninstall | sh
```

If you want to upgrade to the latest version, just uninstall and install again.

---

## 📖 Read more and try it live

- [Sitedog Website](https://sitedog.io/)
- [Sitedog Live Editor](https://sitedog.io/editor#demo)

---

## 🐶 We eat our own dog food!

We use sitedog for our own projects and documentation. Here's our card deck:

[![sitedog-cli](https://i.imgur.com/DNhwj6T.png)](https://sitedog.io/live-editor.html#pako:eJytVVtv2zYUfvevIPLQN4m-zbENBGmG2m2GJVtvQfskUOKxxJYiFZLyJdiP3yElW8qQFfUwwICBc_3Odz4eWeGA6zwWejkgBJg9VEyBXJLCucouKTVpamJbsQxoZfQ3yJyltkmirKqoZIoLlVOuSyaUxSKKlZAVwKquCKvi3W4XnzxxpstjRvufaeWMlqH7sQGiar1YtpKstiKV0JVVcxWdzPElzyWD-DRDXGjreqV8DbQUOu8qQB23tgCpnZDO5tPLEUVwIi-cnyljSh26tGPRYPYoGS8DSAOV7sJy4SRLQ2VPY4KBR-pa2gaY00R5-n1-Lqwzh3NqUE8dcgQmOaaHWk5_B7UkPjrXedJGjzsfgX0lDNgluRgPx7NoiL_phYfkcwiruXC2wVWxHBJbAfDEj2hcB9C7gifeQRpz2FKmmDxYYWkIiVq4ESLXU1eJaivd03Uhr0BFb399tdGmTDYsc9pclToVEkLHXOtcArHATFYQHNHq_uobe9xEBXYaS9RGXuNcujYZJIJf2SxqZLTsqcHLfe8MI9bVm82y1zTxkLpWXGe238h7UbV09PTLw816u_4z2bwb8d8uv07L8t7dJTd2z9a72-_vP6zGw6f848Nwe0cBuQwtcBOiRMq6-v5lbEResuZVAIpOUWFuYfLHt0e5Wte32Wf1sJ6n727oZ-VQD8CvleYQ4WijaDZ85a4mXz-t8_tHab-8nd3fr1Q0DM1Yqmv3TEtF3WjpIxLxpllKCKKp1Cn1HNEPq5s3d6u45IMBPvDYPrsQJmwhk7rm58i9SzhD3yHpZXVbMFswROCNGU3m8WixiMeTSbyYo88_exQ6tgD3hHloKlGSOWo3xcan-zDwz4DrZvMRwVk980yGeCFb4Ru9FRzMkuTe2LAqBbOJw8kvXofzKPRFcHCFb6nc45SZNryJ9YeB_OOmBmNfUiw7XcMG6mCQg3uReoxxTEocvp3znDVg0XOXgCkvr-A_fCt8-9PB-8F96gf8-5HySPojWBf3BP4IsioOqd7TDYwv53wyGS3mQzbKYLjIJnwxnaXpZpjOJ-lgcLxQmRQ9jesSd8GJxMERipYvEX3s1rtxvWL_L0pyhNTe5FYJiLQ2kkT2967LM-2Qv4gtQkatfjaHnkKb7L8BYybMKw)

Even for such a small project there are already a lot of information that easily can be lost.

---

### What's next?

We believe that Sitedog should be a part of your workflow, not just a tool to show your config. Therefore it must be literally at hands. Current features and plans:

**✅ Available now:**
- Smart stack detection with `sitedog sniff` (detects languages, frameworks, services)
- Live preview server for real-time config editing
- Cloud sync for team collaboration
- HTML card generation for static sites

**🚀 Coming soon:**

- [x] [VS Code extension](https://github.com/sitedog-io/sitedog-vscode)
- [x] [Chrome extension](https://github.com/sitedog-io/sitedog-chrome) to augment GitLab/Github with sitedog cards
- [ ] GitLab CI example
- [x] [GitHub Actions example](https://github.com/sitedog-io/sitedog-cli/blob/main/.github/workflows/sitedog-cloud.yml)
- [ ] Git hooks example

---

## Cloud access (Beta)

Sitedog Cloud is a platform to manage your sites/apps configs.

[Request access](https://docs.google.com/forms/d/1z5VAFvFP_fH1dJ7Y4mmNtM_AsxaFwIkQRE20zgSV0vM/edit)

---

# Development process

```sh
# Show all created versions
make show-versions

# bump cli version and create a git tag
make bump-version v=v0.1.1

# if everything is okay, push tag to main triggering release pipeline
make push-version
```

