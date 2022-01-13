package config

import "fmt"

//go:generate packer-sdc mapstructure-to-hcl2 -type Provisioner

// Provisioner is our provisioner configuration.
type Provisioner struct {
	Version        string
	User           string
	DbusVersion    string
	DubsX11Version string
	SSHPubEntry    string
}

// Default inputs default values.
func (p *Provisioner) Defaults() {
	if p.Version == "" {
		p.Version = "latest"
	}
	if p.User == "" {
		p.User = "agent"
	}
	if p.DbusVersion == "" {
		p.DbusVersion = "latest"
	}
	if p.DubsX11Version == "" {
		p.DubsX11Version = "latest"
	}
}

// Validate validates the config looks correct.
func (p *Provisioner) Validate() error {
	if p.SSHPubEntry == "" {
		return fmt.Errorf("SSHPubEntry must exist")
	}
	return nil
}
