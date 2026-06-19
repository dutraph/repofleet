// Package config persists repos' settings: the directories to scan for
// local repositories, and one or more git-server accounts (GitHub,
// GitLab, Azure DevOps, Bitbucket) each holding a PAT used to browse and
// clone remote repos.
//
// The file lives at $XDG_CONFIG_HOME/fleet/config.yaml (or the macOS
// equivalent) with 0600 permissions. Multi-account is the default
// shape; a legacy single-account file is migrated on first Load().
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Provider identifiers as stored in YAML.
const (
	ProviderGitHub    = "github"
	ProviderGitLab    = "gitlab"
	ProviderAzure     = "azure"
	ProviderBitbucket = "bitbucket"
)

// Account is a single git-server connection.
type Account struct {
	Name     string `yaml:"name"`               // unique label
	Provider string `yaml:"provider"`           // one of the Provider* consts
	PAT      string `yaml:"pat,omitempty"`      // personal access token
	BaseURL  string `yaml:"base_url,omitempty"` // self-hosted GitLab/GHE base; blank = public
	Org      string `yaml:"org,omitempty"`      // Azure DevOps organization
	Username string `yaml:"username,omitempty"` // Bitbucket username (for app passwords)
}

// Config is the on-disk representation.
type Config struct {
	ScanRoots []string  `yaml:"scan_roots,omitempty"`
	Accounts  []Account `yaml:"accounts,omitempty"`
	Active    string    `yaml:"active,omitempty"`
}

func Path() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "fleet", "config.yaml"), nil
}

// Load reads the config, applying sensible defaults so the TUI can run
// for local scanning even before any account is added.
func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	var cfg Config
	data, err := os.ReadFile(p)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		// First run: no file yet. That's fine — default to scanning home.
	} else if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if len(cfg.ScanRoots) == 0 {
		if home, err := os.UserHomeDir(); err == nil {
			cfg.ScanRoots = []string{home}
		}
	}
	if cfg.Active != "" && !cfg.has(cfg.Active) && len(cfg.Accounts) > 0 {
		cfg.Active = cfg.Accounts[0].Name
	}
	if cfg.Active == "" && len(cfg.Accounts) > 0 {
		cfg.Active = cfg.Accounts[0].Name
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

func (c *Config) has(name string) bool {
	for _, a := range c.Accounts {
		if a.Name == name {
			return true
		}
	}
	return false
}

// ActiveAccount returns the active account, or nil when none configured.
func (c *Config) ActiveAccount() *Account {
	for i := range c.Accounts {
		if c.Accounts[i].Name == c.Active {
			return &c.Accounts[i]
		}
	}
	if len(c.Accounts) > 0 {
		return &c.Accounts[0]
	}
	return nil
}

// SetActive changes the active account by name.
func (c *Config) SetActive(name string) bool {
	if !c.has(name) {
		return false
	}
	c.Active = name
	return true
}

// UpsertAccount inserts or updates an account by name, preserving the
// existing PAT when the incoming one is blank.
func (c *Config) UpsertAccount(a Account) {
	a.Name = strings.TrimSpace(a.Name)
	if a.Name == "" {
		return
	}
	for i := range c.Accounts {
		if c.Accounts[i].Name == a.Name {
			if a.PAT == "" {
				a.PAT = c.Accounts[i].PAT
			}
			c.Accounts[i] = a
			return
		}
	}
	c.Accounts = append(c.Accounts, a)
	if c.Active == "" {
		c.Active = a.Name
	}
}

// ---- CLI helpers (invoked from cmd/fleet/main.go) ----

func ListAccounts() error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	if len(cfg.Accounts) == 0 {
		fmt.Println("No accounts configured. Run `fleet login` to add one.")
		return nil
	}
	for _, a := range cfg.Accounts {
		mark := "  "
		if a.Name == cfg.Active {
			mark = "★ "
		}
		fmt.Printf("%s%-20s %s\n", mark, a.Name, a.Provider)
	}
	return nil
}

func SwitchAccount(name string) error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	if !cfg.SetActive(name) {
		names := make([]string, 0, len(cfg.Accounts))
		for _, a := range cfg.Accounts {
			names = append(names, a.Name)
		}
		return fmt.Errorf("not configured: %q. Available: %s", name, strings.Join(names, ", "))
	}
	if err := Save(cfg); err != nil {
		return err
	}
	fmt.Printf("Active account is now %s\n", name)
	return nil
}

// RunLoginWizard adds or updates a git-server account interactively.
func RunLoginWizard() error {
	reader := bufio.NewReader(os.Stdin)
	cfg, err := Load()
	if err != nil {
		return err
	}

	fmt.Println("fleet login")
	fmt.Println("───────────")
	if len(cfg.Accounts) > 0 {
		fmt.Println("Configured accounts:")
		for _, a := range cfg.Accounts {
			mark := "  "
			if a.Name == cfg.Active {
				mark = "★ "
			}
			fmt.Printf("  %s%-20s %s\n", mark, a.Name, a.Provider)
		}
		fmt.Println()
	}

	prov := ""
	for prov == "" {
		in := strings.ToLower(prompt(reader, "Provider [github/gitlab/azure/bitbucket]", "github"))
		switch in {
		case "github", "gh":
			prov = ProviderGitHub
		case "gitlab", "gl":
			prov = ProviderGitLab
		case "azure", "azuredevops", "ado":
			prov = ProviderAzure
		case "bitbucket", "bb":
			prov = ProviderBitbucket
		default:
			fmt.Println("  please pick one of github/gitlab/azure/bitbucket")
		}
	}

	name := prompt(reader, "Account label", prov)
	acct := Account{Name: name, Provider: prov}
	if existing := find(cfg.Accounts, name); existing != nil {
		acct = *existing
		acct.Provider = prov
	}

	switch prov {
	case ProviderAzure:
		acct.Org = prompt(reader, "Azure DevOps organization", acct.Org)
		if acct.Org == "" {
			return errors.New("Azure DevOps requires an organization")
		}
	case ProviderBitbucket:
		acct.Username = prompt(reader, "Bitbucket username", acct.Username)
		if acct.Username == "" {
			return errors.New("Bitbucket requires a username (used with the app password)")
		}
	case ProviderGitLab:
		acct.BaseURL = prompt(reader, "Base URL (blank for gitlab.com)", acct.BaseURL)
	case ProviderGitHub:
		acct.BaseURL = prompt(reader, "Base URL (blank for github.com)", acct.BaseURL)
	}

	secretLabel := "Personal Access Token"
	if prov == ProviderBitbucket {
		secretLabel = "App password"
	}
	pat := prompt(reader, secretLabel, maskedHint(acct.PAT))
	if pat != maskedHint(acct.PAT) && pat != "" {
		acct.PAT = pat
	}
	if acct.PAT == "" {
		return errors.New("a token is required")
	}

	cfg.UpsertAccount(acct)
	if cfg.Active == "" || strings.ToLower(prompt(reader, fmt.Sprintf("Make %q the active account? [Y/n]", name), "Y")) != "n" {
		cfg.Active = name
	}

	if err := Save(cfg); err != nil {
		return err
	}
	p, _ := Path()
	fmt.Printf("\nSaved to %s (active: %s)\n", p, cfg.Active)
	return nil
}

func find(accts []Account, name string) *Account {
	for i := range accts {
		if accts[i].Name == name {
			return &accts[i]
		}
	}
	return nil
}

func prompt(r *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := r.ReadString('\n')
	v := strings.TrimSpace(line)
	if v == "" {
		return def
	}
	return v
}

func maskedHint(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "****"
	}
	return strings.Repeat("*", 4) + s[len(s)-4:]
}
