# Parascan CLI

Org-level dependencies scanner for your repos. Automatically detects technologies, frameworks, and services used in your projects.

## 🚀 Installation

```sh
curl -sL https://get.parascope.io | sh
```

## 📦 Usage

### Detect your stack and create parascope.yml

```sh
para scan                          # detect stack and create parascope.yml
para scan ./my-project             # detect stack in directory and create config
para scan --verbose                # show detailed detection process
para scan -v ./my-project          # verbose analysis of specific directory
```

### CLI help

```sh
para help
Usage: para <command> <path(optional)>

Commands:
  scan    Detect your stack and create parascope.yml
  help    Show this help message

Options for scan:
  --verbose, -v    Show detailed detection information

Examples:
  para scan                          # detect stack and create parascope.yml
  para scan ./my-project             # detect stack in directory and create config
  para scan --verbose                # show detailed detection process
  para scan -v ./my-project          # verbose analysis of specific directory
```

## 🚀 Uninstallation

```sh
curl -sL https://get.parascope.io/uninstall | sh
```

If you want to upgrade to the latest version, just uninstall and install again.

---

## 🔍 What it detects

Parascan automatically detects:

- **Languages**: Python, Node.js, Ruby, Go, Java, PHP, .NET, and more
- **Frameworks**: React, Vue, Angular, Django, Rails, Spring, and others
- **Services**: GitHub, GitLab, AWS, Firebase, Stripe, and 50+ other services
- **Package Managers**: npm, yarn, pip, composer, bundler, and more
- **Repositories**: Git repository URLs (with automatic credential sanitization)

## 📁 Output

Creates a `parascope.yml` file with detected services:

```yaml
my-project:
  Repository: https://github.com/user/repo
  React: https://reactjs.org
  Node.js: https://nodejs.org
  AWS: https://aws.amazon.com
  Stripe: https://stripe.com
```

## 🛠️ Development

```sh
# Build for all platforms
make build

# Install development version
make dev-install

# Run tests
make test

# Show all available commands
make help
```

## 📖 About

Parascan is a fork of [sitedog-cli](https://github.com/sitedog-io/sitedog-cli) focused on organization-level dependency scanning. It automatically detects and catalogs the technologies and services used across your repositories.

### Key Features

- **Smart Detection**: Automatically detects languages, frameworks, and services
- **Credential Safe**: Sanitizes Git URLs to remove access tokens
- **CI/CD Ready**: Works in automated environments
- **Cross-Platform**: Supports Linux, macOS, and Windows

---

## Development process

```sh
# Show all created versions
make show-versions

# bump cli version and create a git tag
make bump-version v=v0.1.1

# if everything is okay, push tag to main triggering release pipeline
make push-version
```