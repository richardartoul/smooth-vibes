# smooth

Version control for vibe coders. A friendly TUI and web interface for git.

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/YOUR_USERNAME/smooth/main/install.sh | sh
```

### Other methods

<details>
<summary>Using Go</summary>

```bash
go install github.com/YOUR_USERNAME/smooth@latest
```
</details>

<details>
<summary>From releases</summary>

Download the latest binary for your platform from the [releases page](https://github.com/YOUR_USERNAME/smooth/releases).
</details>

<details>
<summary>From source</summary>

```bash
git clone https://github.com/YOUR_USERNAME/smooth.git
cd smooth
go build -o smooth .
```
</details>

## Usage

```bash
# Start the TUI interface
smooth

# Start the web interface (http://localhost:3000)
smooth web

# Show help
smooth help
```

## Requirements

- Git must be installed and available in your PATH
- Run `smooth` from within a git repository

