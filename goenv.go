package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/johnsiilver/packer-plugin-goenv/internal/config"

	"github.com/gopherfs/fs/io/mem/simple"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/plugin"
	packerConfig "github.com/hashicorp/packer-plugin-sdk/template/config"
)

func main() {
	set := plugin.NewSet()

	set.RegisterProvisioner("goenv", &Provisioner{})
	err := set.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

// Provisioner implements packer.Provisioner.
type Provisioner struct {
	packer.Provisioner // Embed the interface.

	conf     *config.Provisioner
	content  []byte
	fileName string
}

func (p *Provisioner) ConfigSpec() hcldec.ObjectSpec {
	return new(config.FlatProvisioner).HCL2Spec()
}

func (p *Provisioner) Prepare(raws ...interface{}) error {
	c := config.Provisioner{}
	if err := packerConfig.Decode(&c, nil, raws...); err != nil {
		return err
	}
	c.Defaults()
	return nil
}

func (p *Provisioner) Provision(ctx context.Context, u packer.Ui, c packer.Communicator, m map[string]interface{}) error {
	u.Message("Being Go environment install")
	if err := p.fetch(ctx, u, c); err != nil {
		return err
	}
	if err := p.push(ctx, u, c); err != nil {
		return err
	}
	if err := p.unpack(ctx, u, c); err != nil {
		return err
	}
	if err := p.test(ctx, u, c); err != nil {
		return err
	}
	u.Message("Go environment install finished")
	return nil
}

func (p *Provisioner) fetch(ctx context.Context, u packer.Ui, c packer.Communicator) error {
	const (
		goURL = `https://golang.org/dl/go%s.linux-amd64.tar.gz`
		name  = `go%s.linux-amd64.tar.gz`
	)

	u.Message("Determining latest Go version")
	if p.conf.Version != "latest" {
		resp, err := http.Get("https://golang.org/VERSION?m=text")
		if err != nil {
			return fmt.Errorf("problem asking Google for latest Go version: %w", err)
		}
		ver, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("problem reading latest Go version: %w", err)
		}

		p.conf.Version = strings.TrimPrefix(string(ver), "go")
		u.Message("Latest Go version: " + p.conf.Version)
	}

	url := fmt.Sprintf(goURL, p.conf.Version)

	u.Message("Downloading Go version: " + url)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("problem reaching golang.org for version(%s): %w)", p.conf.Version, err)
	}
	defer resp.Body.Close()

	p.content, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("problem downloading file: %w", err)
	}
	p.fileName = fmt.Sprintf(name, p.conf.Version)
	u.Message("Downloading complete")

	return nil
}

func (p *Provisioner) push(ctx context.Context, u packer.Ui, c packer.Communicator) error {
	u.Message("Pushing Go tarball")

	fs := simple.New()
	fs.WriteFile("/tarball", p.content, 0700)
	fi, _ := fs.Stat("/tarball")

	err := c.Upload(
		"/tmp/"+p.fileName,
		bytes.NewReader(p.content),
		&fi,
	)
	if err != nil {
		return err
	}
	u.Message("Go tarball delivered to: /tmp/" + p.fileName)
	return nil
}

func (p *Provisioner) unpack(ctx context.Context, u packer.Ui, c packer.Communicator) error {
	const cmd = `sudo tar -C /usr/local -xzf /tmp/%s`
	u.Message("Unpacking Go tarball to /usr/local")

	b := bytes.Buffer{}
	rc := &packer.RemoteCmd{
		Command: fmt.Sprintf(cmd, p.fileName),
		Stdout:  &b,
		Stderr:  &b,
	}

	if err := c.Start(ctx, rc); err != nil {
		return fmt.Errorf("problem unpacking tarball(%s):\n%s", err, b.String())
	}
	u.Message("Unpacked Go tarball")
	return nil
}

func (p *Provisioner) test(ctx context.Context, u packer.Ui, c packer.Communicator) error {
	u.Message("Testing Go install")

	b := bytes.Buffer{}
	rc := &packer.RemoteCmd{
		Command: `/usr/local/go/bin/go version`,
		Stdout:  &b,
		Stderr:  &b,
	}
	if err := c.Start(ctx, rc); err != nil {
		return fmt.Errorf("problem testing Go install(%s):\n%s", err, b.String())
	}
	u.Message("Go installed successfully")
	return nil
}