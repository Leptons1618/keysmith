package core

import "strings"

// Service describes one Git host the wizard can set up.
type Service struct {
	ID      string // short id: github, gitlab, bitbucket
	Name    string // display name
	Host    string // ssh host for git access
	AltHost string // port-443 fallback hostname ("" = none)
	AltPort string
	KeysURL string // browser URL of the SSH keys settings page
	Steps   []string
}

// Services is the registry the wizard offers, in display order.
var Services = []Service{
	{
		ID:      "github",
		Name:    "GitHub",
		Host:    "github.com",
		AltHost: "ssh.github.com",
		AltPort: "443",
		KeysURL: "https://github.com/settings/keys",
		Steps: []string{
			"1. Sign in to github.com in your browser.",
			"2. Open Settings, then \"SSH and GPG keys\" (the button below opens it for you).",
			"3. Click \"New SSH key\".",
			"4. Title: anything you like, e.g. \"" + "work laptop" + "\".",
			"5. Key type stays \"Authentication Key\".",
			"6. Paste the public key from your clipboard into the \"Key\" field.",
			"7. Click \"Add SSH key\" and confirm if prompted.",
			"8. Come back here and run the connection test.",
		},
	},
	{
		ID:      "gitlab",
		Name:    "GitLab",
		Host:    "gitlab.com",
		AltHost: "altssh.gitlab.com",
		AltPort: "443",
		KeysURL: "https://gitlab.com/-/user_settings/ssh_keys",
		Steps: []string{
			"1. Sign in to gitlab.com in your browser.",
			"2. Click your avatar, then \"Edit profile\" / Preferences.",
			"3. Open \"SSH Keys\" in the left menu (the button below opens it for you).",
			"4. Click \"Add new key\".",
			"5. Paste the public key from your clipboard into the \"Key\" field.",
			"6. Title: anything you like; usage stays \"Authentication & Signing\" or set \"Authentication\".",
			"7. Click \"Add key\".",
			"8. Come back here and run the connection test.",
		},
	},
	{
		ID:      "bitbucket",
		Name:    "Bitbucket",
		Host:    "bitbucket.org",
		AltHost: "altssh.bitbucket.org",
		AltPort: "443",
		KeysURL: "https://bitbucket.org/account/settings/ssh-keys/",
		Steps: []string{
			"1. Sign in to bitbucket.org in your browser.",
			"2. Click the settings gear, then \"Personal Bitbucket settings\".",
			"3. Under SECURITY find \"SSH keys\" (the button below opens it for you).",
			"4. Click \"Add key\".",
			"5. Label: anything you like.",
			"6. Paste the public key from your clipboard into the \"Key\" field.",
			"7. Click \"Add key\" / Save.",
			"8. Come back here and run the connection test.",
		},
	},
}

// ServiceByID looks a service up by its short id; ok is false when unknown.
func ServiceByID(id string) (Service, bool) {
	for _, s := range Services {
		if s.ID == id {
			return s, true
		}
	}
	return Service{}, false
}

// InstructionText renders the numbered steps as one plain-text block,
// identical for both frontends.
func (s Service) InstructionText() string {
	var b strings.Builder
	b.WriteString("How to add your key to " + s.Name + ":")
	for _, step := range s.Steps {
		b.WriteString("\n  ")
		b.WriteString(step)
	}
	return b.String()
}
