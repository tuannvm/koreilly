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
   - API token authentication
   - Token validation and management
   - Secure token storage using native `encoding/json`

2. **API Client** (`client/`)
   - Native `net/http` client with retry logic
   - Rate limiting using `golang.org/x/time/rate`
   - Bearer token authentication
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

5. **Gmail Service** (`gmail/`)
   - Gmail OAuth2 authentication with environment detection
   - Email composition with EPUB attachments
   - Kindle email delivery support
   - Multiple authentication methods (OAuth2 browser, manual, app password)

6. **TUI Interface** (`tui/`)
   - Bubble Tea based terminal user interface
   - Interactive book selection and download progress
   - Real-time status updates
   - Gmail configuration and email delivery confirmation

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
│   │   └── token.go
│   ├── client/
│   │   ├── client.go
│   │   ├── retry.go
│   │   └── ratelimit.go
│   ├── book/
│   │   ├── book.go
│   │   ├── metadata.go
│   │   ├── chapter.go
│   │   └── assets.go
│   ├── epub/
│   │   ├── epub.go
│   │   ├── generator.go
│   │   └── templates.go
│   ├── gmail/
│   │   ├── gmail.go
│   │   ├── oauth2.go
│   │   ├── environment.go
│   │   └── templates.go
│   ├── tui/
│   │   ├── app.go
│   │   ├── models/
│   │   │   ├── auth.go
│   │   │   ├── download.go
│   │   │   ├── search.go
│   │   ├── settings.go
│   │   └── gmail.go
│   │   ├── views/
│   │   │   ├── auth.go
│   │   │   ├── download.go
│   │   │   ├── search.go
│   │   │   ├── settings.go
│   │   │   └── gmail.go
│   │   └── components/
│   │       ├── progress.go
│   │       ├── table.go
│   │       └── form.go
│   ├── config/
│   │   └── config.go
│   └── utils/
│       ├── filesystem.go
│       ├── html.go
│       └── validation.go
├── pkg/
│   └── models/
│       ├── book.go
│       ├── chapter.go
│       └── gmail.go
├── docs/
├── scripts/
├── testdata/
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
type Config struct {
    APIToken        string        `json:"api_token"`
    OutputDir       string        `json:"output_dir"`
    KindleMode      bool          `json:"kindle_mode"`
    PreserveLog     bool          `json:"preserve_log"`
    ProxyURL        string        `json:"proxy_url"`
    UserAgent       string        `json:"user_agent"`
    MaxRetries      int           `json:"max_retries"`
    RequestDelay    time.Duration `json:"request_delay"`
    MaxConcurrent   int           `json:"max_concurrent"`
    GmailConfig     EmailConfig   `json:"gmail_config"`
}

type EmailConfig struct {
    GmailEmail       string `json:"gmail_email"`
    KindleEmail      string `json:"kindle_email"`
    SendToKindle     bool   `json:"send_to_kindle"`
    EmailSubject     string `json:"email_subject"`
    TokenFile        string `json:"token_file"`
    CredentialsFile  string `json:"credentials_file"`
}

// Using native os and encoding/json
func (c *Config) Load() error
func (c *Config) Save() error
func (c *Config) LoadFromEnv() error
func (c *Config) ValidateGmailConfig() error
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
func NewClient(config *Config) (*Client, error)
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
    config   *Config
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
    config     *Config
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

#### 3.4 Gmail Configuration Model
```go
type GmailModel struct {
    inputs       []textinput.Model
    focused      int
    testResult   string
    testing      bool
    spinner      spinner.Model
    config       *EmailConfig
    authMethod   GmailAuthMethod
    environment  *Environment
}

func NewGmailModel(config *EmailConfig) *GmailModel
func (m GmailModel) Update(msg tea.Msg) (*GmailModel, tea.Cmd)
func (m GmailModel) View() string
func (m GmailModel) TestGmailConnection() tea.Cmd
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

### Phase 7: Gmail Integration and TUI Polish (Week 6-7)

#### 7.1 Gmail Service Integration
- Gmail OAuth2 authentication with environment detection
- Email composition with EPUB attachments
- Multiple authentication methods (browser, manual, app password)
- Kindle delivery after successful download

#### 7.2 Enhanced Download Progress
- Real-time download progress with Gmail delivery status
- Email delivery confirmation and error handling
- Integrated workflow from download to Kindle delivery

#### 7.3 Final TUI Polish
- Consistent styling and theming
- Keyboard shortcuts and navigation improvements
- Help screens and user guidance
- Settings persistence and management

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

// Gmail API (only external API dependency)
"golang.org/x/oauth2/google"
"google.golang.org/api/gmail/v1"

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

### Removed External Dependencies
```go
// Removed in favor of native alternatives:
// "github.com/PuerkitoBio/goquery" -> net/html
// "github.com/bmaupin/go-epub" -> archive/zip + encoding/xml
// "github.com/spf13/cobra" -> manual flag parsing or custom CLI
// "github.com/spf13/viper" -> encoding/json + os
// "github.com/schollz/progressbar/v3" -> bubbles/progress
// "github.com/sirupsen/logrus" -> native log package
// "github.com/stretchr/testify" -> native testing

// Removed due to API token authentication:
// "net/http/cookiejar" -> no cookies needed
// "database/sql" -> no browser cookie extraction needed
// Cookie management packages -> API tokens don't expire like sessions
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

# Gmail Configuration
KOREILLY_GMAIL_EMAIL=""
KOREILLY_KINDLE_EMAIL=""
KOREILLY_SEND_TO_KINDLE="false"
KOREILLY_GMAIL_CREDENTIALS="./credentials.json"
KOREILLY_GMAIL_TOKEN="./token.json"
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
  "gmail_config": {
    "gmail_email": "",
    "kindle_email": "",
    "send_to_kindle": false,
    "email_subject": "{{.Title}} - O'Reilly Book",
    "token_file": "./token.json",
    "credentials_file": "./credentials.json"
  },
  "ui": {
    "theme": "default",
    "show_help": true,
    "auto_save_settings": true
  }
}
```

## TUI User Experience

### Main Interface Flow
```
┌─────────────────────────────────────────────────────────────┐
│                      KOReilly v1.0                         │
├─────────────────────────────────────────────────────────────┤
│  [T]oken [S]earch [D]ownload [G]mail Setup [Q]uit          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Status: ● Connected to O'Reilly Learning                  │
│  API Token: ●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●●abcd        │
│  Output: ./books/                                           │
│  Gmail: 📧 user@gmail.com → your-username@kindle.com       │
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

## Testing Strategy

### Unit Tests (Native testing package)
```go
func TestConfig_Load(t *testing.T)
func TestClient_DoWithRetry(t *testing.T)
func TestEPUBBuilder_Create(t *testing.T)
func TestHTMLProcessor_SanitizeHTML(t *testing.T)
func TestGmailService_OAuth2(t *testing.T)
func TestGmailService_SendToKindle(t *testing.T)
func TestAuthService_ValidateToken(t *testing.T)
```

### Integration Tests
```go
func TestEndToEndDownload(t *testing.T)
func TestAPITokenAuthenticationFlow(t *testing.T)
func TestEPUBValidation(t *testing.T)
func TestGmailDeliveryFlow(t *testing.T)
func TestKindleEmailDelivery(t *testing.T)
```

### TUI Tests
```go
func TestAuthModel_Update(t *testing.T)
func TestDownloadModel_ProgressUpdate(t *testing.T)
func TestSearchModel_Navigation(t *testing.T)
func TestGmailModel_Configuration(t *testing.T)
func TestGmailModel_TestConnection(t *testing.T)
```

## Security Considerations

1. **API Token Management**
   - Secure API token storage using native `os` permissions (0600)
   - Environment variable support for CI/CD environments
   - No plaintext token storage in logs or error messages
   - Automatic token masking in UI displays
   - Token validation against O'Reilly API on startup

2. **Gmail OAuth2 Security**
   - OAuth2 flow with Google's secure authorization
   - Token refresh handling using Google's libraries
   - Secure credential storage with file permissions
   - No password storage required
   - TLS encryption for all Gmail API calls

3. **Network Security**
   - HTTPS-only communication with O'Reilly Learning API
   - Bearer token authentication in request headers
   - Rate limiting to prevent API abuse
   - Proxy support for corporate environments
   - Certificate validation for all connections

4. **File System Security**
   - Configuration files with restricted permissions (0600)
   - EPUB files created with appropriate permissions
   - No temporary credential files
   - Secure cleanup of sensitive data on exit

5. **Token Security Best Practices**
   - No hardcoded API tokens in source code
   - Token expiration checking and user notification
   - Secure token generation recommendations for users
   - Clear token revocation instructions in documentation

## Email Service Features

### Simplified Dependencies

```go
// Gmail-specific dependencies (minimal)
"golang.org/x/oauth2"
"golang.org/x/oauth2/google"
"google.golang.org/api/gmail/v1"

// Native Go libraries (no change)
"net/http"
"encoding/json"
"encoding/base64"
"mime"
"mime/multipart"
"io"
"os"
"path/filepath"
// ... other native libraries
```

### Gmail-Only Benefits

#### **Simplified User Experience**
- **One provider to support** = less confusion
- **No provider selection** = faster setup
- **Gmail OAuth2** = most secure method
- **Largest user base** = covers most users

#### **Reduced Code Complexity**
- **No provider detection logic**
- **No SMTP templates**
- **No manual configuration**
- **Single authentication flow**

#### **Better User Support**
- **One set of instructions**
- **Well-documented Gmail API**
- **Consistent behavior**
- **Google's reliable infrastructure**

### Implementation Timeline (Updated)

#### **Week 1**: Core infrastructure with API token authentication and basic TUI
- Project setup with minimal dependencies
- API token authentication system
- Bearer token HTTP client
- Basic TUI interface with token input

#### **Week 2**: Book discovery with native HTML parsing, search TUI, and Gmail OAuth2 setup
- O'Reilly API integration with token auth
- Book search and metadata retrieval
- Gmail OAuth2 setup and authentication
- TUI search interface

#### **Week 3**: Content processing, native EPUB generation, and download progress interface
- Chapter and asset downloading
- Native EPUB generation using archive/zip
- Download progress interface
- Error handling and retry logic

#### **Week 4**: Gmail integration for Kindle delivery, TUI polish, and testing
- Gmail API integration for Kindle delivery
- Email composition with EPUB attachments
- TUI polish and user experience improvements
- Testing and documentation

### Quick Setup Guide for Users (Updated)

```markdown
# KOReilly Setup Guide

## Prerequisites
- O'Reilly Learning subscription
- Gmail account (for Kindle delivery)

## Setup Steps

### 1. Get your O'Reilly API token:
- Go to learning.oreilly.com
- Sign in and visit Profile → API Keys
- Create a new API key and copy it

### 2. Run KOReilly:
```bash
koreilly
```

### 3. Enter your API token when prompted

### 4. (Optional) Set up Gmail for Kindle delivery:

**Option A: Full Desktop Environment**
- Press 'G' for Gmail setup
- Follow OAuth2 authorization (browser opens automatically)
- Enter your Kindle email address

**Option B: SSH/Remote/Headless Environment**
- Press 'G' for Gmail setup
- Choose "Manual OAuth2"
- Copy the authorization URL
- Open URL on phone/another device
- Paste authorization code back in terminal

**Option C: Simple App Password (Fallback)**
- Press 'G' for Gmail setup
- Choose "App Password"
- Generate Gmail app password at myaccount.google.com
- Enter email and app password

### 5. Start downloading books!

**Setup Times:**
- API token: ~2 minutes
- Gmail OAuth2 (desktop): +2 minutes  
- Gmail OAuth2 (manual): +3 minutes
- Gmail App Password: +2 minutes
```

## Timeline Summary

- **Week 1**: Core infrastructure with API token authentication and basic TUI
- **Week 2**: Book discovery with native HTML parsing, search TUI, and Gmail OAuth2 setup
- **Week 3**: Content processing, native EPUB generation, and download progress interface
- **Week 4**: Gmail integration for Kindle delivery, TUI polish, and testing

Total estimated development time: **4 weeks** for MVP with minimal dependencies, beautiful TUI interface, and Gmail-only Kindle delivery.
