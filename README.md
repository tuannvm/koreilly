# GOReily - O'Reilly Learning Downloader

GOReily is a command-line tool for downloading and managing O'Reilly Learning content. It provides a beautiful terminal interface for searching, downloading, and sending books to your Kindle.

## Features

- 🔐 Secure API key authentication
- 📚 Search and download O'Reilly Learning content
- 📖 Generate EPUB files
- ✉️ Send books directly to your Kindle
- 🎨 Beautiful terminal user interface
- ⚡ Fast and efficient downloads

## Installation

### Prerequisites

- Go 1.21 or later
- An O'Reilly Learning account with API access
- (Optional) Kindle email address for direct delivery

### Build from source

```bash
# Clone the repository
git clone https://github.com/yourusername/goreilly.git
cd goreilly

# Build the binary
go install github.com/tuannvm/goreilly/cmd/goreilly@latest

# Move the binary to your PATH (optional)
sudo mv goreilly /usr/local/bin/
```

## Usage

### First-time setup

1. Get your O'Reilly API key from [O'Reilly Learning](https://learning.oreilly.com/api/v2/)
2. Run the application:
   ```bash
   goreilly
   ```
3. Enter your API key when prompted
4. (Optional) Set up your Kindle email in the settings

### Commands

- `goreilly` - Start the interactive TUI
- `goreilly search <query>` - Search for books
- `goreilly download <book-id>` - Download a book
- `goreilly config` - Show or edit configuration

## Configuration

Configuration is stored in `~/.config/goreilly/config.yaml`. You can also use environment variables:

```bash
# Required
KOREILLY_API_KEY=your_api_key_here

# Optional
KOREILLY_OUTPUT_DIR=./books
KOREILLY_GMAIL_EMAIL=your.email@gmail.com
KOREILLY_KINDLE_EMAIL=your_kindle@kindle.com
```

## License

MIT

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## Disclaimer

This project is not affiliated with O'Reilly Media, Inc. Please respect O'Reilly's terms of service and only download content you have access to.