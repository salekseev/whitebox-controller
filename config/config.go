package config

import (
	"errors"
	"fmt"
	"github.com/summerwind/whitebox-controller/handler"
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
	Metrics     *MetricsConfig      `json:"metrics,omitempty"`
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

	if c.Metrics != nil {
		err := c.Metrics.Validate()
		if err != nil {
			return fmt.Errorf("metrics: %v", err)
		}
	}

	return nil
}

type ControllerConfig struct {
	Name       string
	Resource   schema.GroupVersionKind `json:"resource"`
	Dependents []DependentConfig       `json:"dependents"`
	References []ReferenceConfig       `json:"references"`
	Reconciler *ReconcilerConfig       `json:"reconciler,omitempty"`
	Finalizer  *HandlerConfig          `json:"finalizer,omitempty"`
	Syncer     *SyncerConfig           `json:"syncer,omitempty"`
}

func (c *ControllerConfig) Validate() error {
	if c.Name == "" {
		return errors.New("name must be specified")
	}

	if c.Resource.Empty() {
		return errors.New("resource is empty")
	}

	for i, ref := range c.References {
		err := ref.Validate()
		if err != nil {
			return fmt.Errorf("references[%d]: %v", i, err)
		}
	}

	for i, dep := range c.Dependents {
		if dep.Empty() {
			return fmt.Errorf("dependents[%d] is empty", i)
		}
	}

	if c.Reconciler == nil {
		return errors.New("reconciler must be specified")
	}

	err := c.Reconciler.Validate()
	if err != nil {
		return fmt.Errorf("reconciler: %v", err)
	}

	if c.Finalizer != nil {
		err := c.Finalizer.Validate()
		if err != nil {
			return fmt.Errorf("finalizer: %v", err)
		}
	}

	if c.Syncer != nil {
		err := c.Syncer.Validate()
		if err != nil {
			return fmt.Errorf("syncer: %v", err)
		}
	}

	return nil
}

type DependentConfig struct {
	schema.GroupVersionKind
	Orphan bool `json:"orphan"`
}

func (c *DependentConfig) Validate() error {
	if c.GroupVersionKind.Empty() {
		return errors.New("resource is empty")
	}

	return nil
}

type ReferenceConfig struct {
	schema.GroupVersionKind
	NameFieldPath string `json:"nameFieldPath"`
}

func (c *ReferenceConfig) Validate() error {
	if c.GroupVersionKind.Empty() {
		return errors.New("resource is empty")
	}

	if c.NameFieldPath == "" {
		return errors.New("nameFieldPath must be specified")
	}

	return nil
}

type ReconcilerConfig struct {
	HandlerConfig
	RequeueAfter string `json:"requeueAfter"`
	Observe      bool   `json:"observe"`
}

func (c *ReconcilerConfig) Validate() error {
	if c.RequeueAfter != "" {
		_, err := time.ParseDuration(c.RequeueAfter)
		if err != nil {
			return fmt.Errorf("invalid requeueAfter: %v", err)
		}
	}

	return c.HandlerConfig.Validate()
}

type HandlerConfig struct {
	Exec *ExecHandlerConfig `json:"exec"`
	HTTP *HTTPHandlerConfig `json:"http"`
	Func *FuncHandlerConfig `json:"-"`
}

func (c *HandlerConfig) Validate() error {
	specified := 0
	if c.Exec != nil {
		specified++
	}
	if c.HTTP != nil {
		specified++
	}
	if c.Func != nil {
		specified++
	}

	if specified == 0 {
		return errors.New("handler must be specified")
	}
	if specified > 1 {
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

	if c.Func != nil {
		err := c.Func.Validate()
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
		return errors.New("cert file must be specified")
	}

	if c.KeyFile != "" {
		return errors.New("key file must be specified")
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
	return nil
}

type WebhookHandlerConfig struct {
	Resource  schema.GroupVersionKind `json:"resource"`
	Validator *HandlerConfig          `json:"validator"`
	Mutator   *HandlerConfig          `json:"mutator"`
	Injector  *InjectorConfig         `json:"injector"`
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

type InjectorConfig struct {
	HandlerConfig
	VerifyKeyFile string `json:"verifyKeyFile"`
}

func (c *InjectorConfig) Validate() error {
	_, err := os.Stat(c.VerifyKeyFile)
	if err != nil {
		return fmt.Errorf("failed to read verification key file: %v", err)
	}

	return c.HandlerConfig.Validate()
}

type MetricsConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func (c *MetricsConfig) Validate() error {
	if c.Port == 0 {
		return errors.New("port must be specified")
	}
	return nil
}

type FuncHandlerConfig struct {
	Handler handler.Handler `json:"-"`
}

func (c *FuncHandlerConfig) Validate() error {
	if c.Handler == nil {
		return errors.New("handler must be specified")
	}

	return nil
}
