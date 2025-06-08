# Safari Books Golang Implementation Plan

## Project Overview

This document outlines the implementation plan for creating a Golang-based alternative to the Python safaribooks tool. The goal is to download and generate EPUB files from O'Reilly Learning (Safari Books Online) library with improved performance, better cross-platform compatibility, modern Go development practices, and a beautiful terminal user interface using Bubble Tea.

## Current Python Implementation Analysis

Based on the original repository, the current tool provides:
- Authentication with O'Reilly Learning platform
- Book metadata retrieval  
- Chapter content downloading
- CSS and image asset downloading
- EPUB generation
- SSO (Single Sign-On) support
- Session cookie management
- Kindle-compatible formatting options

## Architecture Design

### Core Components

1. **Authentication Module** (`auth/`)
   - API key authentication (just enter the O'Reilly API token)
   - Secure token storage using native `encoding/json`

2. **API Client** (`client/`)
   - Native `net/http` client with retry logic
   - Rate limiting using `golang.org/x/time/rate`
   - Bearer token (API key) authentication
   - Proxy support
   - Request/response handling

3. **Book Processing** (`book/`)
   - Metadata extraction using native `encoding/xml` and `net/html`
   - Chapter content retrieval
   - Asset downloading (CSS, images)
   - Content sanitization

4. **EPUB Generator** (`epub/`)
   - Native EPUB package creation using `archive/zip`
   - Manifest generation with `encoding/xml`
   - Navigation document creation
   - Content organization

5. **Gmail Delivery Service** (`gmail/`)
   - Gmail-based delivery to Kindle devices via "Send to Kindle" email service
   - Gmail SMTP integration with app password authentication for sending EPUBs
   - Email delivery status and error reporting

6. **TUI Interface** (`tui/`)
   - Bubble Tea based terminal user interface
   - Interactive book selection and download progress
   - Real-time status updates
   - Gmail delivery configuration and confirmation

7. **Configuration** (`config/`)
   - Native environment variable handling
   - JSON configuration file support using `encoding/json`
   - Default settings

## Directory Structure

```
koreilly/
├── cmd/
│   └── koreilly/
│       └── main.go
├── internal/
│   ├── auth/
│   │   ├── auth.go
│   │   ├── token.go
│   │   └── storage.go          # Secure token storage
│   ├── client/
│   │   ├── client.go
│   │   ├── retry.go
│   │   ├── ratelimit.go
│   │   └── middleware.go       # Request/response middleware
│   ├── services/               # Business logic layer
│   │   ├── book/
│   │   │   ├── service.go
│   │   │   ├── metadata.go
│   │   │   ├── chapter.go
│   │   │   └── assets.go
│   │   ├── epub/
│   │   │   ├── builder.go
│   │   │   ├── generator.go
│   │   │   └── templates.go
│   │   └── delivery/
│   │       ├── gmail.go
│   │       ├── smtp.go
│   │       └── validator.go    # Email validation
│   ├── tui/
│   │   ├── app.go
│   │   ├── state.go            # Application state management
│   │   ├── models/
│   │   │   ├── auth.go
│   │   │   ├── search.go
│   │   │   ├── download.go
│   │   │   ├── settings.go
│   │   │   └── gmail.go
│   │   ├── views/
│   │   │   ├── auth.go
│   │   │   ├── search.go
│   │   │   ├── download.go
│   │   │   ├── settings.go
│   │   │   └── gmail.go
│   │   ├── components/
│   │   │   ├── progress.go
│   │   │   ├── table.go
│   │   │   ├── form.go
│   │   │   └── notification.go # Toast notifications
│   │   └── styles/
│   │       ├── theme.go
│   │       └── colors.go
│   ├── config/
│   │   ├── config.go
│   │   ├── defaults.go         # Default configuration values
│   │   └── validation.go       # Config validation
│   └── utils/
│       ├── filesystem.go
│       ├── html.go
│       ├── validation.go
│       └── logger.go           # Structured logging
├── pkg/
│   ├── models/
│   │   ├── book.go
│   │   ├── chapter.go
│   │   ├── asset.go
│   │   ├── config.go
│   │   └── delivery.go         # Email delivery models
│   └── errors/
│       ├── errors.go           # Custom error types
│       └── codes.go            # Error codes
├── assets/                     # Embedded assets
│   ├── templates/
│   │   ├── epub/
│   │   └── email/
│   └── styles/
│       └── kindle.css
├── docs/
│   ├── api.md                  # O'Reilly API documentation
│   ├── setup.md               # Setup guide
│   └── troubleshooting.md
├── scripts/
│   ├── build.sh
│   ├── test.sh
│   └── release.sh
├── testdata/
│   ├── books/                  # Sample book data
│   ├── responses/              # Mock API responses
│   └── configs/                # Test configurations
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
├── .goreleaser.yml
└── README.md
```

## Implementation Phases

### Phase 1: Core Infrastructure (Week 1-2)

#### 1.1 Project Setup
- Initialize Go module with minimal dependencies
- Set up CI/CD with GitHub Actions
- Configure native Go tools (go vet, go fmt, golint)
- Create basic project structure

#### 1.2 Configuration Management (Native)
```go
type BookConfig struct {
    APIToken        string          `json:"api_token"`
    OutputDir       string          `json:"output_dir"`
    KindleMode      bool            `json:"kindle_mode"`
    PreserveLog     bool            `json:"preserve_log"`
    ProxyURL        string          `json:"proxy_url"`
    UserAgent       string          `json:"user_agent"`
    MaxRetries      int             `json:"max_retries"`
    RequestDelay    time.Duration   `json:"request_delay"`
    MaxConcurrent   int             `json:"max_concurrent"`
    EmailDelivery   EmailConfig     `json:"email_delivery"`
}

type EmailConfig struct {
    Enabled         bool         `json:"enabled"`
    Email           string       `json:"email"`               // Gmail account email
    AppPassword     string       `json:"app_password"`       // Gmail app password (not regular password)
    SMTPServer      string       `json:"smtp_server"`        // smtp.gmail.com
    SMTPPort        int          `json:"smtp_port"`          // 587
    Recipients      []KindleConfig  `json:"recipients"`
    Subject         string          `json:"subject"`
}

type KindleConfig struct {
    Name            string `json:"name"`               // Friendly name (e.g., "My Kindle")
    Email           string `json:"email"`              // Target email (@kindle.com or any email)
    Type            string `json:"type"`               // "kindle", "email", etc.
    Default         bool   `json:"default"`            // Default recipient
}

// Using native os and encoding/json
func (c *BookConfig) Load() error
func (c *BookConfig) Save() error
func (c *BookConfig) LoadFromEnv() error
func (c *BookConfig) ValidateEmailConfig() error
func (c *BookConfig) GetDefaultRecipient() *KindleConfig
func (c *BookConfig) AddRecipient(recipient KindleConfig) error
```

#### 1.3 HTTP Client Foundation (Native)
```go
type Client struct {
    httpClient   *http.Client
    baseURL      string
    apiToken     string
    rateLimiter  *rate.Limiter
    userAgent    string
    maxRetries   int
    proxy        *url.URL
}

// Using native net/http with Bearer token authentication
func NewClient(config *BookConfig) (*Client, error)
func (c *Client) Do(req *http.Request) (*http.Response, error)
func (c *Client) DoWithRetry(req *http.Request) (*http.Response, error)
func (c *Client) addAuthHeaders(req *http.Request)
func (c *Client) SetAPIToken(token string)
```

### Phase 2: Authentication System (Week 2-3)

#### 2.1 API Token Authentication (Native)
```go
type AuthService struct {
    client   *Client
    apiToken string
    config   *BookConfig
}

// Using native net/http with Bearer token authentication
func (a *AuthService) SetAPIToken(token string) error
func (a *AuthService) ValidateToken() error
func (a *AuthService) GetAuthHeaders() map[string]string
func (a *AuthService) IsAuthenticated() bool
```

#### 2.2 Token Management (Native)
```go
// Using native os and encoding/json for secure token storage
func (a *AuthService) LoadToken() error
func (a *AuthService) SaveToken() error
func (a *AuthService) ClearToken() error
func (a *AuthService) RefreshTokenIfNeeded() error

// Token validation against O'Reilly API
func (a *AuthService) validateTokenWithAPI() error
```

### Phase 3: Bubble Tea TUI Foundation (Week 2-3)

#### 3.1 Main Application Model
```go
type App struct {
    state      AppState
    width      int
    height     int
    config     *BookConfig
    authModel  *AuthModel
    searchModel *SearchModel
    downloadModel *DownloadModel
    settingsModel *SettingsModel
    emailModel *EmailModel
}

type AppState int

const (
    StateAuth AppState = iota
    StateSearch
    StateDownload
    StateSettings
    StateEmail
    StateHelp
)

func (a App) Init() tea.Cmd
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (a App) View() string
```

#### 3.2 Authentication Model
```go
type AuthModel struct {
    tokenInput textinput.Model
    err        error
    loading    bool
    spinner    spinner.Model
    validated  bool
    helpText   string
}

func NewAuthModel() *AuthModel {
    input := textinput.New()
    input.Placeholder = "Enter your O'Reilly Learning API token..."
    input.Focus()
    input.CharLimit = 256
    input.Width = 50
    
    return &AuthModel{
        tokenInput: input,
        helpText: "Get your API token from: learning.oreilly.com/profile/api-keys",
    }
}

func (m AuthModel) Update(msg tea.Msg) (*AuthModel, tea.Cmd)
func (m AuthModel) View() string
func (m AuthModel) ValidateToken() tea.Cmd
```

#### 3.3 Download Progress Model
```go
type DownloadModel struct {
    progress   progress.Model
    chapters   []Chapter
    current    int
    total      int
    status     string
    logs       []string
    done       bool
    err        error
}

func NewDownloadModel() *DownloadModel
func (m DownloadModel) Update(msg tea.Msg) (*DownloadModel, tea.Cmd)
func (m DownloadModel) View() string
```

#### 3.4 Email Configuration Model
```go
type EmailModel struct {
    inputs       []textinput.Model  // [gmail_email, app_password, recipient_email, recipient_name]
    focused      int
    testResult   string
    testing      bool
    spinner      spinner.Model
    config       *EmailConfig
    recipients   []KindleConfig
    selectedRecipient int
}

func NewEmailModel(config *EmailConfig) *EmailModel
func (m EmailModel) Update(msg tea.Msg) (*EmailModel, tea.Cmd)
func (m EmailModel) View() string
func (m EmailModel) TestEmailConnection() tea.Cmd
func (m EmailModel) AddRecipient() tea.Cmd
func (m EmailModel) RemoveRecipient(index int) tea.Cmd
```

### Phase 4: Book Discovery and Metadata (Week 3-4)

#### 4.1 Book Models (Native)
```go
type Book struct {
    ID           string    `json:"id"`
    Title        string    `json:"title"`
    Authors      []string  `json:"authors"`
    ISBN         string    `json:"isbn"`
    Publisher    string    `json:"publisher"`
    Description  string    `json:"description"`
    ReleaseDate  time.Time `json:"release_date"`
    URL          string    `json:"url"`
    Chapters     []Chapter `json:"chapters"`
    Assets       Assets    `json:"assets"`
}

type Chapter struct {
    ID       string `json:"id"`
    Title    string `json:"title"`
    URL      string `json:"url"`
    Content  string `json:"-"`
    Order    int    `json:"order"`
    FilePath string `json:"file_path"`
}

type Assets struct {
    CSS    []Asset `json:"css"`
    Images []Asset `json:"images"`
}

type Asset struct {
    URL      string `json:"url"`
    Filename string `json:"filename"`
    Content  []byte `json:"-"`
    FilePath string `json:"file_path"`
}
```

#### 4.2 Book Service (Native HTML parsing)
```go
type BookService struct {
    client *Client
}

// Using net/html for parsing
func (b *BookService) GetBookInfo(bookID string) (*Book, error)
func (b *BookService) GetChapters(bookID string) ([]Chapter, error)
func (b *BookService) DownloadChapter(chapter *Chapter) error
func (b *BookService) DownloadAssets(book *Book) error
func (b *BookService) SearchBooks(query string) ([]Book, error)
```

### Phase 5: Content Processing (Week 4-5)

#### 5.1 HTML Processing (Native)
```go
type ContentProcessor struct {
    kindleMode bool
}

// Using net/html and strings packages
func (c *ContentProcessor) SanitizeHTML(content string) string
func (c *ContentProcessor) ExtractImages(content string) []string
func (c *ContentProcessor) ApplyKindleCSS(content string) string
func (c *ContentProcessor) FixRelativeLinks(content string) string
func (c *ContentProcessor) ParseHTML(htmlContent string) (*html.Node, error)
func (c *ContentProcessor) RenderHTML(node *html.Node) string
```

#### 5.2 Asset Management (Native)
```go
type AssetManager struct {
    client    *Client
    outputDir string
}

// Using native io, os, path/filepath
func (a *AssetManager) DownloadCSS(urls []string) error
func (a *AssetManager) DownloadImages(urls []string) error
func (a *AssetManager) OptimizeImages() error
func (a *AssetManager) CreateAssetDirectory(bookID string) error
```

### Phase 6: EPUB Generation (Week 5-6)

#### 6.1 EPUB Builder (Native)
```go
type EPUBBuilder struct {
    book      *Book
    outputDir string
    kindleMode bool
}

// Using archive/zip, encoding/xml, html/template
func (e *EPUBBuilder) Create() error
func (e *EPUBBuilder) generateManifest() error
func (e *EPUBBuilder) generateTableOfContents() error
func (e *EPUBBuilder) packageEPUB() error
func (e *EPUBBuilder) createMimeType() error
func (e *EPUBBuilder) createContainer() error
func (e *EPUBBuilder) createOPF() error
func (e *EPUBBuilder) createNCX() error
```

#### 6.2 Template System (Native)
```go
// Using html/template and text/template
var (
    containerTemplate = template.Must(template.New("container").Parse(containerXML))
    opfTemplate      = template.Must(template.New("opf").Parse(opfXML))
    ncxTemplate      = template.Must(template.New("ncx").Parse(ncxXML))
    chapterTemplate  = template.Must(template.New("chapter").Parse(chapterHTML))
)

const containerXML = `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`
```

### Phase 7: Email Delivery Integration and TUI Polish (Week 6-7)

#### 7.1 Email Delivery Service Integration
- Implement SMTP or Gmail API for sending emails to Kindle addresses
- Email delivery with EPUB attachments to "Send to Kindle" email addresses
- Track and report email delivery status to the TUI

#### 7.2 Enhanced Download Progress
- Real-time download progress with email delivery status
- Delivery confirmation and error handling via email status
- Integrated workflow from download to Kindle email delivery

#### 7.3 Final TUI Polish
- Improved layout, shortcuts, and error displays
- Refined loading animations and notifications
- Persistent settings, search state, and delivery logs
- Consistent styling and theming
- Keyboard shortcuts and navigation improvements
- Help screens and user guidance
- Settings persistence and management

## Updated Implementation Timeline

### **Week 1: Foundation & Core Services**
- Project setup with Go modules and tooling
- Configuration management with layered approach (file + env + validation)
- Service layer architecture with interfaces and dependency injection
- HTTP client with connection pooling, middleware, and retry logic
- Error handling with structured error types and user-friendly messages
- API token authentication service with secure storage
- Basic TUI shell with Bubble Tea and centralized state management

### **Week 2: Book Services & Enhanced TUI**
- O'Reilly API integration with context support and rate limiting
- Book search and metadata extraction with input validation
- TUI models for search and navigation with loading states
- Comprehensive error handling with retry logic and user feedback
- Progress tracking infrastructure with persistence
- Unit tests for core services with mock interfaces

### **Week 3: Download System & EPUB Generation**
- Worker pool-based concurrent download system with memory management
- Chapter and asset downloading with resume capability and progress tracking
- Streaming EPUB generation to handle large books efficiently
- Download progress UI with ETA, cancellation, and error recovery
- State persistence for interrupted downloads with automatic resume
- Performance optimization: connection pooling and resource management

### **Week 4: Gmail Integration & Polish**
- Gmail SMTP service with comprehensive validation and error handling
- Email composition with EPUB attachments and size limits
- Gmail configuration UI with connection testing and credential validation
- Security hardening: input sanitization, rate limiting, secure storage
- Final TUI polish: themes, keyboard shortcuts, help system, accessibility
- Integration tests for email delivery and end-to-end workflows

### **Week 5: Quality & Release (Optional)**
- Security audit: vulnerability scanning, penetration testing, OWASP compliance
- Performance optimization: memory profiling, CPU profiling, load testing  
- Cross-platform testing and compatibility (macOS, Linux, Windows)
- Release preparation with GoReleaser and automated CI/CD pipelines
- Comprehensive documentation: setup guides, troubleshooting, API docs
- Code quality tools: linting, static analysis, test coverage reporting

**Total Development Time**: 4-5 weeks for production-ready CLI tool with enterprise-grade features.

## Summary of Key Improvements

1. **Better Architecture**: Service layer, dependency injection, interfaces
2. **Enhanced Security**: Secure storage, input validation, rate limiting
3. **Improved UX**: Progress tracking with ETA, resume downloads, themes
4. **Performance**: Connection pooling, worker pools, streaming EPUB generation
5. **Reliability**: Context cancellation, graceful shutdown, error recovery
6. **Maintainability**: Structured logging, metrics, comprehensive testing
7. **Professional Polish**: CI/CD, documentation, cross-platform support

This approach ensures you build a robust, maintainable, and user-friendly tool that can handle real-world usage scenarios effectively.

## Dependencies

### Core Dependencies (Minimal External)
```go
// Bubble Tea TUI framework
"github.com/charmbracelet/bubbletea"
"github.com/charmbracelet/lipgloss"
"github.com/charmbracelet/bubbles/textinput"
"github.com/charmbracelet/bubbles/spinner"
"github.com/charmbracelet/bubbles/progress"
"github.com/charmbracelet/bubbles/table"

// Email delivery (Gmail SMTP)
"net/smtp"
"crypto/tls"
```

// Rate limiting (only external networking dependency)
"golang.org/x/time/rate"

// Native Go libraries
"net/http"
"net/url"
"net/html"
"encoding/json"
"encoding/xml"
"html/template"
"text/template"
"archive/zip"
"mime"
"mime/multipart"
"mime/quotedprintable"
"io"
"os"
"path/filepath"
"strings"
"time"
"sync"
"context"
"bufio"
"bytes"
"fmt"
"log"
"errors"
"strconv"
"regexp"
"encoding/base64"
"crypto/rand"
"runtime"
"os/exec"
```

## Configuration

### Environment Variables
```bash
# O'Reilly Authentication
KOREILLY_API_TOKEN=""

# Download Settings  
KOREILLY_OUTPUT_DIR="./books"
KOREILLY_MAX_CONCURRENT="5"
KOREILLY_PROXY=""
KOREILLY_USER_AGENT="KOReilly/1.0"
KOREILLY_MAX_RETRIES="3"
KOREILLY_REQUEST_DELAY="1s"

# Email Delivery Configuration (Gmail only)
KOREILLY_EMAIL_ENABLED="false"
KOREILLY_EMAIL_ADDRESS=""
KOREILLY_EMAIL_APP_PASSWORD=""
KOREILLY_DEFAULT_RECIPIENT="your-username@kindle.com"
```

### Configuration File (koreilly.json)
```json
{
  "auth": {
    "api_token": ""
  },
  "download": {
    "output_dir": "./books",
    "kindle_mode": false,
    "preserve_log": false,
    "max_concurrent": 5,
    "request_delay": "1s"
  },
  "network": {
    "proxy": "",
    "user_agent": "KOReilly/1.0",
    "max_retries": 3,
    "timeout": "30s"
  },
  "email_delivery": {
    "enabled": false,
    "email": "",
    "app_password": "",
    "smtp_server": "smtp.gmail.com",
    "smtp_port": 587,
    "recipients": [
      {
        "name": "My Kindle",
        "email": "your-username@kindle.com",
        "type": "kindle",
        "default": true
      }
    ],
    "subject": "{{.Title}} - O'Reilly Book"
  },
  "ui": {
    "theme": "default",
    "show_help": true,
    "auto_save_settings": true
  }
}
```

## User Setup Guide

### Gmail Integration Setup

To enable automatic book delivery to your Kindle device via email, you'll need to configure Gmail SMTP authentication and set up your Kindle to receive emails from your Gmail account.

#### Step 1: Create a Gmail App Password

1. **Go to your Google Account settings**
   - Visit [myaccount.google.com](https://myaccount.google.com)
   - Sign in to your Gmail account

2. **Enable 2-Factor Authentication** (required for app passwords)
   - Go to "Security" in the left sidebar
   - Under "Signing in to Google", select "2-Step Verification"
   - Follow the setup instructions if not already enabled

3. **Generate an App Password**
   - Still in the "Security" section, select "App passwords"
   - You may need to sign in again
   - Select "Mail" from the "Select app" dropdown
   - Select "Other (Custom name)" from the "Select device" dropdown
   - Enter "KOReilly CLI" as the custom name
   - Click "Generate"
   - **Copy the 16-character password** (it will look like: `abcd efgh ijkl mnop`)

4. **Save the App Password**
   - Use this app password in your KOReilly configuration
   - **Never use your regular Gmail password** - only use the generated app password
   - Store it securely as it won't be shown again

#### Step 2: Configure Your Kindle Email Whitelist

1. **Find your Kindle email address**
   - On your Kindle: Go to Settings → Device Options → Device Email
   - It will look like: `your-username@kindle.com`

2. **Add your Gmail to Kindle's approved list**
   - Visit [Amazon's Manage Your Content and Devices](https://www.amazon.com/mn/dcw/myx.html)
   - Sign in to your Amazon account
   - Go to the "Preferences" tab
   - Find "Personal Document Settings"
   - Under "Approved Personal Document E-mail List", click "Add a new approved e-mail address"
   - Enter your Gmail address (the one you'll use with KOReilly)
   - Click "Add Address"

3. **Configure delivery preferences** (optional)
   - In the same section, you can set whether documents are delivered via Wi-Fi or also via cellular
   - You can also set up automatic conversion of documents

#### Step 3: Configure KOReilly

Update your configuration file or environment variables:

**Environment Variables:**
```bash
export KOREILLY_EMAIL_ENABLED="true"
export KOREILLY_EMAIL_ADDRESS="your-email@gmail.com"
export KOREILLY_EMAIL_APP_PASSWORD="abcd efgh ijkl mnop"  # Use the app password from Step 1
export KOREILLY_DEFAULT_RECIPIENT="your-username@kindle.com"  # From Step 2
```

**Configuration File (koreilly.json):**
```json
{
  "email_delivery": {
    "enabled": true,
    "email": "your-email@gmail.com",
    "app_password": "abcd efgh ijkl mnop",
    "smtp_server": "smtp.gmail.com",
    "smtp_port": 587,
    "recipients": [
      {
        "name": "My Kindle",
        "email": "your-username@kindle.com",
        "type": "kindle",
        "default": true
      }
    ]
  }
}
```

#### Step 4: Test the Setup

1. **Test email delivery in KOReilly:**
   - Use the TUI's email setup section to test connectivity
   - Send a test email to verify the configuration works

2. **Check your Kindle:**
   - Downloaded books should appear in your Kindle library within a few minutes
   - Books are typically delivered to the "Documents" section

**Common Issues:**

* **"Authentication failed" error:**
	+ Verify you're using the app password, not your regular Gmail password
	+ Ensure 2-Factor Authentication is enabled on your Google account
	+ Double-check the app password was copied correctly (remove any spaces)
* **Emails not appearing on Kindle:**
	+ Verify your Gmail address is in the Kindle approved email list
	+ Check the Kindle email address is correct in your configuration
	+ Ensure your Kindle is connected to Wi-Fi
* **"Invalid recipients" error:**
	+ Verify the Kindle email format: `username@kindle.com`
	+ Check for typos in the email address
	+ Ensure the recipient is marked as "kindle" type in configuration
* **SMTP connection issues:**
	+ Verify SMTP settings: `smtp.gmail.com:587`
	+ Check your internet connection
	+ Some corporate networks may block SMTP ports

## TUI User Experience

### Main Interface Flow
```
┌─────────────────────────────────────────────────────────────┐
│                      KOReilly v1.0                         │
├─────────────────────────────────────────────────────────────┤
│  [T]oken [S]earch [D]ownload [K]indle Setup [Q]uit         │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Status: ● Connected to O'Reilly Learning                  │
│  API Token: ●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●abcd        │
│  Output: ./books/                                           │
│  Email Delivery: user@gmail.com → your-username@kindle.com │
│                                                             │
│  Recent Downloads:                                          │
│  ✓ Python Crash Course (9781593279288) 📧 Sent            │
│  ✓ Clean Code (9780136083238) 📧 Sent                     │
│  ⏳ Learning Go (9781492077213) - 45% complete              │
│                                                             │
│  [Enter] to continue download, [Tab] to navigate           │
└─────────────────────────────────────────────────────────────┘
```

### API Token Authentication Interface
```
┌─────────────────────────────────────────────────────────────┐
│                 O'Reilly API Authentication                │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Enter your O'Reilly Learning API Token:                   │
│                                                             │
│  Token: ▌                                                  │
│                                                             │
│  💡 How to get your API token:                             │
│  1. Go to learning.oreilly.com                             │
│  2. Sign in to your account                                 │
│  3. Visit Profile → API Keys                               │
│  4. Create a new API key                                    │
│  5. Copy and paste it above                                 │
│                                                             │
│  🔐 Your token is stored securely locally                  │
│                                                             │
│  [Enter] Validate [H]elp [Q]uit                            │
└─────────────────────────────────────────────────────────────┘
```

### Token Validation Interface
```
┌─────────────────────────────────────────────────────────────┐
│                   Validating API Token                     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│                    🔄 Validating...                        │
│                                                             │
│  Checking token with O'Reilly Learning API...              │
│                                                             │
│  • Connecting to api.oreilly.com...                  ✓     │
│  • Verifying token authenticity...               ⏳       │
│  • Checking access permissions...                          │
│                                                             │
│  This may take a few seconds...                            │
│                                                             │
│  [Esc] Cancel                                              │
└─────────────────────────────────────────────────────────────┘
```

### Authentication Success Interface
```
┌─────────────────────────────────────────────────────────────┐
│                  Authentication Successful                 │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│                       ✅ Success!                          │
│                                                             │
│  Your API token has been validated and saved.              │
│                                                             │
│  📚 Account Details:                                       │
│  • Subscription: O'Reilly Learning Premium                 │
│  • Access Level: Full Library                              │
│  • Books Available: 50,000+                               │
│  • Valid Until: 2024-12-31                                │
│                                                             │
│  You can now search and download books from the           │
│  O'Reilly Learning library.                               │
│                                                             │
│  [Enter] Continue [T]est Connection [Q]uit                 │
└─────────────────────────────────────────────────────────────┘
```

### Kindle Setup Instructions Help Panel
```
┌─────────────────────────────────────────────────────────────┐
│                Kindle Setup Instructions                   │
├─────────────────────────────────────────────────────────────┤
│ To send books to your Kindle, you need to:                 │
│                                                            │
│ 1. Create an "App Password" for your email account:        │
│    - For Gmail:                                            │
│      a. Visit https://myaccount.google.com/security        │
│      b. Enable 2-Step Verification                        │
│      c. Under "App passwords", create one for KOReilly     │
│    - For other providers, check their help docs.           │
│                                                            │
│ 2. Add your sending email to your Kindle whitelist:        │
│    - Go to https://amazon.com/myk                          │
│    - Settings → "Approved Personal Document E-mail List"   │
│    - Add your email (e.g., user@gmail.com)                 │
│                                                            │
│ 💡 Press [H] in Kindle Setup for these instructions        │
│                                                            │
│ [Enter] Back [Q] Quit                                      │
└─────────────────────────────────────────────────────────────┘
```

## Development Best Practices

### 1. Testing Strategy
```go
// Table-driven tests for comprehensive coverage
func TestBookService_Search(t *testing.T) {
    tests := []struct {
        name     string
        query    string
        expected []models.Book
        wantErr  bool
    }{
        // Test cases here
    }
}

// Mock interfaces for isolated unit tests
type MockBookService struct {
    SearchFunc func(ctx context.Context, query string) ([]models.Book, error)
}

// Integration tests with test containers
func TestDownloadFlow(t *testing.T) {
    // Set up test environment
    // Run full download flow
    // Verify EPUB generation
    // Test email delivery
}
```

### 2. Code Quality Tools
```makefile
# Makefile targets for development
.PHONY: lint test build release

lint:
	golangci-lint run ./...
	govulncheck ./...

test:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

build:
	go build -ldflags "-X main.version=$(VERSION)" ./cmd/koreilly

release:
	goreleaser release --rm-dist
```

### 3. CI/CD Pipeline
```yaml
# .github/workflows/ci.yml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: make lint
      - run: make test
      - run: make build
```

### 4. Documentation Strategy
- **API Documentation**: Use Go doc comments
- **User Guide**: Markdown with screenshots
- **Architecture Docs**: Decision records (ADRs)
- **Troubleshooting**: Common issues and solutions
## Architectural Best Practices & Recommendations

### **1. Service Layer Architecture**
Implement a clean service layer pattern to separate business logic from infrastructure concerns:

```go
// internal/services/interfaces.go
type BookService interface {
    Search(ctx context.Context, query string, opts SearchOptions) ([]models.Book, error)
    GetMetadata(ctx context.Context, bookID string) (*models.Book, error)
    DownloadChapter(ctx context.Context, chapterURL string) (*models.Chapter, error)
}

type EPUBService interface {
    Generate(ctx context.Context, book *models.Book, chapters []models.Chapter) (*models.EPUB, error)
    Save(ctx context.Context, epub *models.EPUB, outputPath string) error
}

type DeliveryService interface {
    ValidateConfig(config *EmailConfig) error
    SendEPUB(ctx context.Context, epub *models.EPUB, recipients []KindleConfig) error
    TestConnection(ctx context.Context, config *EmailConfig) error
}
```

### **2. Configuration Management Best Practices**
- **Layered Configuration**: Environment variables override config file values
- **Validation**: Validate all configuration at startup
- **Secrets Management**: Never log sensitive data (API tokens, passwords)
- **Default Values**: Provide sensible defaults for all optional settings

```go
// internal/config/loader.go
func Load() (*BookConfig, error) {
    cfg := NewDefaultConfig()
    
    // Layer 1: Load from config file
    if err := cfg.LoadFromFile(); err != nil && !os.IsNotExist(err) {
        return nil, fmt.Errorf("loading config file: %w", err)
    }
    
    // Layer 2: Override with environment variables
    if err := cfg.LoadFromEnv(); err != nil {
        return nil, fmt.Errorf("loading env vars: %w", err)
    }
    
    // Layer 3: Validate final configuration
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }
    
    return cfg, nil
}
```

### **3. Error Handling Strategy**
- **Structured Errors**: Use custom error types with context
- **Error Wrapping**: Preserve error chains for debugging
- **User-Friendly Messages**: Show actionable error messages in TUI
- **Retry Logic**: Implement exponential backoff for network operations

```go
// pkg/errors/errors.go
type ErrType string

const (
    ErrTypeAuth       ErrType = "authentication"
    ErrTypeNetwork    ErrType = "network"
    ErrTypeValidation ErrType = "validation"
    ErrTypeEmail      ErrType = "email_delivery"
)

type AppError struct {
    Type    ErrType
    Code    string
    Message string
    Err     error
    Context map[string]interface{}
}

func (e *AppError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
    }
    return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func NewAuthError(message string, err error) *AppError {
    return &AppError{
        Type:    ErrTypeAuth,
        Message: message,
        Err:     err,
    }
}
```

### **4. TUI State Management**
- **Centralized State**: Use a single source of truth for application state
- **Immutable Updates**: Update state through pure functions
- **Persistence**: Save critical state to prevent data loss
- **Loading States**: Show progress for all async operations

```go
// internal/tui/state/manager.go
type AppState struct {
    CurrentView    ViewType
    Auth          *AuthState
    Search        *SearchState
    Download      *DownloadState
    EmailDelivery *EmailState
    Settings      *SettingsState
}

type StateManager struct {
    state    *AppState
    mu       sync.RWMutex
    persist  PersistenceLayer
}

func (sm *StateManager) UpdateAsync(ctx context.Context, updater func(*AppState) *AppState) {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    newState := updater(sm.state)
    sm.state = newState
    
    // Persist critical state changes
    go sm.persist.Save(ctx, newState)
}
```

### **5. Security Recommendations**
- **Input Validation**: Sanitize all user inputs and API responses
- **Rate Limiting**: Protect against API abuse
- **Secure Storage**: Encrypt sensitive data at rest
- **Network Security**: Use TLS for all communications

```go
// internal/security/validator.go
func ValidateBookID(bookID string) error {
    if len(bookID) == 0 {
        return errors.New("book ID cannot be empty")
    }
    
    // Only allow alphanumeric and specific characters
    if !regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`).MatchString(bookID) {
        return errors.New("book ID contains invalid characters")
    }
    
    return nil
}

func SanitizeFilename(name string) string {
    // Remove or replace dangerous characters
    safe := regexp.MustCompile(`[<>:"/\\|?*]`).ReplaceAllString(name, "_")
    
    // Limit length to prevent filesystem issues
    if len(safe) > 255 {
        safe = safe[:255]
    }
    
    return safe
}
```

### **6. Performance Optimization**
- **Connection Pooling**: Reuse HTTP connections
- **Worker Pools**: Limit concurrent operations
- **Memory Management**: Stream large files instead of loading into memory
- **Caching**: Cache metadata and frequently accessed data

```go
// internal/client/pool.go
type ConnectionPool struct {
    client *http.Client
    pool   *sync.Pool
}

func NewConnectionPool(maxConnections int) *ConnectionPool {
    transport := &http.Transport{
        MaxIdleConns:        maxConnections,
        MaxIdleConnsPerHost: maxConnections / 4,
        IdleConnTimeout:     90 * time.Second,
        DisableCompression:  false,
    }
    
    return &ConnectionPool{
        client: &http.Client{
            Transport: transport,
            Timeout:   30 * time.Second,
        },
    }
}
```

### **7. Testing Strategy**
- **Unit Tests**: Test individual components in isolation
- **Integration Tests**: Test component interactions
- **End-to-End Tests**: Test complete user workflows
- **Mock Services**: Use interfaces for dependency injection

```go
// internal/services/book/service_test.go
func TestBookService_Search(t *testing.T) {
    tests := []struct {
        name     string
        query    string
        mockResp string
        want     []models.Book
        wantErr  bool
    }{
        {
            name:     "successful search",
            query:    "golang",
            mockResp: `{"results": [{"id": "123", "title": "Go Programming"}]}`,
            want:     []models.Book{{ID: "123", Title: "Go Programming"}},
            wantErr:  false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            client := &MockHTTPClient{
                DoFunc: func(req *http.Request) (*http.Response, error) {
                    return &http.Response{
                        StatusCode: 200,
                        Body:       io.NopCloser(strings.NewReader(tt.mockResp)),
                    }, nil
                },
            }
            
            service := NewBookService(client)
            got, err := service.Search(context.Background(), tt.query, SearchOptions{})
            
            if (err != nil) != tt.wantErr {
                t.Errorf("Search() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("Search() = %v, want %v", got, tt.want)
            }
        })
    }
}
```
