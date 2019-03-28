package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type HandlerType string

const (
	HandlerTypeExec HandlerType = "exec"
)

type Config struct {
	Controllers []*ControllerConfig `json:"controllers"`
	Webhook     *WebhookConfig      `json:"webhook,omitempty"`
}

func LoadFile(p string) (*Config, error) {
	buf, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}

	c := &Config{}
	err = yaml.Unmarshal(buf, c)
	if err != nil {
		return nil, err
	}

	err = c.Validate()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Config) Validate() error {
	for i, controller := range c.Controllers {
		err := controller.Validate()
		if err != nil {
			return fmt.Errorf("controller[%d]: %v", i, err)
		}
	}

	if c.Webhook != nil {
		err := c.Webhook.Validate()
		if err != nil {
			return fmt.Errorf("webhook: %v", err)
		}
	}

	return nil
}

type ControllerConfig struct {
	Name       string
	Resource   schema.GroupVersionKind   `json:"resource"`
	Dependents []schema.GroupVersionKind `json:"dependents"`
	Reconciler *HandlerConfig            `json:"reconciler,omitempty"`
	Observer   *HandlerConfig            `json:"observer,omitempty"`
	Syncer     *SyncerConfig             `json:"syncer,omitempty"`
}

func (c *ControllerConfig) Validate() error {
	if c.Name == "" {
		return errors.New("name must be specified")
	}

	if c.Resource.Empty() {
		return errors.New("resource is empty")
	}

	if c.Reconciler == nil && c.Observer == nil {
		return errors.New("either reconciler or observer must be specified")
	}
	if c.Reconciler != nil && c.Observer != nil {
		return errors.New("exactly one of reconciler or observer must be specified")
	}

	if c.Reconciler != nil {
		err := c.Reconciler.Validate()
		if err != nil {
			return fmt.Errorf("reconciler: %v", err)
		}

		for i, dep := range c.Dependents {
			if dep.Empty() {
				return fmt.Errorf("dependents[%d] is empty", i)
			}
		}

		if c.Syncer != nil {
			err := c.Syncer.Validate()
			if err != nil {
				return fmt.Errorf("syncer: %v", err)
			}
		}
	}

	if c.Observer != nil {
		err := c.Observer.Validate()
		if err != nil {
			return fmt.Errorf("observer: %v", err)
		}
	}

	return nil
}

type HandlerConfig struct {
	Exec *ExecHandlerConfig `json:"exec"`
	HTTP *HTTPHandlerConfig `json:"http"`
}

func (c *HandlerConfig) Validate() error {
	if c.Exec == nil && c.HTTP == nil {
		return errors.New("handler must be specified")
	}
	if c.Exec != nil && c.HTTP != nil {
		return errors.New("exactly one handler must be specified")
	}

	if c.Exec != nil {
		err := c.Exec.Validate()
		if err != nil {
			return err
		}
	}

	if c.HTTP != nil {
		err := c.HTTP.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

type ExecHandlerConfig struct {
	Command    string            `json:"command"`
	Args       []string          `json:"args"`
	WorkingDir string            `json:"workingDir"`
	Env        map[string]string `json:"env"`
	Timeout    string            `json:"timeout"`
	Debug      bool              `json:"debug"`
}

func (c ExecHandlerConfig) Validate() error {
	if c.Command == "" {
		return errors.New("command must be specified")
	}

	if c.Timeout != "" {
		_, err := time.ParseDuration(c.Timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout: %v", err)
		}
	}

	return nil
}

type HTTPHandlerConfig struct {
	URL     string                `json:"url"`
	TLS     *HTTPHanlderTLSConfig `json:"tls,omitempty"`
	Timeout string                `json:"timeout"`
	Debug   bool                  `json:"debug"`
}

func (c HTTPHandlerConfig) Validate() error {
	if c.URL == "" {
		return errors.New("url must be specified")
	}

	if c.TLS != nil {
		err := c.TLS.Validate()
		if err != nil {
			return fmt.Errorf("tls: %v", err)
		}
	}

	if c.Timeout != "" {
		_, err := time.ParseDuration(c.Timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout: %v", err)
		}
	}

	return nil
}

type HTTPHanlderTLSConfig struct {
	CertFile   string `json:"certFile"`
	KeyFile    string `json:"keyFile"`
	CACertFile string `json:"caCertFile"`
}

func (c *HTTPHanlderTLSConfig) Validate() error {
	if c.CertFile != "" {
		_, err := os.Stat(c.CertFile)
		if err != nil {
			return fmt.Errorf("failed to read cert file: %v", err)
		}
	}

	if c.KeyFile != "" {
		_, err := os.Stat(c.KeyFile)
		if err != nil {
			return fmt.Errorf("failed to read key file: %v", err)
		}
	}

	if c.CACertFile != "" {
		_, err := os.Stat(c.CACertFile)
		if err != nil {
			return fmt.Errorf("failed to read ca cert file: %v", err)
		}
	}

	return nil
}

type SyncerConfig struct {
	Interval string `json:"interval"`
}

func (c SyncerConfig) Validate() error {
	if c.Interval != "" {
		_, err := time.ParseDuration(c.Interval)
		if err != nil {
			return fmt.Errorf("invalid interval: %v", err)
		}
	}

	return nil
}

type WebhookConfig struct {
	Host     string                  `json:"host"`
	Port     int                     `json:"port"`
	TLS      *WebhookTLSConfig       `json:"tls"`
	Handlers []*WebhookHandlerConfig `json:"handlers"`
}

func (c *WebhookConfig) Validate() error {
	if c.TLS == nil {
		return errors.New("tls must be specified")
	}

	err := c.TLS.Validate()
	if err != nil {
		return fmt.Errorf("tls: %v", err)
	}

	return nil
}

type WebhookTLSConfig struct {
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
}

func (c *WebhookTLSConfig) Validate() error {
	if c.CertFile == "" {
		return errors.New("cert file must be specified")
	}
	if c.KeyFile == "" {
		return errors.New("key file must be specified")
	}

	_, err := os.Stat(c.CertFile)
	if err != nil {
		return fmt.Errorf("failed to read cert file: %v", err)
	}
	_, err = os.Stat(c.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to read key file: %v", err)
	}

	return nil
}

type WebhookHandlerConfig struct {
	Resource  schema.GroupVersionKind `json:"resource"`
	Validator *HandlerConfig          `json:"validator"`
	Mutator   *HandlerConfig          `json:"mutator"`
}

func (c *WebhookHandlerConfig) Validate() error {
	if c.Resource.Empty() {
		return errors.New("resource is empty")
	}

	if c.Validator != nil {
		err := c.Validator.Validate()
		if err != nil {
			return fmt.Errorf("validator: %v", err)
		}
	}

	return nil
}